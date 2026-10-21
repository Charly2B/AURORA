package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"aurora/internal/probe"
)

// Card : vignette d'œuvre (grilles, carrousels).
type Card struct {
	ID          int64    `json:"id"`
	Kind        string   `json:"kind"`
	Title       string   `json:"title"`
	Year        *int     `json:"year,omitempty"`
	Poster      string   `json:"poster,omitempty"`
	Backdrop    string   `json:"backdrop,omitempty"`
	Thumb       string   `json:"thumb,omitempty"`
	Rating      *float64 `json:"rating,omitempty"`
	Runtime     *int     `json:"runtime,omitempty"` // minutes
	Duration    float64  `json:"duration,omitempty"`
	Resolution  string   `json:"resolution,omitempty"`
	HDR         string   `json:"hdr,omitempty"`
	Watched     bool     `json:"watched"`
	Position    float64  `json:"position,omitempty"`
	Favorite    bool     `json:"favorite"`
	Watchlist   bool     `json:"watchlist"`
	Overview    string   `json:"overview,omitempty"`
	Genres      []string `json:"genres,omitempty"`
	AddedAt     int64    `json:"addedAt"`
	SeriesID    int64    `json:"seriesId,omitempty"`
	SeriesTitle string   `json:"seriesTitle,omitempty"`
	Season      *int     `json:"season,omitempty"`
	Episode     *int     `json:"episode,omitempty"`
	EpisodeEnd  *int     `json:"episodeEnd,omitempty"`
	Unwatched   int      `json:"unwatched,omitempty"` // séries : épisodes non vus
	Total       int      `json:"total,omitempty"`     // séries : épisodes disponibles
	Versions    int      `json:"versions,omitempty"`
}

// ImageURL construit l'URL d'une image en cache.
func ImageURL(name string) string {
	if name == "" {
		return ""
	}
	return "/api/v1/img/" + name
}

// ResolutionLabel : hauteur -> libellé.
func ResolutionLabel(w, h int) string {
	switch {
	case h == 0 && w == 0:
		return ""
	case w >= 3200 || h >= 2000:
		return "4K"
	case w >= 1800 || h >= 1000:
		return "1080p"
	case w >= 1200 || h >= 700:
		return "720p"
	default:
		return "SD"
	}
}

// cardColumns : colonnes communes ; l'alias i = item, st = user_state.
const cardColumns = `i.id, i.kind, i.title, i.year, i.poster, i.backdrop, i.still, i.rating, i.runtime, i.overview, i.genres, i.added_at,
	i.season, i.episode, i.episode_end, i.parent_id,
	COALESCE(st.watched,0), COALESCE(st.position,0), COALESCE(st.favorite,0), COALESCE(st.watchlist,0),
	(SELECT MAX(f.duration) FROM media_files f WHERE f.item_id=i.id AND f.present=1),
	(SELECT MAX(COALESCE(f.width,0)) FROM media_files f WHERE f.item_id=i.id AND f.present=1),
	(SELECT MAX(COALESCE(f.height,0)) FROM media_files f WHERE f.item_id=i.id AND f.present=1),
	(SELECT MAX(f.hdr) FROM media_files f WHERE f.item_id=i.id AND f.present=1),
	(SELECT COUNT(*) FROM media_files f WHERE f.item_id=i.id AND f.present=1),
	COALESCE(p.title,''), p.poster, p.backdrop,
	CASE WHEN i.kind='series' THEN (SELECT e.id FROM items e WHERE e.parent_id=i.id
		AND EXISTS (SELECT 1 FROM media_files f WHERE f.item_id=e.id AND f.present=1)
		ORDER BY CASE WHEN e.season=0 THEN 1 ELSE 0 END, e.season, e.episode LIMIT 1) END`

const cardJoins = ` LEFT JOIN user_state st ON st.item_id=i.id AND st.user_id=? LEFT JOIN items p ON p.id=i.parent_id `

func scanCards(rows *sql.Rows) ([]Card, error) {
	defer rows.Close()
	out := []Card{}
	for rows.Next() {
		var c Card
		var year, season, episode, episodeEnd, runtime sql.NullInt64
		var parent sql.NullInt64
		var rating, duration sql.NullFloat64
		var poster, backdrop, still, genres string
		var width, height sql.NullInt64
		var hdr sql.NullString
		var parentPoster, parentBackdrop sql.NullString
		var firstEp sql.NullInt64
		if err := rows.Scan(&c.ID, &c.Kind, &c.Title, &year, &poster, &backdrop, &still, &rating, &runtime, &c.Overview, &genres,
			&c.AddedAt, &season, &episode, &episodeEnd, &parent, &c.Watched, &c.Position, &c.Favorite, &c.Watchlist,
			&duration, &width, &height, &hdr, &c.Versions, &c.SeriesTitle, &parentPoster, &parentBackdrop, &firstEp); err != nil {
			return nil, err
		}
		if c.Kind == "series" && firstEp.Valid {
			c.Thumb = fmt.Sprintf("/api/v1/items/%d/thumb", firstEp.Int64) // repli sans affiche : image du premier épisode
		}
		c.Year = intp(year)
		c.Season, c.Episode, c.EpisodeEnd, c.Runtime = intp(season), intp(episode), intp(episodeEnd), intp(runtime)
		if rating.Valid {
			c.Rating = &rating.Float64
		}
		c.Duration = duration.Float64
		c.Resolution = ResolutionLabel(int(width.Int64), int(height.Int64))
		c.HDR = hdr.String
		c.Poster, c.Backdrop = ImageURL(poster), ImageURL(backdrop)
		json.Unmarshal([]byte(genres), &c.Genres)
		if c.Kind == "episode" {
			c.SeriesID = parent.Int64
			if still != "" {
				c.Thumb = ImageURL(still)
			}
			if c.Poster == "" {
				c.Poster = ImageURL(parentPoster.String)
			}
			if c.Backdrop == "" {
				c.Backdrop = ImageURL(parentBackdrop.String)
			}
		} else {
			c.SeriesTitle = ""
		}
		if c.Thumb == "" && c.Kind != "series" && c.Versions > 0 {
			c.Thumb = fmt.Sprintf("/api/v1/items/%d/thumb", c.ID)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func intp(v sql.NullInt64) *int {
	if !v.Valid {
		return nil
	}
	x := int(v.Int64)
	return &x
}

// Store : accès lecture au catalogue.
type Store struct{ DB *sql.DB }

// ListQuery : paramètres de liste (films ou séries).
type ListQuery struct {
	Kind       string // movie | series
	LibraryID  int64
	Sort       string // title, added, year, runtime, rating, quality
	Desc       bool
	Genre      string
	Year       int
	Decade     int
	Resolution string // 4K, 1080p, 720p, SD
	HDR        bool
	Watched    string // "", "watched", "unwatched"
	Favorite   bool
	Watchlist  bool
	Person     string
	Collection int64
	Language   string // code ISO 639 d'une piste audio
	Offset     int
	Limit      int
}

var sortColumns = map[string]string{
	"title":   "i.sort_title",
	"added":   "i.added_at",
	"year":    "COALESCE(i.year,0)",
	"runtime": "COALESCE(i.runtime, (SELECT MAX(f.duration)/60 FROM media_files f WHERE f.item_id=i.id))",
	"rating":  "COALESCE(i.rating,0)",
	"quality": "(SELECT MAX(COALESCE(f.height,0)) FROM media_files f WHERE f.item_id=i.id AND f.present=1)",
}

// List renvoie une page de vignettes et le total.
func (s *Store) List(ctx context.Context, v Viewer, q ListQuery) ([]Card, int, error) {
	where, args := v.Where("i")
	conds := []string{where}
	if q.Kind == "series" {
		conds = append(conds, "i.kind='series'", presentSeries)
	} else {
		conds = append(conds, "i.kind='movie'", presentMovie)
	}
	if q.LibraryID > 0 {
		conds = append(conds, "i.library_id=?")
		args = append(args, q.LibraryID)
	}
	if q.Genre != "" {
		conds = append(conds, "EXISTS (SELECT 1 FROM json_each(i.genres) g WHERE g.value = ?)")
		args = append(args, q.Genre)
	}
	if q.Year > 0 {
		conds = append(conds, "i.year=?")
		args = append(args, q.Year)
	}
	if q.Decade > 0 {
		conds = append(conds, "i.year BETWEEN ? AND ?")
		args = append(args, q.Decade, q.Decade+9)
	}
	if q.Person != "" {
		conds = append(conds, "EXISTS (SELECT 1 FROM credits c WHERE c.item_id=i.id AND c.name=?)")
		args = append(args, q.Person)
	}
	if q.Collection > 0 {
		conds = append(conds, "EXISTS (SELECT 1 FROM collection_items ci WHERE ci.item_id=i.id AND ci.collection_id=?)")
		args = append(args, q.Collection)
	}
	fileCond := func(c string, a ...any) {
		target := "f.item_id=i.id"
		if q.Kind == "series" {
			target = "f.item_id IN (SELECT id FROM items e WHERE e.parent_id=i.id)"
		}
		conds = append(conds, "EXISTS (SELECT 1 FROM media_files f WHERE "+target+" AND f.present=1 AND "+c+")")
		args = append(args, a...)
	}
	switch q.Resolution {
	case "4K":
		fileCond("(f.width >= 3200 OR f.height >= 2000)")
	case "1080p":
		fileCond("(f.width >= 1800 OR f.height >= 1000) AND f.width < 3200 AND f.height < 2000")
	case "720p":
		fileCond("(f.width >= 1200 OR f.height >= 700) AND f.width < 1800 AND f.height < 1000")
	case "SD":
		fileCond("f.width < 1200 AND f.height < 700")
	}
	if q.HDR {
		fileCond("f.hdr <> ''")
	}
	if q.Language != "" {
		fileCond("EXISTS (SELECT 1 FROM json_each(f.audio) a WHERE json_extract(a.value,'$.language') = ?)", q.Language)
	}
	if q.Kind != "series" {
		switch q.Watched {
		case "watched":
			conds = append(conds, "COALESCE(st.watched,0)=1")
		case "unwatched":
			conds = append(conds, "COALESCE(st.watched,0)=0")
		}
	} else {
		unwatched := `EXISTS (SELECT 1 FROM items e LEFT JOIN user_state es ON es.item_id=e.id AND es.user_id=st_user.uid
			WHERE e.parent_id=i.id AND COALESCE(es.watched,0)=0 AND EXISTS (SELECT 1 FROM media_files f WHERE f.item_id=e.id AND f.present=1))`
		switch q.Watched {
		case "watched":
			conds = append(conds, "NOT "+unwatched)
		case "unwatched":
			conds = append(conds, unwatched)
		}
	}
	if q.Favorite {
		conds = append(conds, "COALESCE(st.favorite,0)=1")
	}
	if q.Watchlist {
		conds = append(conds, "COALESCE(st.watchlist,0)=1")
	}
	col, ok := sortColumns[q.Sort]
	if !ok {
		col = sortColumns["title"]
	}
	dir := "ASC"
	if q.Desc {
		dir = "DESC"
	}
	if q.Limit <= 0 || q.Limit > 500 {
		q.Limit = 100
	}
	from := " FROM items i" + cardJoins + " JOIN (SELECT ? AS uid) st_user "
	whereSQL := " WHERE " + strings.Join(conds, " AND ")
	allArgs := append([]any{v.UserID, v.UserID}, args...)

	var total int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*)"+from+whereSQL, allArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.DB.QueryContext(ctx, "SELECT "+cardColumns+from+whereSQL+
		" ORDER BY "+col+" "+dir+", i.sort_title ASC, i.id ASC LIMIT ? OFFSET ?", append(allArgs, q.Limit, q.Offset)...)
	if err != nil {
		return nil, 0, err
	}
	cards, err := scanCards(rows)
	if err != nil {
		return nil, 0, err
	}
	if q.Kind == "series" {
		s.fillSeriesCounts(ctx, v, cards)
	}
	return cards, total, nil
}

// fillSeriesCounts calcule épisodes disponibles / non vus pour des vignettes de séries.
func (s *Store) fillSeriesCounts(ctx context.Context, v Viewer, cards []Card) {
	for i := range cards {
		if cards[i].Kind != "series" {
			continue
		}
		s.DB.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(SUM(CASE WHEN COALESCE(st.watched,0)=0 THEN 1 ELSE 0 END),0)
			FROM items e LEFT JOIN user_state st ON st.item_id=e.id AND st.user_id=?
			WHERE e.parent_id=? AND e.season > 0 AND EXISTS (SELECT 1 FROM media_files f WHERE f.item_id=e.id AND f.present=1)`,
			v.UserID, cards[i].ID).Scan(&cards[i].Total, &cards[i].Unwatched)
	}
}

// Facets : valeurs disponibles pour les filtres.
type Facets struct {
	Genres      []string `json:"genres"`
	Years       []int    `json:"years"`
	Resolutions []string `json:"resolutions"`
	Languages   []string `json:"languages"`
	HasHDR      bool     `json:"hasHdr"`
	People      []string `json:"people"` // réalisateurs les plus présents
}

// Facets calcule les filtres utiles (on ne propose que des valeurs existantes).
func (s *Store) Facets(ctx context.Context, v Viewer, kind string, libraryID int64) (*Facets, error) {
	where, args := v.Where("i")
	if libraryID > 0 { // valeurs propres au dossier affiché
		where += " AND i.library_id = " + strconv.FormatInt(libraryID, 10)
	}
	itemKind, present := "movie", presentMovie
	if kind == "series" {
		itemKind, present = "series", presentSeries
	}
	base := " FROM items i WHERE i.kind='" + itemKind + "' AND " + present + " AND " + where
	f := &Facets{Genres: []string{}, Years: []int{}, Resolutions: []string{}, Languages: []string{}, People: []string{}}
	rows, err := s.DB.QueryContext(ctx, "SELECT DISTINCT g.value FROM items i, json_each(i.genres) g WHERE i.kind='"+itemKind+
		"' AND "+present+" AND "+where+" ORDER BY g.value", args...)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var g string
		if rows.Scan(&g) == nil {
			f.Genres = append(f.Genres, g)
		}
	}
	rows.Close()
	rows, err = s.DB.QueryContext(ctx, "SELECT DISTINCT i.year"+base+" AND i.year IS NOT NULL ORDER BY i.year DESC", args...)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var y int
		if rows.Scan(&y) == nil {
			f.Years = append(f.Years, y)
		}
	}
	rows.Close()
	fileScope := "f.item_id IN (SELECT i.id" + base + ")"
	if kind == "series" {
		fileScope = "f.item_id IN (SELECT e.id FROM items e WHERE e.parent_id IN (SELECT i.id" + base + "))"
	}
	rows, err = s.DB.QueryContext(ctx, `SELECT DISTINCT CASE WHEN f.width>=3200 OR f.height>=2000 THEN '4K'
		WHEN f.width>=1800 OR f.height>=1000 THEN '1080p' WHEN f.width>=1200 OR f.height>=700 THEN '720p'
		WHEN f.width>0 THEN 'SD' END FROM media_files f WHERE f.present=1 AND `+fileScope, args...)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var r sql.NullString
		if rows.Scan(&r) == nil && r.Valid {
			f.Resolutions = append(f.Resolutions, r.String)
		}
	}
	rows.Close()
	order := map[string]int{"4K": 0, "1080p": 1, "720p": 2, "SD": 3}
	sortBy(f.Resolutions, func(a, b string) bool { return order[a] < order[b] })
	s.DB.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM media_files f WHERE f.present=1 AND f.hdr<>'' AND "+fileScope+")", args...).Scan(&f.HasHDR)
	rows, err = s.DB.QueryContext(ctx, `SELECT json_extract(a.value,'$.language') l, COUNT(*) n FROM media_files f, json_each(f.audio) a
		WHERE f.present=1 AND `+fileScope+` AND l <> '' GROUP BY l ORDER BY n DESC LIMIT 12`, args...)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var l string
		var n int
		if rows.Scan(&l, &n) == nil {
			f.Languages = append(f.Languages, l)
		}
	}
	rows.Close()
	rows, err = s.DB.QueryContext(ctx, `SELECT c.name, COUNT(*) n FROM credits c WHERE c.role IN ('director','creator')
		AND c.item_id IN (SELECT i.id`+base+`) GROUP BY c.name HAVING n > 1 ORDER BY n DESC, c.name LIMIT 30`, args...)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var p string
		var n int
		if rows.Scan(&p, &n) == nil {
			f.People = append(f.People, p)
		}
	}
	rows.Close()
	return f, nil
}

func sortBy(s []string, less func(a, b string) bool) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && less(s[j], s[j-1]); j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// FileInfo : version (fichier) d'une œuvre, avec ses pistes.
type FileInfo struct {
	ID           int64            `json:"id"`
	Name         string           `json:"name"`
	Label        string           `json:"label"`
	Edition      string           `json:"edition,omitempty"`
	Size         int64            `json:"size"`
	Container    string           `json:"container"`
	Duration     float64          `json:"duration"`
	Bitrate      int64            `json:"bitrate"`
	VideoCodec   string           `json:"videoCodec"`
	VideoProfile string           `json:"videoProfile"`
	Width        int              `json:"width"`
	Height       int              `json:"height"`
	Resolution   string           `json:"resolution"`
	HDR          string           `json:"hdr"`
	Audio        []probe.Audio    `json:"audio"`
	Subtitles    []probe.Subtitle `json:"subtitles"`
	Chapters     []probe.Chapter  `json:"chapters"`
	ProbeState   string           `json:"probeState"`
	ProbeError   string           `json:"probeError,omitempty"`
	Present      bool             `json:"present"`
	AddedAt      int64            `json:"addedAt"`
}

// Files renvoie les versions présentes d'un item (meilleure qualité d'abord).
func (s *Store) Files(ctx context.Context, itemID int64) ([]FileInfo, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id, path, version_label, edition, size, container, COALESCE(duration,0),
		COALESCE(bitrate,0), video_codec, video_profile, COALESCE(width,0), COALESCE(height,0), hdr, audio, subtitles,
		external_subs, chapters, probe_state, probe_error, present, added_at
		FROM media_files WHERE item_id=? AND present=1 ORDER BY COALESCE(height,0) DESC, size DESC`, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []FileInfo{}
	for rows.Next() {
		var f FileInfo
		var path, audio, subs, ext, chapters string
		if err := rows.Scan(&f.ID, &path, &f.Label, &f.Edition, &f.Size, &f.Container, &f.Duration, &f.Bitrate, &f.VideoCodec,
			&f.VideoProfile, &f.Width, &f.Height, &f.HDR, &audio, &subs, &ext, &chapters, &f.ProbeState, &f.ProbeError,
			&f.Present, &f.AddedAt); err != nil {
			return nil, err
		}
		f.Name = baseName(path)
		f.Resolution = ResolutionLabel(f.Width, f.Height)
		json.Unmarshal([]byte(audio), &f.Audio)
		f.Subtitles = MergedSubtitles(subs, ext)
		json.Unmarshal([]byte(chapters), &f.Chapters)
		if f.Audio == nil {
			f.Audio = []probe.Audio{}
		}
		if f.Chapters == nil {
			f.Chapters = []probe.Chapter{}
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// MergedSubtitles concatène pistes internes puis fichiers externes, réindexés de 0 à n-1.
// Le chemin des fichiers externes est retiré : il ne quitte jamais le serveur.
func MergedSubtitles(internal, external string) []probe.Subtitle {
	all := AllSubtitles(internal, external)
	for i := range all {
		all[i].External = ""
	}
	return all
}

// AllSubtitles : comme MergedSubtitles mais conserve les chemins (usage serveur uniquement).
func AllSubtitles(internal, external string) []probe.Subtitle {
	var a, b []probe.Subtitle
	json.Unmarshal([]byte(internal), &a)
	json.Unmarshal([]byte(external), &b)
	all := append(a, b...)
	for i := range all {
		all[i].Index = i
	}
	if all == nil {
		all = []probe.Subtitle{}
	}
	return all
}

func baseName(p string) string {
	if i := strings.LastIndexAny(p, `/\`); i >= 0 {
		return p[i+1:]
	}
	return p
}

// Person : personne au générique.
type Person struct {
	Name      string `json:"name"`
	Role      string `json:"role"`
	Character string `json:"character,omitempty"`
	Photo     string `json:"photo,omitempty"`
}

// UserState : état de lecture d'un profil.
type UserState struct {
	Position     float64 `json:"position"`
	Duration     float64 `json:"duration"`
	Watched      bool    `json:"watched"`
	PlayCount    int     `json:"playCount"`
	LastPlayedAt int64   `json:"lastPlayedAt,omitempty"`
	LastFileID   int64   `json:"lastFileId,omitempty"`
	Favorite     bool    `json:"favorite"`
	Watchlist    bool    `json:"watchlist"`
}

// Detail : fiche complète.
type Detail struct {
	Card
	OriginalTitle string     `json:"originalTitle,omitempty"`
	Tagline       string     `json:"tagline,omitempty"`
	RatingSource  string     `json:"ratingSource,omitempty"`
	ContentRating string     `json:"contentRating,omitempty"`
	Status        string     `json:"status,omitempty"`
	AirDate       string     `json:"airDate,omitempty"`
	Trailer       string     `json:"trailer,omitempty"`
	MatchState    string     `json:"matchState"`
	TmdbID        *int       `json:"tmdbId,omitempty"`
	LibraryID     int64      `json:"libraryId"`
	Files         []FileInfo `json:"files"`
	People        []Person   `json:"people"`
	State         UserState  `json:"state"`
	Collections   []Card     `json:"collections,omitempty"`
	Seasons       []Season   `json:"seasons,omitempty"`
	NextUp        *Card      `json:"nextUp,omitempty"`
	Similar       []Card     `json:"similar,omitempty"`
}

// Season : saison d'une série.
type Season struct {
	Number    int    `json:"number"`
	Title     string `json:"title"`
	Overview  string `json:"overview,omitempty"`
	Poster    string `json:"poster,omitempty"`
	Episodes  int    `json:"episodes"`
	Unwatched int    `json:"unwatched"`
}

// ErrNotFound : item absent ou non visible pour ce profil.
var ErrNotFound = errors.New("introuvable")

// Item renvoie la fiche d'un item visible par le profil.
func (s *Store) Item(ctx context.Context, v Viewer, id int64) (*Detail, error) {
	where, args := v.Where("i")
	q := "SELECT " + cardColumns + ", i.original_title, i.tagline, i.rating_source, i.content_rating, i.status, i.air_date, i.trailer, " +
		"i.match_state, i.tmdb_id, i.library_id, COALESCE(st.play_count,0), COALESCE(st.last_played_at,0), COALESCE(st.last_file_id,0), " +
		"COALESCE(st.duration,0) FROM items i" + cardJoins + " WHERE i.id=? AND " + where
	row := s.DB.QueryRowContext(ctx, q, append([]any{v.UserID, id}, args...)...)
	var d Detail
	var year, season, episode, episodeEnd, runtime, parent, tmdb sql.NullInt64
	var rating, duration sql.NullFloat64
	var poster, backdrop, still, genres string
	var width, height sql.NullInt64
	var hdr, parentPoster, parentBackdrop sql.NullString
	var firstEp sql.NullInt64
	c := &d.Card
	err := row.Scan(&c.ID, &c.Kind, &c.Title, &year, &poster, &backdrop, &still, &rating, &runtime, &c.Overview, &genres,
		&c.AddedAt, &season, &episode, &episodeEnd, &parent, &c.Watched, &c.Position, &c.Favorite, &c.Watchlist,
		&duration, &width, &height, &hdr, &c.Versions, &c.SeriesTitle, &parentPoster, &parentBackdrop, &firstEp,
		&d.OriginalTitle, &d.Tagline, &d.RatingSource, &d.ContentRating, &d.Status, &d.AirDate, &d.Trailer, &d.MatchState, &tmdb,
		&d.LibraryID, &d.State.PlayCount, &d.State.LastPlayedAt, &d.State.LastFileID, &d.State.Duration)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	c.Year, c.Season, c.Episode, c.EpisodeEnd, c.Runtime = intp(year), intp(season), intp(episode), intp(episodeEnd), intp(runtime)
	if rating.Valid {
		c.Rating = &rating.Float64
	}
	d.TmdbID = intp(tmdb)
	c.Duration = duration.Float64
	c.Resolution = ResolutionLabel(int(width.Int64), int(height.Int64))
	c.HDR = hdr.String
	c.Poster, c.Backdrop = ImageURL(poster), ImageURL(backdrop)
	if c.Kind == "series" && firstEp.Valid {
		c.Thumb = fmt.Sprintf("/api/v1/items/%d/thumb", firstEp.Int64)
	}
	json.Unmarshal([]byte(genres), &c.Genres)
	d.State.Position, d.State.Watched, d.State.Favorite, d.State.Watchlist = c.Position, c.Watched, c.Favorite, c.Watchlist
	if c.Kind == "episode" {
		c.SeriesID = parent.Int64
		if still != "" {
			c.Thumb = ImageURL(still)
		}
		if c.Backdrop == "" {
			c.Backdrop = ImageURL(parentBackdrop.String)
		}
		if c.Poster == "" {
			c.Poster = ImageURL(parentPoster.String)
		}
	}
	if c.Thumb == "" && c.Kind != "series" && c.Versions > 0 {
		c.Thumb = fmt.Sprintf("/api/v1/items/%d/thumb", c.ID)
	}
	if d.Files, err = s.Files(ctx, id); err != nil {
		return nil, err
	}
	if d.People, err = s.people(ctx, id); err != nil {
		return nil, err
	}
	if c.Kind == "series" {
		if d.Seasons, err = s.seasons(ctx, v, id); err != nil {
			return nil, err
		}
		d.NextUp, _ = s.NextUp(ctx, v, id)
		cc := []Card{d.Card}
		s.fillSeriesCounts(ctx, v, cc)
		d.Total, d.Unwatched = cc[0].Total, cc[0].Unwatched
	}
	if c.Kind == "movie" {
		d.Collections, _ = s.collectionMates(ctx, v, id)
		d.Similar, _ = s.similar(ctx, v, id, c.Genres)
	}
	return &d, nil
}

func (s *Store) people(ctx context.Context, id int64) ([]Person, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT name, role, character, profile FROM credits WHERE item_id=?
		ORDER BY CASE role WHEN 'director' THEN 0 WHEN 'creator' THEN 1 ELSE 2 END, ord`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Person{}
	for rows.Next() {
		var p Person
		var photo string
		if rows.Scan(&p.Name, &p.Role, &p.Character, &photo) == nil {
			p.Photo = ImageURL(photo)
			out = append(out, p)
		}
	}
	return out, rows.Err()
}

func (s *Store) seasons(ctx context.Context, v Viewer, seriesID int64) ([]Season, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT e.season, COALESCE(se.title,''), COALESCE(se.overview,''), COALESCE(se.poster,''),
		COUNT(*), SUM(CASE WHEN COALESCE(st.watched,0)=0 THEN 1 ELSE 0 END)
		FROM items e LEFT JOIN seasons se ON se.series_id=e.parent_id AND se.season=e.season
		LEFT JOIN user_state st ON st.item_id=e.id AND st.user_id=?
		WHERE e.parent_id=? AND EXISTS (SELECT 1 FROM media_files f WHERE f.item_id=e.id AND f.present=1)
		GROUP BY e.season ORDER BY CASE WHEN e.season=0 THEN 1 ELSE 0 END, e.season`, v.UserID, seriesID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Season{}
	for rows.Next() {
		var se Season
		var poster string
		if err := rows.Scan(&se.Number, &se.Title, &se.Overview, &poster, &se.Episodes, &se.Unwatched); err != nil {
			return nil, err
		}
		se.Poster = ImageURL(poster)
		if se.Title == "" {
			if se.Number == 0 {
				se.Title = "Épisodes spéciaux"
			} else {
				se.Title = fmt.Sprintf("Saison %d", se.Number)
			}
		}
		out = append(out, se)
	}
	return out, rows.Err()
}

// Episodes renvoie les épisodes d'une saison.
func (s *Store) Episodes(ctx context.Context, v Viewer, seriesID int64, season int) ([]Card, error) {
	where, args := v.Where("i")
	rows, err := s.DB.QueryContext(ctx, "SELECT "+cardColumns+" FROM items i"+cardJoins+
		" WHERE i.parent_id=? AND i.season=? AND "+presentMovie+" AND "+where+" ORDER BY i.episode",
		append([]any{v.UserID, seriesID, season}, args...)...)
	if err != nil {
		return nil, err
	}
	return scanCards(rows)
}

// NextUp : épisode à regarder ensuite — l'épisode en cours s'il y en a un, sinon le premier non vu
// après le dernier vu (saisons normales, puis spéciaux en dernier).
func (s *Store) NextUp(ctx context.Context, v Viewer, seriesID int64) (*Card, error) {
	where, args := v.Where("i")
	base := "SELECT " + cardColumns + " FROM items i" + cardJoins + " WHERE i.parent_id=? AND " + presentMovie + " AND " + where
	order := " ORDER BY CASE WHEN i.season=0 THEN 1 ELSE 0 END, i.season, i.episode"
	a := append([]any{v.UserID, seriesID}, args...)
	// 1. En cours.
	rows, err := s.DB.QueryContext(ctx, base+" AND COALESCE(st.watched,0)=0 AND COALESCE(st.position,0) > 0 ORDER BY st.last_played_at DESC LIMIT 1", a...)
	if err != nil {
		return nil, err
	}
	if cs, err := scanCards(rows); err == nil && len(cs) > 0 {
		return &cs[0], nil
	}
	// 2. Après le dernier épisode vu (hors spéciaux).
	var ls, le sql.NullInt64
	s.DB.QueryRowContext(ctx, `SELECT e.season, e.episode FROM items e JOIN user_state st ON st.item_id=e.id AND st.user_id=?
		WHERE e.parent_id=? AND st.watched=1 AND e.season > 0 ORDER BY st.last_played_at DESC, e.season DESC, e.episode DESC LIMIT 1`,
		v.UserID, seriesID).Scan(&ls, &le)
	if ls.Valid {
		rows, err = s.DB.QueryContext(ctx, base+" AND COALESCE(st.watched,0)=0 AND i.season > 0 AND (i.season > ? OR (i.season = ? AND i.episode > ?))"+order+" LIMIT 1",
			append(a, ls.Int64, ls.Int64, le.Int64)...)
		if err != nil {
			return nil, err
		}
		if cs, err := scanCards(rows); err == nil && len(cs) > 0 {
			return &cs[0], nil
		}
	}
	// 3. Premier non vu.
	rows, err = s.DB.QueryContext(ctx, base+" AND COALESCE(st.watched,0)=0"+order+" LIMIT 1", a...)
	if err != nil {
		return nil, err
	}
	if cs, err := scanCards(rows); err == nil && len(cs) > 0 {
		return &cs[0], nil
	}
	return nil, nil
}

// NextEpisode renvoie l'épisode qui suit (pour l'enchaînement automatique), ou nil.
func (s *Store) NextEpisode(ctx context.Context, v Viewer, episodeID int64) (*Card, error) {
	var parent, season, episode, episodeEnd sql.NullInt64
	if err := s.DB.QueryRowContext(ctx, `SELECT parent_id, season, episode, episode_end FROM items WHERE id=? AND kind='episode'`, episodeID).
		Scan(&parent, &season, &episode, &episodeEnd); err != nil {
		return nil, err
	}
	last := episode.Int64
	if episodeEnd.Valid {
		last = episodeEnd.Int64
	}
	where, args := v.Where("i")
	q := "SELECT " + cardColumns + " FROM items i" + cardJoins + " WHERE i.parent_id=? AND " + presentMovie + " AND " + where +
		" AND ((i.season = ? AND i.episode > ?) OR i.season > ?) AND i.season > 0 ORDER BY i.season, i.episode LIMIT 1"
	if season.Int64 == 0 {
		q = "SELECT " + cardColumns + " FROM items i" + cardJoins + " WHERE i.parent_id=? AND " + presentMovie + " AND " + where +
			" AND i.season = ? AND i.episode > ? AND ? = 0 ORDER BY i.episode LIMIT 1"
	}
	rows, err := s.DB.QueryContext(ctx, q, append(append([]any{v.UserID, parent.Int64}, args...), season.Int64, last, season.Int64)...)
	if err != nil {
		return nil, err
	}
	cs, err := scanCards(rows)
	if err != nil || len(cs) == 0 {
		return nil, err
	}
	return &cs[0], nil
}

func (s *Store) collectionMates(ctx context.Context, v Viewer, id int64) ([]Card, error) {
	where, args := v.Where("i")
	rows, err := s.DB.QueryContext(ctx, "SELECT "+cardColumns+" FROM items i"+cardJoins+
		" JOIN collection_items ci ON ci.item_id=i.id WHERE ci.collection_id IN (SELECT collection_id FROM collection_items WHERE item_id=?) AND "+
		presentMovie+" AND "+where+" ORDER BY COALESCE(i.year,0)", append([]any{v.UserID, id}, args...)...)
	if err != nil {
		return nil, err
	}
	cards, err := scanCards(rows)
	if err != nil || len(cards) < 2 {
		return nil, err
	}
	return cards, nil
}

// similar : suggestions locales simples — mêmes genres, non vus, les mieux notés.
func (s *Store) similar(ctx context.Context, v Viewer, id int64, genres []string) ([]Card, error) {
	if len(genres) == 0 {
		return nil, nil
	}
	where, args := v.Where("i")
	gj, _ := json.Marshal(genres)
	rows, err := s.DB.QueryContext(ctx, "SELECT "+cardColumns+" FROM items i"+cardJoins+
		" WHERE i.kind='movie' AND i.id<>? AND "+presentMovie+" AND "+where+
		" AND (SELECT COUNT(*) FROM json_each(i.genres) g WHERE g.value IN (SELECT value FROM json_each(?))) > 0"+
		" ORDER BY (SELECT COUNT(*) FROM json_each(i.genres) g WHERE g.value IN (SELECT value FROM json_each(?))) DESC,"+
		" COALESCE(st.watched,0), COALESCE(i.rating,0) DESC LIMIT 12",
		append(append([]any{v.UserID, id}, args...), string(gj), string(gj))...)
	if err != nil {
		return nil, err
	}
	return scanCards(rows)
}

// ItemVisible vérifie qu'un item est visible pour le profil (contrôle d'accès des routes de lecture).
func (s *Store) ItemVisible(ctx context.Context, v Viewer, id int64) bool {
	where, args := v.Where("i")
	var ok bool
	s.DB.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM items i WHERE i.id=? AND "+where+")", append([]any{id}, args...)...).Scan(&ok)
	return ok
}

// Collections renvoie les collections avec au moins deux œuvres visibles.
func (s *Store) Collections(ctx context.Context, v Viewer) ([]Card, error) {
	where, args := v.Where("i")
	rows, err := s.DB.QueryContext(ctx, `SELECT c.id, c.name, c.poster, c.backdrop, COUNT(*) n FROM collections c
		JOIN collection_items ci ON ci.collection_id=c.id JOIN items i ON i.id=ci.item_id
		WHERE `+presentMovie+` AND `+where+` GROUP BY c.id HAVING n >= 2 ORDER BY c.name`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Card{}
	for rows.Next() {
		var c Card
		var poster, backdrop string
		if rows.Scan(&c.ID, &c.Title, &poster, &backdrop, &c.Total) == nil {
			c.Kind = "collection"
			c.Poster, c.Backdrop = ImageURL(poster), ImageURL(backdrop)
			out = append(out, c)
		}
	}
	return out, rows.Err()
}

// CollectionItems renvoie les films d'une collection.
func (s *Store) CollectionItems(ctx context.Context, v Viewer, id int64) (string, []Card, error) {
	var name string
	if err := s.DB.QueryRowContext(ctx, `SELECT name FROM collections WHERE id=?`, id).Scan(&name); err != nil {
		return "", nil, ErrNotFound
	}
	cards, _, err := s.List(ctx, v, ListQuery{Kind: "movie", Collection: id, Sort: "year", Limit: 200})
	return name, cards, err
}
