// Package metadata enrichit le catalogue depuis TMDB (optionnel : clé API fournie par l'administrateur).
// Conditions TMDB : attribution obligatoire (« This product uses the TMDB API but is not endorsed or
// certified by TMDB »), affichée dans « À propos ». Les images sont mises en cache localement pour
// fonctionner sans Internet.
package metadata

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Client TMDB minimal.
type Client struct {
	APIKey   string // clé v3 ou jeton v4 (« eyJ… »)
	Language string // fr-FR par défaut
	ImageDir string
	HTTP     *http.Client
	BaseURL  string // surchargé dans les tests
	ImageURL string

	pacer *pacer // partagé entre les copies (WithLanguage) : une seule cadence vers TMDB
}

type pacer struct {
	mu   sync.Mutex
	last time.Time
}

// NewClient crée un client.
func NewClient(apiKey, language, imageDir string) *Client {
	if language == "" {
		language = "fr-FR"
	}
	return &Client{APIKey: strings.TrimSpace(apiKey), Language: language, ImageDir: imageDir,
		HTTP: &http.Client{Timeout: 20 * time.Second}, BaseURL: "https://api.themoviedb.org/3",
		ImageURL: "https://image.tmdb.org/t/p", pacer: &pacer{}}
}

// WithLanguage renvoie une copie dans une autre langue, qui partage connexion et cadence.
func (c *Client) WithLanguage(lang string) *Client {
	cp := *c
	cp.Language = lang
	return &cp
}

// ErrNoKey : aucune clé configurée.
var ErrNoKey = errors.New("aucune clé TMDB configurée")

// pace : au plus ~8 requêtes/s, bien sous la limite TMDB, pour rester discret.
func (c *Client) pace() {
	c.pacer.mu.Lock()
	defer c.pacer.mu.Unlock()
	if d := 125*time.Millisecond - time.Since(c.pacer.last); d > 0 {
		time.Sleep(d)
	}
	c.pacer.last = time.Now()
}

func (c *Client) get(ctx context.Context, path string, q url.Values, out any) error {
	if c.APIKey == "" {
		return ErrNoKey
	}
	if q == nil {
		q = url.Values{}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return err
	}
	if strings.HasPrefix(c.APIKey, "eyJ") {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	} else {
		q.Set("api_key", c.APIKey)
	}
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Accept", "application/json")
	for attempt := 0; ; attempt++ {
		c.pace()
		resp, err := c.HTTP.Do(req)
		if err != nil {
			return fmt.Errorf("TMDB injoignable : %w", stripURL(err))
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		resp.Body.Close()
		if err != nil {
			return err
		}
		switch {
		case resp.StatusCode == http.StatusTooManyRequests && attempt < 3:
			time.Sleep(time.Duration(2<<attempt) * time.Second)
			continue
		case resp.StatusCode == http.StatusUnauthorized:
			return errors.New("clé TMDB refusée (401)")
		case resp.StatusCode == http.StatusNotFound:
			return ErrNotFound
		case resp.StatusCode != http.StatusOK:
			return fmt.Errorf("TMDB : statut %d", resp.StatusCode)
		}
		return json.Unmarshal(body, out)
	}
}

// stripURL retire l'URL d'une erreur réseau : avec une clé v3, elle contient « api_key=… » et finirait
// dans le journal, la base (tâches) et l'interface.
func stripURL(err error) error {
	var ue *url.Error
	if errors.As(err, &ue) {
		return ue.Err
	}
	return err
}

// ErrNotFound : ressource TMDB introuvable.
var ErrNotFound = errors.New("introuvable sur TMDB")

// Validate vérifie la clé.
func (c *Client) Validate(ctx context.Context) error {
	var out map[string]any
	return c.get(ctx, "/configuration", nil, &out)
}

// SearchResult : candidat de correspondance.
type SearchResult struct {
	ID            int     `json:"id"`
	Title         string  `json:"title"`
	OriginalTitle string  `json:"originalTitle"`
	Year          int     `json:"year"`
	Overview      string  `json:"overview"`
	Poster        string  `json:"poster"` // URL TMDB (aperçu dans l'interface de correction)
	Popularity    float64 `json:"popularity"`
}

type searchResp struct {
	Results []struct {
		ID            int     `json:"id"`
		Title         string  `json:"title"`
		Name          string  `json:"name"`
		OriginalTitle string  `json:"original_title"`
		OriginalName  string  `json:"original_name"`
		ReleaseDate   string  `json:"release_date"`
		FirstAirDate  string  `json:"first_air_date"`
		Overview      string  `json:"overview"`
		PosterPath    string  `json:"poster_path"`
		Popularity    float64 `json:"popularity"`
	} `json:"results"`
}

// Search cherche un film (kind="movie") ou une série (kind="tv").
func (c *Client) Search(ctx context.Context, kind, query string, year int) ([]SearchResult, error) {
	q := url.Values{"query": {query}, "language": {c.Language}, "include_adult": {"false"}}
	if year > 0 {
		if kind == "movie" {
			q.Set("year", fmt.Sprint(year))
		} else {
			q.Set("first_air_date_year", fmt.Sprint(year))
		}
	}
	var r searchResp
	if err := c.get(ctx, "/search/"+kind, q, &r); err != nil {
		return nil, err
	}
	out := make([]SearchResult, 0, len(r.Results))
	for _, x := range r.Results {
		s := SearchResult{ID: x.ID, Title: x.Title + x.Name, OriginalTitle: x.OriginalTitle + x.OriginalName,
			Year: yearOf(x.ReleaseDate + x.FirstAirDate), Overview: x.Overview, Popularity: x.Popularity}
		if x.PosterPath != "" {
			s.Poster = c.ImageURL + "/w185" + x.PosterPath
		}
		out = append(out, s)
	}
	return out, nil
}

func yearOf(date string) int {
	if len(date) < 4 {
		return 0
	}
	var y int
	fmt.Sscanf(date[:4], "%d", &y)
	return y
}

// Image télécharge (une fois) une image TMDB et renvoie son nom de cache local.
// size : w185 (portraits), w300 (vignettes d'épisode), w500 (affiches), w1280 (fonds).
func (c *Client) Image(ctx context.Context, size, path string) (string, error) {
	if path == "" {
		return "", nil
	}
	h := sha1.Sum([]byte(size + path))
	name := hex.EncodeToString(h[:]) + filepath.Ext(path)
	dst := filepath.Join(c.ImageDir, name)
	if st, err := os.Stat(dst); err == nil && st.Size() > 0 {
		return name, nil
	}
	if err := os.MkdirAll(c.ImageDir, 0o750); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.ImageURL+"/"+size+path, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", stripURL(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("image TMDB : statut %d", resp.StatusCode)
	}
	tmp := dst + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(f, io.LimitReader(resp.Body, 20<<20)); err != nil {
		f.Close()
		os.Remove(tmp)
		return "", err
	}
	f.Close()
	return name, os.Rename(tmp, dst)
}

// Credit : personne au générique.
type Credit struct {
	Name        string `json:"name"`
	Character   string `json:"character"`
	Job         string `json:"job"`
	ProfilePath string `json:"profile_path"`
	Order       int    `json:"order"`
}

// MovieDetails : fiche film.
type MovieDetails struct {
	ID            int     `json:"id"`
	Title         string  `json:"title"`
	OriginalTitle string  `json:"original_title"`
	Overview      string  `json:"overview"`
	Tagline       string  `json:"tagline"`
	ReleaseDate   string  `json:"release_date"`
	Runtime       int     `json:"runtime"`
	VoteAverage   float64 `json:"vote_average"`
	VoteCount     int     `json:"vote_count"`
	PosterPath    string  `json:"poster_path"`
	BackdropPath  string  `json:"backdrop_path"`
	Genres        []struct {
		Name string `json:"name"`
	} `json:"genres"`
	BelongsTo *struct {
		ID           int    `json:"id"`
		Name         string `json:"name"`
		PosterPath   string `json:"poster_path"`
		BackdropPath string `json:"backdrop_path"`
	} `json:"belongs_to_collection"`
	Credits struct {
		Cast []Credit `json:"cast"`
		Crew []Credit `json:"crew"`
	} `json:"credits"`
	ReleaseDates struct {
		Results []struct {
			Country string `json:"iso_3166_1"`
			Dates   []struct {
				Certification string `json:"certification"`
			} `json:"release_dates"`
		} `json:"results"`
	} `json:"release_dates"`
	Videos struct {
		Results []struct {
			Key      string `json:"key"`
			Site     string `json:"site"`
			Type     string `json:"type"`
			Official bool   `json:"official"`
		} `json:"results"`
	} `json:"videos"`
}

// Movie récupère une fiche film (langue configurée, synopsis en anglais en repli).
func (c *Client) Movie(ctx context.Context, id int) (*MovieDetails, error) {
	var d MovieDetails
	q := url.Values{"language": {c.Language}, "append_to_response": {"credits,release_dates,videos"},
		"include_video_language": {lang2(c.Language) + ",en,null"}}
	if err := c.get(ctx, fmt.Sprintf("/movie/%d", id), q, &d); err != nil {
		return nil, err
	}
	if d.Overview == "" && !strings.HasPrefix(c.Language, "en") {
		var en MovieDetails
		if c.get(ctx, fmt.Sprintf("/movie/%d", id), url.Values{"language": {"en-US"}}, &en) == nil {
			d.Overview = en.Overview
			if d.Tagline == "" {
				d.Tagline = en.Tagline
			}
		}
	}
	return &d, nil
}

// Certification renvoie la classification et l'âge minimal (France d'abord, États-Unis en repli).
func (d *MovieDetails) Certification() (string, *int) {
	byCountry := map[string]string{}
	for _, r := range d.ReleaseDates.Results {
		for _, x := range r.Dates {
			if x.Certification != "" {
				byCountry[r.Country] = x.Certification
				break
			}
		}
	}
	return certification(byCountry)
}

// Trailer renvoie la clé YouTube de la bande-annonce officielle (lien externe, jamais embarqué hors-ligne).
func (d *MovieDetails) Trailer() string {
	for _, v := range d.Videos.Results {
		if v.Site == "YouTube" && v.Type == "Trailer" && v.Official {
			return v.Key
		}
	}
	for _, v := range d.Videos.Results {
		if v.Site == "YouTube" && v.Type == "Trailer" {
			return v.Key
		}
	}
	return ""
}

// TVDetails : fiche série.
type TVDetails struct {
	ID             int     `json:"id"`
	Name           string  `json:"name"`
	OriginalName   string  `json:"original_name"`
	Overview       string  `json:"overview"`
	Tagline        string  `json:"tagline"`
	FirstAirDate   string  `json:"first_air_date"`
	Status         string  `json:"status"`
	VoteAverage    float64 `json:"vote_average"`
	VoteCount      int     `json:"vote_count"`
	PosterPath     string  `json:"poster_path"`
	BackdropPath   string  `json:"backdrop_path"`
	EpisodeRunTime []int   `json:"episode_run_time"`
	Genres         []struct {
		Name string `json:"name"`
	} `json:"genres"`
	CreatedBy []Credit `json:"created_by"`
	Credits   struct {
		Cast []Credit `json:"cast"`
	} `json:"credits"`
	ContentRatings struct {
		Results []struct {
			Country string `json:"iso_3166_1"`
			Rating  string `json:"rating"`
		} `json:"results"`
	} `json:"content_ratings"`
	Seasons []struct {
		SeasonNumber int    `json:"season_number"`
		Name         string `json:"name"`
		Overview     string `json:"overview"`
		PosterPath   string `json:"poster_path"`
		AirDate      string `json:"air_date"`
	} `json:"seasons"`
}

// TV récupère une fiche série.
func (c *Client) TV(ctx context.Context, id int) (*TVDetails, error) {
	var d TVDetails
	q := url.Values{"language": {c.Language}, "append_to_response": {"credits,content_ratings"}}
	if err := c.get(ctx, fmt.Sprintf("/tv/%d", id), q, &d); err != nil {
		return nil, err
	}
	if d.Overview == "" && !strings.HasPrefix(c.Language, "en") {
		var en TVDetails
		if c.get(ctx, fmt.Sprintf("/tv/%d", id), url.Values{"language": {"en-US"}}, &en) == nil {
			d.Overview = en.Overview
		}
	}
	return &d, nil
}

// Certification série.
func (d *TVDetails) Certification() (string, *int) {
	byCountry := map[string]string{}
	for _, r := range d.ContentRatings.Results {
		if r.Rating != "" {
			byCountry[r.Country] = r.Rating
		}
	}
	return certification(byCountry)
}

// SeasonDetails : épisodes d'une saison.
type SeasonDetails struct {
	Episodes []struct {
		EpisodeNumber int     `json:"episode_number"`
		Name          string  `json:"name"`
		Overview      string  `json:"overview"`
		AirDate       string  `json:"air_date"`
		Runtime       int     `json:"runtime"`
		StillPath     string  `json:"still_path"`
		VoteAverage   float64 `json:"vote_average"`
	} `json:"episodes"`
}

// Season récupère une saison.
func (c *Client) Season(ctx context.Context, tvID, season int) (*SeasonDetails, error) {
	var d SeasonDetails
	err := c.get(ctx, fmt.Sprintf("/tv/%d/season/%d", tvID, season), url.Values{"language": {c.Language}}, &d)
	return &d, err
}

func lang2(l string) string {
	b, _, _ := strings.Cut(l, "-")
	return b
}

// certification : France puis États-Unis. L'âge n'est déduit que de valeurs connues ; sinon nil
// (le contrôle parental traite alors le contenu comme « non classé »).
func certification(byCountry map[string]string) (string, *int) {
	age := func(v int) *int { return &v }
	if c, ok := byCountry["FR"]; ok {
		switch strings.ToUpper(strings.TrimSpace(c)) {
		case "U", "TP", "TOUS PUBLICS", "0":
			return "Tous publics", age(0)
		case "10", "-10":
			return "-10", age(10)
		case "12", "-12":
			return "-12", age(12)
		case "16", "-16":
			return "-16", age(16)
		case "18", "-18":
			return "-18", age(18)
		}
	}
	if c, ok := byCountry["US"]; ok {
		switch strings.ToUpper(strings.TrimSpace(c)) {
		case "G", "TV-Y", "TV-G":
			return c, age(0)
		case "TV-Y7":
			return c, age(7)
		case "PG", "TV-PG":
			return c, age(10)
		case "PG-13":
			return c, age(13)
		case "TV-14":
			return c, age(14)
		case "R", "TV-MA":
			return c, age(17)
		case "NC-17":
			return c, age(18)
		}
	}
	return "", nil
}

// Artwork : image proposée par TMDB pour une œuvre.
type Artwork struct {
	Path     string  `json:"path"`     // chemin TMDB (ex. /abc.jpg)
	Preview  string  `json:"preview"`  // aperçu (w342 ou w780), chargé par le navigateur de l'administrateur
	Language string  `json:"language"` // fr, en… ou vide (sans texte)
	Votes    float64 `json:"votes"`
	Width    int     `json:"width"`
	Height   int     `json:"height"`
}

// Images liste les affiches et fonds d'un film (kind="movie") ou d'une série (kind="tv").
func (c *Client) Images(ctx context.Context, kind string, id int) (posters, backdrops []Artwork, err error) {
	var r struct {
		Posters   []imgEntry `json:"posters"`
		Backdrops []imgEntry `json:"backdrops"`
	}
	q := url.Values{"include_image_language": {lang2(c.Language) + ",en,null"}}
	if err := c.get(ctx, fmt.Sprintf("/%s/%d/images", kind, id), q, &r); err != nil {
		return nil, nil, err
	}
	conv := func(in []imgEntry, size string) []Artwork {
		out := make([]Artwork, 0, len(in))
		for _, x := range in {
			out = append(out, Artwork{Path: x.FilePath, Preview: c.ImageURL + "/" + size + x.FilePath, Language: x.Lang,
				Votes: x.VoteAverage, Width: x.Width, Height: x.Height})
		}
		return out
	}
	return conv(r.Posters, "w342"), conv(r.Backdrops, "w780"), nil
}

type imgEntry struct {
	FilePath    string  `json:"file_path"`
	Lang        string  `json:"iso_639_1"`
	VoteAverage float64 `json:"vote_average"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
}
