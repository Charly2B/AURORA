package metadata

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"aurora/internal/catalog"
	"aurora/internal/parse"
)

// Enricher applique les métadonnées TMDB au catalogue.
type Enricher struct {
	DB     *sql.DB
	Client func(ctx context.Context) *Client // relit la clé en base à chaque usage ; nil si non configurée
}

// Progress : rapport d'avancement.
type Progress struct {
	Done, Total int
	Current     string
}

// retryFailedAfter : délai avant de retenter une recherche infructueuse.
const retryFailedAfter = 7 * 24 * time.Hour

// EnrichLibrary cherche les correspondances des œuvres jamais identifiées puis complète les épisodes.
// Films et séries sont traités quel que soit le type de la bibliothèque (une bibliothèque de films peut
// contenir des séries rangées au milieu des films).
func (e *Enricher) EnrichLibrary(ctx context.Context, libID int64, kind string, report func(Progress)) error {
	c := e.Client(ctx)
	if c == nil {
		return nil // fonctionnement purement local : rien à faire
	}
	var firstErr error
	for _, itemKind := range []string{"movie", "series"} {
		err := e.enrichKind(ctx, c, libID, itemKind, report)
		if errors.Is(err, context.Canceled) {
			return err
		}
		if err != nil {
			if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "injoignable") {
				return err // inutile d'insister : clé refusée ou pas d'Internet
			}
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	if err := e.refreshNewEpisodes(ctx, c, libID); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

func (e *Enricher) enrichKind(ctx context.Context, c *Client, libID int64, itemKind string, report func(Progress)) error {
	tmdbKind := "movie"
	if itemKind == "series" {
		tmdbKind = "tv"
	}
	rows, err := e.DB.QueryContext(ctx, `SELECT i.id, i.title, COALESCE(i.year, 0) FROM items i
		WHERE i.library_id=? AND i.kind=? AND (i.match_state='local' OR (i.match_state='failed' AND COALESCE(i.metadata_at,0) < ?))
		ORDER BY i.id`, libID, itemKind, time.Now().Add(-retryFailedAfter).Unix())
	if err != nil {
		return err
	}
	type todo struct {
		id    int64
		title string
		year  int
	}
	var list []todo
	for rows.Next() {
		var t todo
		if rows.Scan(&t.id, &t.title, &t.year) == nil {
			list = append(list, t)
		}
	}
	rows.Close()

	var firstErr error
	var en *Client // client anglais (second recours), créé une fois
	for i, t := range list {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		report(Progress{Done: i, Total: len(list), Current: t.title})
		id, state, err := e.match(ctx, c, &en, tmdbKind, t.title, t.year)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return err
			}
			if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "injoignable") {
				return err
			}
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if id == 0 {
			e.DB.ExecContext(ctx, `UPDATE items SET match_state=?, metadata_at=? WHERE id=? AND match_state IN ('local','failed')`,
				state, time.Now().Unix(), t.id)
			continue
		}
		if itemKind == "movie" {
			_, err = e.ApplyMovie(ctx, c, t.id, id, "matched")
		} else {
			_, err = e.ApplySeries(ctx, c, t.id, id, "matched")
		}
		if err != nil {
			slog.Warn("métadonnées non appliquées", "item", t.id, "err", err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	report(Progress{Done: len(list), Total: len(list)})
	return firstErr
}

// match cherche la fiche d'une œuvre : langue configurée, puis anglais (fichiers nommés avec le titre
// original), puis titre complet quand l'« année » en fait partie (« Wonder Woman 1984 »). Même règle de
// prudence à chaque étape (Choose).
func (e *Enricher) match(ctx context.Context, c *Client, en **Client, tmdbKind, title string, year int) (int, string, error) {
	results, err := c.Search(ctx, tmdbKind, title, 0)
	if err != nil {
		return 0, "", err
	}
	id, state := Choose(results, title, year)
	if id == 0 && !strings.HasPrefix(c.Language, "en") {
		if *en == nil {
			*en = c.WithLanguage("en-US")
		}
		if res, err := (*en).Search(ctx, tmdbKind, title, 0); err == nil {
			if id2, state2 := Choose(res, title, year); id2 != 0 || state == "failed" {
				id, state = id2, state2
			}
		}
	}
	if id == 0 && year > 0 {
		full := fmt.Sprintf("%s %d", title, year)
		if res, err := c.Search(ctx, tmdbKind, full, 0); err == nil {
			if id2, _ := Choose(res, full, 0); id2 != 0 {
				id, state = id2, "matched"
			}
		}
	}
	return id, state, nil
}

// Choose applique la règle de prudence : on n'associe automatiquement que si le titre correspond
// exactement (titre ou titre original, sans accents ni ponctuation) et que l'année lève toute ambiguïté.
// Sinon l'œuvre reste « ambiguous » et l'utilisateur choisit dans l'interface de correction.
func Choose(results []SearchResult, title string, year int) (int, string) {
	if len(results) == 0 {
		return 0, "failed"
	}
	norm := parse.Normalize(title)
	var exact []SearchResult
	for _, r := range results {
		if parse.Normalize(r.Title) == norm || parse.Normalize(r.OriginalTitle) == norm {
			exact = append(exact, r)
		}
	}
	if year > 0 {
		var same, near []SearchResult
		for _, r := range exact {
			switch d := r.Year - year; {
			case d == 0:
				same = append(same, r)
			case d == 1 || d == -1:
				near = append(near, r)
			}
		}
		if len(same) == 1 {
			return same[0].ID, "matched"
		}
		if len(same) == 0 && len(near) == 1 {
			return near[0].ID, "matched" // sortie décalée d'un an selon le pays
		}
		return 0, "ambiguous"
	}
	if len(exact) == 1 {
		return exact[0].ID, "matched"
	}
	return 0, "ambiguous"
}

func genresJSON(gs []struct {
	Name string `json:"name"`
}) string {
	names := make([]string, 0, len(gs))
	for _, g := range gs {
		names = append(names, g.Name)
	}
	b, _ := json.Marshal(names)
	return string(b)
}

func nullInt(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

// ApplyMovie associe un item film à une fiche TMDB. Si un autre film de la bibliothèque porte déjà cette
// fiche, les deux sont fusionnés (fichiers = versions d'une même œuvre). Renvoie l'id final de l'item.
func (e *Enricher) ApplyMovie(ctx context.Context, c *Client, itemID int64, tmdbID int, state string) (int64, error) {
	d, err := c.Movie(ctx, tmdbID)
	if err != nil {
		return 0, err
	}
	poster, _ := c.Image(ctx, "w500", d.PosterPath)
	backdrop, _ := c.Image(ctx, "w1280", d.BackdropPath)
	cert, age := d.Certification()
	var rating any
	ratingSrc := ""
	if d.VoteCount >= 10 {
		rating, ratingSrc = d.VoteAverage, "TMDB"
	}
	var runtime any
	if d.Runtime > 0 {
		runtime = d.Runtime
	}
	var year any
	if y := yearOf(d.ReleaseDate); y > 0 {
		year = y
	}
	cast := e.profiles(ctx, c, d.Credits.Cast, 15)

	tx, err := e.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	target, err := mergeDuplicate(ctx, tx, itemID, "movie", tmdbID)
	if err != nil {
		return 0, err
	}
	_, err = tx.ExecContext(ctx, `UPDATE items SET title=?, sort_title=?, original_title=?, year=?, overview=?, tagline=?,
		runtime=?, genres=?, rating=?, rating_source=?, content_rating=?, min_age=?, tmdb_id=?, poster=?, backdrop=?, trailer=?,
		match_state=?, metadata_at=?, updated_at=? WHERE id=?`,
		d.Title, parse.SortTitle(d.Title), d.OriginalTitle, year, d.Overview, d.Tagline, runtime, genresJSON(d.Genres),
		rating, ratingSrc, cert, nullInt(age), tmdbID, poster, backdrop, d.Trailer(), state, time.Now().Unix(), time.Now().Unix(), target)
	if err != nil {
		return 0, err
	}
	var directors []Credit
	for _, cr := range d.Credits.Crew {
		if cr.Job == "Director" {
			directors = append(directors, cr)
		}
	}
	if err := replaceCredits(ctx, tx, target, cast, directors, "director"); err != nil {
		return 0, err
	}
	if d.BelongsTo != nil {
		cp, _ := c.Image(ctx, "w500", d.BelongsTo.PosterPath)
		cb, _ := c.Image(ctx, "w1280", d.BelongsTo.BackdropPath)
		if _, err := tx.ExecContext(ctx, `INSERT INTO collections(name, kind, tmdb_id, poster, backdrop, created_at) VALUES (?,?,?,?,?,?)
			ON CONFLICT(tmdb_id) DO UPDATE SET name=excluded.name, poster=excluded.poster, backdrop=excluded.backdrop`,
			d.BelongsTo.Name, "tmdb", d.BelongsTo.ID, cp, cb, time.Now().Unix()); err != nil {
			return 0, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO collection_items(collection_id, item_id, ord)
			SELECT id, ?, ? FROM collections WHERE tmdb_id=?`, target, nullInt(intPtr(yearOf(d.ReleaseDate))), d.BelongsTo.ID); err != nil {
			return 0, err
		}
	}
	if err := catalog.ReindexItem(ctx, tx, target); err != nil {
		return 0, err
	}
	return target, tx.Commit()
}

func intPtr(v int) *int { return &v }

// profiles télécharge les portraits des premiers rôles (cache local, utile hors-ligne).
func (e *Enricher) profiles(ctx context.Context, c *Client, cast []Credit, n int) []Credit {
	if len(cast) > n {
		cast = cast[:n]
	}
	out := make([]Credit, len(cast))
	for i, cr := range cast {
		out[i] = cr
		out[i].ProfilePath, _ = c.Image(ctx, "w185", cr.ProfilePath)
	}
	return out
}

func replaceCredits(ctx context.Context, tx *sql.Tx, itemID int64, cast, crew []Credit, crewRole string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM credits WHERE item_id=?`, itemID); err != nil {
		return err
	}
	for i, cr := range cast {
		if _, err := tx.ExecContext(ctx, `INSERT INTO credits(item_id, name, role, character, ord, profile) VALUES (?,?,?,?,?,?)`,
			itemID, cr.Name, "actor", cr.Character, i, cr.ProfilePath); err != nil {
			return err
		}
	}
	for i, cr := range crew {
		if _, err := tx.ExecContext(ctx, `INSERT INTO credits(item_id, name, role, ord) VALUES (?,?,?,?)`,
			itemID, cr.Name, crewRole, i); err != nil {
			return err
		}
	}
	return nil
}

// mergeDuplicate : si un autre item de la même bibliothèque porte déjà tmdbID, on y rattache les fichiers
// (et les épisodes pour une série) puis on retire l'item devenu vide. Aucun fichier vidéo n'est touché.
func mergeDuplicate(ctx context.Context, tx *sql.Tx, itemID int64, kind string, tmdbID int) (int64, error) {
	var other int64
	err := tx.QueryRowContext(ctx, `SELECT o.id FROM items o JOIN items s ON s.id=? WHERE o.library_id=s.library_id
		AND o.kind=? AND o.tmdb_id=? AND o.id<>s.id`, itemID, kind, tmdbID).Scan(&other)
	if errors.Is(err, sql.ErrNoRows) {
		return itemID, nil
	}
	if err != nil {
		return 0, err
	}
	if err := MergeItems(ctx, tx, itemID, other); err != nil {
		return 0, err
	}
	return other, nil
}

// MergeItems fusionne src dans dst (même type). L'état de lecture est conservé (le plus avancé l'emporte).
func MergeItems(ctx context.Context, tx *sql.Tx, src, dst int64) error {
	var kind string
	if err := tx.QueryRowContext(ctx, `SELECT kind FROM items WHERE id=?`, src).Scan(&kind); err != nil {
		return err
	}
	if kind == "series" {
		rows, err := tx.QueryContext(ctx, `SELECT id, season, episode FROM items WHERE parent_id=?`, src)
		if err != nil {
			return err
		}
		type ep struct {
			id   int64
			s, e int
		}
		var eps []ep
		for rows.Next() {
			var x ep
			if rows.Scan(&x.id, &x.s, &x.e) == nil {
				eps = append(eps, x)
			}
		}
		rows.Close()
		for _, x := range eps {
			var existing int64
			err := tx.QueryRowContext(ctx, `SELECT id FROM items WHERE parent_id=? AND season=? AND episode=?`, dst, x.s, x.e).Scan(&existing)
			switch {
			case err == nil:
				if err := MergeItems(ctx, tx, x.id, existing); err != nil {
					return err
				}
			case errors.Is(err, sql.ErrNoRows):
				if _, err := tx.ExecContext(ctx, `UPDATE items SET parent_id=?, match_key=? WHERE id=?`,
					dst, fmt.Sprintf("%d|%d|%d", dst, x.s, x.e), x.id); err != nil {
					return err
				}
			default:
				return err
			}
		}
	}
	stmts := []string{
		`UPDATE media_files SET item_id=? WHERE item_id=?`,
		`INSERT INTO user_state(user_id, item_id, position, duration, watched, play_count, last_played_at, last_file_id, favorite, watchlist, watchlist_at)
		 SELECT user_id, ?, position, duration, watched, play_count, last_played_at, last_file_id, favorite, watchlist, watchlist_at
		 FROM user_state WHERE item_id=? AND true
		 ON CONFLICT(user_id, item_id) DO UPDATE SET
		   watched = MAX(watched, excluded.watched), favorite = MAX(favorite, excluded.favorite),
		   watchlist = MAX(watchlist, excluded.watchlist), play_count = play_count + excluded.play_count,
		   position = CASE WHEN COALESCE(excluded.last_played_at,0) > COALESCE(last_played_at,0) THEN excluded.position ELSE position END,
		   last_played_at = MAX(COALESCE(last_played_at,0), COALESCE(excluded.last_played_at,0))`,
		`INSERT OR IGNORE INTO collection_items(collection_id, item_id, ord) SELECT collection_id, ?, ord FROM collection_items WHERE item_id=?`,
		// Retraits de « Continuer à regarder » : reportés (sinon effacés en cascade, l'œuvre reviendrait).
		`INSERT INTO continue_hidden(user_id, item_id, hidden_at) SELECT user_id, ?, hidden_at FROM continue_hidden WHERE item_id=? AND true
		 ON CONFLICT(user_id, item_id) DO UPDATE SET hidden_at = MAX(hidden_at, excluded.hidden_at)`,
	}
	for _, s := range stmts {
		if _, err := tx.ExecContext(ctx, s, dst, src); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM items_fts WHERE rowid=?`, src); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `DELETE FROM items WHERE id=?`, src)
	return err
}

// ApplySeries associe une série à TMDB et complète saisons et épisodes présents.
func (e *Enricher) ApplySeries(ctx context.Context, c *Client, itemID int64, tmdbID int, state string) (int64, error) {
	d, err := c.TV(ctx, tmdbID)
	if err != nil {
		return 0, err
	}
	poster, _ := c.Image(ctx, "w500", d.PosterPath)
	backdrop, _ := c.Image(ctx, "w1280", d.BackdropPath)
	cert, age := d.Certification()
	var rating any
	ratingSrc := ""
	if d.VoteCount >= 10 {
		rating, ratingSrc = d.VoteAverage, "TMDB"
	}
	var year any
	if y := yearOf(d.FirstAirDate); y > 0 {
		year = y
	}
	var runtime any
	if len(d.EpisodeRunTime) > 0 && d.EpisodeRunTime[0] > 0 {
		runtime = d.EpisodeRunTime[0]
	}
	cast := e.profiles(ctx, c, d.Credits.Cast, 15)
	seasonPosters := map[int]string{}
	for _, s := range d.Seasons {
		seasonPosters[s.SeasonNumber], _ = c.Image(ctx, "w500", s.PosterPath)
	}

	tx, err := e.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	target, err := mergeDuplicate(ctx, tx, itemID, "series", tmdbID)
	if err != nil {
		return 0, err
	}
	now := time.Now().Unix()
	_, err = tx.ExecContext(ctx, `UPDATE items SET title=?, sort_title=?, original_title=?, year=?, overview=?, tagline=?,
		runtime=?, genres=?, rating=?, rating_source=?, content_rating=?, min_age=?, status=?, tmdb_id=?, poster=?, backdrop=?,
		match_state=?, metadata_at=?, updated_at=? WHERE id=?`,
		d.Name, parse.SortTitle(d.Name), d.OriginalName, year, d.Overview, d.Tagline, runtime, genresJSON(d.Genres),
		rating, ratingSrc, cert, nullInt(age), seriesStatus(d.Status), tmdbID, poster, backdrop, state, now, now, target)
	if err != nil {
		return 0, err
	}
	if err := replaceCredits(ctx, tx, target, cast, d.CreatedBy, "creator"); err != nil {
		return 0, err
	}
	for _, s := range d.Seasons {
		if _, err := tx.ExecContext(ctx, `INSERT INTO seasons(series_id, season, title, overview, poster, air_date) VALUES (?,?,?,?,?,?)
			ON CONFLICT(series_id, season) DO UPDATE SET title=excluded.title, overview=excluded.overview,
			poster=excluded.poster, air_date=excluded.air_date`,
			target, s.SeasonNumber, s.Name, s.Overview, seasonPosters[s.SeasonNumber], s.AirDate); err != nil {
			return 0, err
		}
	}
	// Les épisodes héritent de la classification de la série (TMDB n'en fournit pas par épisode).
	if _, err := tx.ExecContext(ctx, `UPDATE items SET content_rating=?, min_age=? WHERE parent_id=?`, cert, nullInt(age), target); err != nil {
		return 0, err
	}
	if err := catalog.ReindexItem(ctx, tx, target); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return target, e.fillEpisodes(ctx, c, target, tmdbID, false)
}

func seriesStatus(s string) string {
	switch s {
	case "Returning Series":
		return "En cours"
	case "Ended":
		return "Terminée"
	case "Canceled":
		return "Annulée"
	case "In Production", "Planned":
		return "En production"
	}
	return s
}

// fillEpisodes complète titres, résumés et vignettes des épisodes présents.
// onlyMissing : ne traiter que les saisons ayant des épisodes jamais enrichis.
func (e *Enricher) fillEpisodes(ctx context.Context, c *Client, seriesID int64, tmdbID int, onlyMissing bool) error {
	q := `SELECT DISTINCT season FROM items WHERE parent_id=?`
	if onlyMissing {
		q += ` AND metadata_at IS NULL`
	}
	rows, err := e.DB.QueryContext(ctx, q, seriesID)
	if err != nil {
		return err
	}
	var seasons []int
	for rows.Next() {
		var s int
		if rows.Scan(&s) == nil {
			seasons = append(seasons, s)
		}
	}
	rows.Close()
	for _, sn := range seasons {
		sd, err := c.Season(ctx, tmdbID, sn)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return err
		}
		for _, ep := range sd.Episodes {
			var id int64
			if e.DB.QueryRowContext(ctx, `SELECT id FROM items WHERE parent_id=? AND season=? AND episode=?`, seriesID, sn, ep.EpisodeNumber).Scan(&id) != nil {
				continue
			}
			still, _ := c.Image(ctx, "w300", ep.StillPath)
			var runtime any
			if ep.Runtime > 0 {
				runtime = ep.Runtime
			}
			title := ep.Name
			if title == "" {
				title = fmt.Sprintf("Épisode %d", ep.EpisodeNumber)
			}
			if _, err := e.DB.ExecContext(ctx, `UPDATE items SET title=?, sort_title=?, overview=?, air_date=?, runtime=?, still=?,
				match_state=CASE WHEN match_state='locked' THEN 'locked' ELSE 'matched' END, metadata_at=?, updated_at=?,
				content_rating=(SELECT content_rating FROM items WHERE id=?), min_age=(SELECT min_age FROM items WHERE id=?)
				WHERE id=?`,
				title, parse.SortTitle(title), ep.Overview, ep.AirDate, runtime, still, time.Now().Unix(), time.Now().Unix(),
				seriesID, seriesID, id); err != nil {
				return err
			}
			catalog.ReindexItem(ctx, e.DB, id)
		}
	}
	return nil
}

// refreshNewEpisodes complète les épisodes ajoutés depuis l'association de leur série.
func (e *Enricher) refreshNewEpisodes(ctx context.Context, c *Client, libID int64) error {
	rows, err := e.DB.QueryContext(ctx, `SELECT DISTINCT s.id, s.tmdb_id FROM items s JOIN items ep ON ep.parent_id=s.id
		WHERE s.library_id=? AND s.kind='series' AND s.tmdb_id IS NOT NULL AND ep.metadata_at IS NULL`, libID)
	if err != nil {
		return err
	}
	type pair struct {
		id   int64
		tmdb int
	}
	var list []pair
	for rows.Next() {
		var p pair
		if rows.Scan(&p.id, &p.tmdb) == nil {
			list = append(list, p)
		}
	}
	rows.Close()
	for _, p := range list {
		if err := e.fillEpisodes(ctx, c, p.id, p.tmdb, true); err != nil {
			return err
		}
	}
	return nil
}
