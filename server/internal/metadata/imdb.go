package metadata

import (
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"aurora/internal/catalog"
	"aurora/internal/parse"
)

// ErrIMDbNotFound distingue une absence (ou un type incompatible) d'une panne réseau.
var ErrIMDbNotFound = errors.New("fiche IMDb introuvable pour ce type d'œuvre")
var imdbIDPattern = regexp.MustCompile(`^tt\d{7,10}$`)

type imdbSuggestion struct {
	ID         string `json:"id"`
	Title      string `json:"l"`
	Kind       string `json:"qid"`
	LegacyKind string `json:"q"`
	Year       int    `json:"y"`
	Image      struct {
		URL string `json:"imageUrl"`
	} `json:"i"`
}

func (s imdbSuggestion) kind() string {
	switch s.Kind {
	case "movie", "tvMovie", "short", "tvSpecial":
		return "movie"
	case "tvSeries", "tvMiniSeries":
		return "tv"
	case "":
		switch strings.ToLower(s.LegacyKind) {
		case "feature", "tv movie", "short", "tv special":
			return "movie"
		case "tv series", "tv mini-series", "tv mini series":
			return "tv"
		}
	}
	return ""
}

// imdbSuggestions utilise le service de suggestions public IMDb, sans transmettre la clé TMDB.
// Ce service ne fournit ni synopsis, ni classification : ces données restent inconnues sans fiche TMDB.
func (c *Client) imdbSuggestions(ctx context.Context, query string) ([]imdbSuggestion, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	query = strings.TrimSpace(query)
	if query == "" || len([]rune(query)) > 300 {
		return []imdbSuggestion{}, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(c.IMDbURL, "/")+"/"+url.PathEscape(query)+".json", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", c.Language)
	if err := c.pace(ctx); err != nil {
		return nil, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("IMDb injoignable : %w", stripURL(err))
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return []imdbSuggestion{}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("IMDb : statut %d", resp.StatusCode)
	}
	var data struct {
		Results []imdbSuggestion `json:"d"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&data); err != nil {
		return nil, fmt.Errorf("réponse IMDb invalide : %w", err)
	}
	return data.Results, nil
}

func (c *Client) imdbResult(ctx context.Context, s imdbSuggestion) (SearchResult, error) {
	result := SearchResult{IMDbID: s.ID, Title: s.Title, OriginalTitle: s.Title, Year: s.Year}
	kind, id, err := c.FindIMDb(ctx, s.ID)
	if err == nil {
		if kind != s.kind() {
			return SearchResult{}, ErrIMDbNotFound
		}
		mapped, err := c.Lookup(ctx, kind, id)
		if err != nil {
			return SearchResult{}, err
		}
		mapped.IMDbID = s.ID
		mapped.imdbTitle = s.Title
		return *mapped, nil
	}
	if !errors.Is(err, ErrIMDbNotFound) && !errors.Is(err, ErrNoKey) {
		return SearchResult{}, err
	}
	if s.Image.URL != "" {
		name, imageErr := c.imdbImage(ctx, s.Image.URL)
		if imageErr != nil && ctx.Err() != nil {
			return SearchResult{}, ctx.Err()
		}
		result.Poster = catalog.ImageURL(name)
	}
	return result, nil
}

func (c *Client) searchIMDb(ctx context.Context, kind, query string, year int) ([]SearchResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	suggestions, err := c.imdbSuggestions(ctx, query)
	if err != nil {
		return nil, err
	}
	results := make([]SearchResult, 0, len(suggestions))
	seen := map[string]bool{}
	for _, s := range suggestions {
		if len(results) >= 8 {
			break
		}
		if !imdbIDPattern.MatchString(s.ID) || s.kind() != kind || s.Title == "" || len([]rune(s.Title)) > 300 || seen[s.ID] {
			continue
		}
		seen[s.ID] = true
		result, err := c.imdbResult(ctx, s)
		if errors.Is(err, ErrIMDbNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	rankCandidates(results, query, year)
	return results, nil
}

// LookupIMDb revérifie l'identité et le type auprès d'IMDb avant une association manuelle.
func (c *Client) LookupIMDb(ctx context.Context, kind, imdbID string) (*SearchResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	if !imdbIDPattern.MatchString(imdbID) || (kind != "movie" && kind != "tv") {
		return nil, ErrIMDbNotFound
	}
	suggestions, err := c.imdbSuggestions(ctx, imdbID)
	if err != nil {
		return nil, err
	}
	for _, s := range suggestions {
		if s.ID == imdbID && s.kind() == kind && s.Title != "" {
			result, err := c.imdbResult(ctx, s)
			return &result, err
		}
	}
	return nil, ErrIMDbNotFound
}

// SearchCandidates cherche d'abord sur TMDB ; une année trop restrictive est retirée, puis de petites
// variantes du titre sont essayées. IMDb est le dernier recours, y compris sans équivalent TMDB.
// Le classement approchant sert seulement à l'affichage, jamais à Choose.
func (c *Client) SearchCandidates(ctx context.Context, kind, query string, year int) ([]SearchResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if kind != "movie" && kind != "tv" {
		return nil, ErrBadLink
	}
	query = strings.TrimSpace(query)
	if query == "" || len([]rune(query)) > 300 {
		return []SearchResult{}, nil
	}
	results, err := c.Search(ctx, kind, query, year)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 && year > 0 {
		results, err = c.Search(ctx, kind, query, 0)
		if err != nil {
			return nil, err
		}
	}
	if len(results) == 0 {
		for _, variant := range searchVariants(query) {
			results, err = c.Search(ctx, kind, variant, 0)
			if err != nil {
				return nil, err
			}
			if len(results) > 0 {
				rankCandidates(results, query, year)
				if candidateScore(results[0], query) > 0 {
					break
				}
			}
		}
	}
	rankCandidates(results, query, year)
	if len(results) > 0 && candidateScore(results[0], query) > 0 {
		return results, nil
	}
	imdb, err := c.searchIMDb(ctx, kind, query, year)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if len(results) > 0 {
			return results, nil
		}
		return nil, err
	}
	if len(imdb) > 0 {
		return imdb, nil
	}
	return results, nil
}

func searchVariants(query string) []string {
	words := strings.Fields(parse.Normalize(query))
	if len(words) == 0 {
		return nil
	}
	variants := []string{}
	singular := append([]string(nil), words...)
	for i, w := range singular {
		if len([]rune(w)) > 3 && strings.HasSuffix(w, "s") {
			singular[i] = strings.TrimSuffix(w, "s")
		}
	}
	if s := strings.Join(singular, " "); s != parse.Normalize(query) {
		variants = append(variants, s)
	}
	// Au plus deux mots distinctifs : si le plus long contient la faute, le suivant reste utile.
	if len(words) > 1 {
		sort.SliceStable(words, func(i, j int) bool { return len([]rune(words[i])) > len([]rune(words[j])) })
		seen := map[string]bool{}
		for _, w := range words {
			if len([]rune(w)) < 4 || seen[w] {
				continue
			}
			variants = append(variants, w)
			seen[w] = true
			if len(seen) == 2 {
				break
			}
		}
	}
	return variants
}

func candidateScore(r SearchResult, query string) int {
	return max(titleScore(r.Title, query), titleScore(r.OriginalTitle, query), titleScore(r.imdbTitle, query))
}
func titleScore(title, query string) int {
	title, query = parse.Normalize(title), parse.Normalize(query)
	if title == query {
		return 1000
	}
	if title == "" || query == "" {
		return 0
	}
	if strings.Contains(title, query) {
		return 800
	}
	a, b := []rune(title), []rune(query)
	limit := max(len(a), len(b)) / 5
	if limit < 1 {
		limit = 1
	}
	if distance := titleDistance(a, b); distance <= limit {
		return 700 - distance*30
	}
	words := strings.Fields(title)
	matched := 0
	for _, q := range strings.Fields(query) {
		for _, w := range words {
			if q == w || (min(len([]rune(q)), len([]rune(w))) >= 4 && titleDistance([]rune(q), []rune(w)) <= 1) {
				matched++
				break
			}
		}
	}
	if matched == len(strings.Fields(query)) {
		return 500
	}
	return 0
}
func rankCandidates(results []SearchResult, query string, year int) {
	sort.SliceStable(results, func(i, j int) bool {
		a, b := candidateScore(results[i], query), candidateScore(results[j], query)
		if a != b {
			return a > b
		}
		if year > 0 {
			a, b = absYear(results[i].Year-year), absYear(results[j].Year-year)
			if a != b {
				return a < b
			}
		}
		return false
	})
}
func absYear(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
func titleDistance(a, b []rune) int {
	prev := make([]int, len(b)+1)
	before := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i, x := range a {
		cur := make([]int, len(b)+1)
		cur[0] = i + 1
		for j, y := range b {
			cost := 0
			if x != y {
				cost = 1
			}
			cur[j+1] = min(prev[j+1]+1, cur[j]+1, prev[j]+cost)
			if i > 0 && j > 0 && x == b[j-1] && a[i-1] == y {
				cur[j+1] = min(cur[j+1], before[j-1]+1)
			}
		}
		before, prev = prev, cur
	}
	return prev[len(b)]
}

func validIMDbImage(u *url.URL) bool {
	return u.Scheme == "https" && u.Host == "m.media-amazon.com" && u.User == nil && strings.HasPrefix(u.Path, "/images/M/")
}
func (c *Client) imdbImage(ctx context.Context, raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || !validIMDbImage(u) {
		return "", errors.New("image IMDb refusée")
	}
	ext := strings.ToLower(filepath.Ext(u.Path))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp":
	default:
		return "", errors.New("format image IMDb refusé")
	}
	hash := sha1.Sum([]byte(raw))
	name := hex.EncodeToString(hash[:]) + ext
	dst := filepath.Join(c.ImageDir, name)
	if st, err := os.Stat(dst); err == nil && st.Size() > 0 {
		return name, nil
	}
	if err := os.MkdirAll(c.ImageDir, 0o750); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return "", err
	}
	client := *c.HTTP
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 || !validIMDbImage(req.URL) {
			return errors.New("redirection image IMDb refusée")
		}
		return nil
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", stripURL(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("image IMDb : statut %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, (10<<20)+1))
	if err != nil {
		return "", err
	}
	if len(data) > 10<<20 {
		return "", errors.New("image IMDb trop volumineuse")
	}
	kind := http.DetectContentType(data)
	if kind != "image/jpeg" && kind != "image/png" && kind != "image/webp" {
		return "", errors.New("contenu image IMDb invalide")
	}
	// Le nom temporaire unique évite les collisions entre deux recherches simultanées.
	f, err := os.CreateTemp(c.ImageDir, "imdb-*.tmp")
	if err != nil {
		return "", err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	if err := os.Rename(f.Name(), dst); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return name, nil
}

// ApplyIMDb associe une identité IMDb vérifiée, même sans fiche TMDB. Les champs indisponibles restent
// inconnus, notamment la classification parentale ; seules les informations réellement reçues sont écrites.
func (e *Enricher) ApplyIMDb(ctx context.Context, c *Client, itemID int64, imdbID, state string) (int64, error) {
	var kind string
	if err := e.DB.QueryRowContext(ctx, `SELECT kind FROM items WHERE id=?`, itemID).Scan(&kind); err != nil {
		return 0, err
	}
	tmdbKind := map[string]string{"movie": "movie", "series": "tv"}[kind]
	if tmdbKind == "" {
		return 0, ErrIMDbNotFound
	}
	result, err := c.LookupIMDb(ctx, tmdbKind, imdbID)
	if err != nil {
		return 0, err
	}
	tx, err := e.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var previousTMDB sql.NullInt64
	var previousIMDb sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT tmdb_id,imdb_id FROM items WHERE id=?`, itemID).Scan(&previousTMDB, &previousIMDb); err != nil {
		return 0, err
	}
	identityChanged := previousTMDB.Valid || (previousIMDb.Valid && previousIMDb.String != "" && previousIMDb.String != imdbID)
	target := itemID
	var duplicate int64
	err = tx.QueryRowContext(ctx, `SELECT o.id FROM items o JOIN items s ON s.id=? WHERE o.library_id=s.library_id AND o.kind=s.kind AND (o.imdb_id=? OR (? > 0 AND o.tmdb_id=?)) AND o.id<>s.id ORDER BY CASE WHEN o.match_state='locked' THEN 0 ELSE 1 END,o.id LIMIT 1`, itemID, imdbID, result.ID, result.ID).Scan(&duplicate)
	if err == nil {
		if err := MergeItems(ctx, tx, itemID, duplicate); err != nil {
			return 0, err
		}
		target = duplicate
	} else if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	var targetState string
	if err := tx.QueryRowContext(ctx, `SELECT match_state,tmdb_id,imdb_id FROM items WHERE id=?`, target).Scan(&targetState, &previousTMDB, &previousIMDb); err != nil {
		return 0, err
	}
	if state != "locked" && targetState == "locked" {
		return target, tx.Commit()
	} // fusion automatique : le choix humain de la cible reste intact
	identityChanged = identityChanged || previousTMDB.Valid || (previousIMDb.Valid && previousIMDb.String != "" && previousIMDb.String != imdbID)
	if result.ID > 0 {
		if err := tx.Commit(); err != nil {
			return 0, err
		}
		if kind == "movie" {
			target, err = e.ApplyMovie(ctx, c, target, result.ID, state)
		} else {
			target, err = e.ApplySeries(ctx, c, target, result.ID, state)
		}
		if err != nil {
			return target, err
		}
		_, err = e.DB.ExecContext(ctx, `UPDATE items SET imdb_id=? WHERE id=?`, imdbID, target)
		return target, err
	}
	var year any
	if result.Year > 0 {
		year = result.Year
	}
	poster := strings.TrimPrefix(result.Poster, "/api/v1/img/")
	now := time.Now().Unix()
	_, err = tx.ExecContext(ctx, `UPDATE items SET title=?,sort_title=?,original_title=?,year=?,imdb_id=?,tmdb_id=NULL,poster=?,overview='',tagline='',runtime=NULL,genres='[]',rating=NULL,rating_source='',content_rating='',min_age=NULL,status='',backdrop='',trailer='',match_state=?,metadata_at=?,updated_at=? WHERE id=?`, result.Title, parse.SortTitle(result.Title), result.OriginalTitle, year, imdbID, poster, state, now, now, target)
	if err != nil {
		return 0, err
	}
	if err := replaceCredits(ctx, tx, target, nil, nil, ""); err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM collection_items WHERE item_id=? AND collection_id IN (SELECT id FROM collections WHERE kind='tmdb')`, target); err != nil {
		return 0, err
	}
	if kind == "series" {
		if _, err := tx.ExecContext(ctx, `UPDATE items SET content_rating='',min_age=NULL WHERE parent_id=?`, target); err != nil {
			return 0, err
		}
		if identityChanged {
			if _, err := tx.ExecContext(ctx, `UPDATE seasons SET title='',overview='',poster='',air_date='' WHERE series_id=?`, target); err != nil {
				return 0, err
			}
			rows, err := tx.QueryContext(ctx, `SELECT id,episode FROM items WHERE parent_id=?`, target)
			if err != nil {
				return 0, err
			}
			type episodeIdentity struct {
				id     int64
				number int
			}
			var episodes []episodeIdentity
			for rows.Next() {
				var ep episodeIdentity
				if err := rows.Scan(&ep.id, &ep.number); err != nil {
					rows.Close()
					return 0, err
				}
				episodes = append(episodes, ep)
			}
			rowErr := rows.Err()
			rows.Close()
			if rowErr != nil {
				return 0, rowErr
			}
			for _, ep := range episodes {
				title := fmt.Sprintf("Épisode %d", ep.number)
				if _, err := tx.ExecContext(ctx, `UPDATE items SET
                    title=CASE WHEN match_state='locked' THEN title ELSE ? END,
                    sort_title=CASE WHEN match_state='locked' THEN sort_title ELSE ? END,
                    year=CASE WHEN match_state='locked' THEN year ELSE NULL END,
                    overview=CASE WHEN match_state='locked' THEN overview ELSE '' END,
                    original_title='',tagline='',air_date='',runtime=NULL,genres='[]',rating=NULL,rating_source='',
                    status='',tmdb_id=NULL,imdb_id=NULL,poster='',backdrop='',still='',trailer='',
                    metadata_at=CASE WHEN match_state='locked' THEN metadata_at ELSE NULL END,
                    match_state=CASE WHEN match_state='locked' THEN match_state ELSE 'local' END,
                    updated_at=? WHERE id=?`, title, parse.SortTitle(title), now, ep.id); err != nil {
					return 0, err
				}
				if err := replaceCredits(ctx, tx, ep.id, nil, nil, ""); err != nil {
					return 0, err
				}
				if err := catalog.ReindexItem(ctx, tx, ep.id); err != nil {
					return 0, err
				}
			}
		}
	}
	if err := catalog.ReindexItem(ctx, tx, target); err != nil {
		return 0, err
	}
	return target, tx.Commit()
}

// chooseIMDb applique exactement Choose en utilisant seulement des identifiants temporaires d'index.
// Aucun identifiant IMDb n'est interprété comme un identifiant TMDB.
func chooseIMDb(results []SearchResult, title string, year int) (*SearchResult, string) {
	indexed := append([]SearchResult(nil), results...)
	for i := range indexed {
		indexed[i].ID = i + 1
		if indexed[i].imdbTitle != "" && parse.Normalize(indexed[i].imdbTitle) == parse.Normalize(title) {
			indexed[i].Title = indexed[i].imdbTitle
		}
	}
	index, state := Choose(indexed, title, year)
	if index == 0 {
		return nil, state
	}
	return &results[index-1], state
}
