package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	"aurora/internal/parse"
)

// Section : rangée de l'accueil.
type Section struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	LibraryID int64  `json:"libraryId,omitempty"` // rangée propre à un dossier
	Kind      string `json:"kind,omitempty"`
	Items     []Card `json:"items"`
}

// LibraryRef : bibliothèque contenant au moins une œuvre d'un type donné.
type LibraryRef struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Movies int    `json:"movies"`
	Series int    `json:"series"`
}

// librariesWith : bibliothèques visibles contenant au moins une œuvre du type, dans l'ordre de création.
func (s *Store) librariesWith(ctx context.Context, v Viewer, kind string) ([]LibraryRef, error) {
	all, err := s.Libraries(ctx, v)
	if err != nil {
		return nil, err
	}
	var out []LibraryRef
	for _, l := range all {
		if (kind == "movie" && l.Movies > 0) || (kind == "series" && l.Series > 0) {
			out = append(out, l)
		}
	}
	return out, nil
}

// Libraries : bibliothèques visibles avec le nombre de films et de séries présents (respecte le profil).
func (s *Store) Libraries(ctx context.Context, v Viewer) ([]LibraryRef, error) {
	where, args := v.Where("i")
	q := `SELECT l.id, l.name,
		(SELECT COUNT(*) FROM items i WHERE i.library_id=l.id AND i.kind='movie' AND ` + presentMovie + ` AND ` + where + `),
		(SELECT COUNT(*) FROM items i WHERE i.library_id=l.id AND i.kind='series' AND ` + presentSeries + ` AND ` + where + `)
		FROM libraries l`
	all := append(append([]any{}, args...), args...)
	if !v.AllLibraries {
		q += ` WHERE l.id IN (SELECT library_id FROM user_libraries WHERE user_id=?)`
		all = append(all, v.UserID)
	}
	rows, err := s.DB.QueryContext(ctx, q+` ORDER BY l.id`, all...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []LibraryRef{}
	for rows.Next() {
		var l LibraryRef
		if err := rows.Scan(&l.ID, &l.Name, &l.Movies, &l.Series); err != nil {
			return nil, err
		}
		if v.MaxAge != nil && l.Movies+l.Series == 0 {
			continue // profil restreint : pas même le nom d'un dossier dont tout lui est masqué
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// HomeSectionIDs : sections disponibles, dans l'ordre par défaut.
var HomeSectionIDs = []struct{ ID, Title string }{
	{"continue", "Continuer à regarder"},
	{"nextup", "Prochain épisode"},
	{"recent_movies", "Films récemment ajoutés"},
	{"recent_series", "Séries récemment ajoutées"},
	{"favorites", "Favoris"},
	{"watchlist", "Ma liste"},
	{"collections", "Collections"},
	{"unwatched_movies", "Films non visionnés"},
	{"series_in_progress", "Séries en cours"},
	{"suggestions", "Suggestions de ma bibliothèque"},
}

// HomePref : préférence d'affichage d'une section.
type HomePref struct {
	ID      string `json:"id"`
	Visible bool   `json:"visible"`
}

// Home : contenu de la page d'accueil.
type Home struct {
	Hero     *Card     `json:"hero,omitempty"`
	Sections []Section `json:"sections"`
}

// ResolveHomePrefs complète les préférences enregistrées avec les sections connues.
func ResolveHomePrefs(prefs []HomePref) []HomePref {
	seen := map[string]bool{}
	var out []HomePref
	for _, p := range prefs {
		for _, s := range HomeSectionIDs {
			if s.ID == p.ID && !seen[p.ID] {
				out = append(out, p)
				seen[p.ID] = true
			}
		}
	}
	for _, s := range HomeSectionIDs {
		if !seen[s.ID] {
			out = append(out, HomePref{ID: s.ID, Visible: s.ID != "series_in_progress"})
		}
	}
	return out
}

// notHidden : œuvre (film, ou série d'un épisode) non retirée de « Continuer » depuis sa dernière lecture.
// Un seul argument : l'identifiant du profil.
const notHidden = ` AND NOT EXISTS (SELECT 1 FROM continue_hidden h WHERE h.user_id=? AND h.item_id=COALESCE(i.parent_id, i.id)
	AND h.hidden_at >= COALESCE(st.last_played_at, 0))`

// Home construit l'accueil. Les sections vides ne sont pas renvoyées.
func (s *Store) Home(ctx context.Context, v Viewer, prefs []HomePref) (*Home, error) {
	h := &Home{Sections: []Section{}}
	titles := map[string]string{}
	for _, x := range HomeSectionIDs {
		titles[x.ID] = x.Title
	}
	where, wargs := v.Where("i")
	q := func(extra string, args ...any) ([]Card, error) {
		rows, err := s.DB.QueryContext(ctx, "SELECT "+cardColumns+" FROM items i"+cardJoins+" WHERE "+where+" AND "+extra,
			append(append([]any{v.UserID}, wargs...), args...)...)
		if err != nil {
			return nil, err
		}
		return scanCards(rows)
	}
	var continueCards []Card
	for _, p := range ResolveHomePrefs(prefs) {
		if !p.Visible {
			continue
		}
		var items []Card
		var err error
		switch p.ID {
		case "continue":
			// Sauf œuvre retirée par le profil (film, ou série de l'épisode) depuis sa dernière lecture.
			items, err = q(presentMovie+` AND i.kind IN ('movie','episode') AND COALESCE(st.watched,0)=0 AND COALESCE(st.position,0) > 0`+
				notHidden+` ORDER BY st.last_played_at DESC LIMIT 20`, v.UserID)
			continueCards = items
		case "nextup":
			items, err = s.nextUpList(ctx, v, continueCards)
		case "recent_movies", "recent_series":
			// Une rangée par dossier (« Séries en cours », « Séries 2021 »…) : les bibliothèques ne sont pas
			// mélangées. Un seul dossier : la rangée garde son titre générique.
			kind, present, order := "movie", presentMovie, "i.added_at"
			if p.ID == "recent_series" {
				kind, present, order = "series", presentSeries, "(SELECT MAX(e.added_at) FROM items e WHERE e.parent_id=i.id)"
			}
			libs, err := s.librariesWith(ctx, v, kind)
			if err != nil {
				return nil, err
			}
			for _, lib := range libs {
				cards, err := q(present+` AND i.kind='`+kind+`' AND i.library_id=? ORDER BY `+order+` DESC LIMIT 20`, lib.ID)
				if err != nil {
					return nil, err
				}
				if kind == "series" {
					s.fillSeriesCounts(ctx, v, cards)
				}
				title := titles[p.ID]
				if len(libs) > 1 {
					title = lib.Name
				}
				if len(cards) > 0 {
					h.Sections = append(h.Sections, Section{ID: fmt.Sprintf("%s_%d", p.ID, lib.ID), Title: title,
						LibraryID: lib.ID, Kind: kind, Items: cards})
				}
			}
			continue
		case "favorites":
			items, err = q(`COALESCE(st.favorite,0)=1 AND ((i.kind='movie' AND ` + presentMovie + `) OR (i.kind='series' AND ` + presentSeries + `))
				ORDER BY i.sort_title LIMIT 30`)
		case "watchlist":
			items, err = q(`COALESCE(st.watchlist,0)=1 AND ((i.kind='movie' AND ` + presentMovie + `) OR (i.kind='series' AND ` + presentSeries + `))
				ORDER BY st.watchlist_at DESC LIMIT 30`)
		case "collections":
			items, err = s.Collections(ctx, v)
		case "unwatched_movies":
			// Ordre mélangé mais stable sur la journée (pas de « roulette » à chaque rechargement).
			day := time.Now().YearDay()
			items, err = q(presentMovie+` AND i.kind='movie' AND COALESCE(st.watched,0)=0 AND COALESCE(st.position,0)=0
				ORDER BY (i.id * ?) % 1009 LIMIT 20`, day+7)
		case "series_in_progress":
			items, err = q(presentSeries+` AND i.kind='series' AND EXISTS (SELECT 1 FROM items e JOIN user_state es ON es.item_id=e.id AND es.user_id=?
				WHERE e.parent_id=i.id AND es.watched=1) AND EXISTS (SELECT 1 FROM items e LEFT JOIN user_state es ON es.item_id=e.id AND es.user_id=?
				WHERE e.parent_id=i.id AND COALESCE(es.watched,0)=0)
				AND NOT EXISTS (SELECT 1 FROM continue_hidden h WHERE h.user_id=? AND h.item_id=i.id AND h.hidden_at >= COALESCE(
					(SELECT MAX(es.last_played_at) FROM items e JOIN user_state es ON es.item_id=e.id AND es.user_id=? WHERE e.parent_id=i.id), 0))
				LIMIT 20`, v.UserID, v.UserID, v.UserID, v.UserID)
			s.fillSeriesCounts(ctx, v, items)
		case "suggestions":
			items, err = s.suggestions(ctx, v)
		}
		if err != nil {
			return nil, err
		}
		if len(items) > 0 {
			h.Sections = append(h.Sections, Section{ID: p.ID, Title: titles[p.ID], Items: items})
		}
	}
	h.Hero = s.hero(ctx, v, continueCards, q)
	return h, nil
}

func (s *Store) hero(ctx context.Context, v Viewer, cont []Card, q func(string, ...any) ([]Card, error)) *Card {
	for _, c := range cont {
		if c.Backdrop != "" && c.Kind == "movie" {
			return &c
		}
	}
	day := time.Now().YearDay()
	// Un film retiré de « Continuer » n'est pas remis en avant ici (avec « Reprendre »).
	cs, err := q(presentMovie+` AND i.kind='movie' AND i.backdrop<>'' AND COALESCE(st.watched,0)=0`+notHidden+`
		ORDER BY i.added_at > ? DESC, (i.id * ?) % 211 LIMIT 1`, v.UserID, time.Now().AddDate(0, 0, -30).Unix(), day+3)
	if err == nil && len(cs) > 0 {
		return &cs[0]
	}
	cs, err = q(presentMovie+` AND i.kind='movie'`+notHidden+` ORDER BY i.added_at DESC LIMIT 1`, v.UserID)
	if err == nil && len(cs) > 0 {
		return &cs[0]
	}
	return nil
}

// nextUpList : prochain épisode de chaque série commencée, par activité récente.
func (s *Store) nextUpList(ctx context.Context, v Viewer, exclude []Card) ([]Card, error) {
	// Séries retirées par le profil depuis leur dernière lecture : absentes (elles reviennent à la prochaine).
	rows, err := s.DB.QueryContext(ctx, `SELECT x.parent_id, x.last FROM (
			SELECT e.parent_id, MAX(st.last_played_at) last FROM user_state st JOIN items e ON e.id=st.item_id
			WHERE st.user_id=? AND e.kind='episode' AND st.last_played_at IS NOT NULL GROUP BY e.parent_id) x
		WHERE NOT EXISTS (SELECT 1 FROM continue_hidden h WHERE h.user_id=? AND h.item_id=x.parent_id AND h.hidden_at >= x.last)
		ORDER BY x.last DESC LIMIT 30`, v.UserID, v.UserID)
	if err != nil {
		return nil, err
	}
	var series []int64
	for rows.Next() {
		var id, last int64
		if rows.Scan(&id, &last) == nil {
			series = append(series, id)
		}
	}
	rows.Close()
	skip := map[int64]bool{}
	for _, c := range exclude {
		skip[c.ID] = true
	}
	out := []Card{}
	for _, sid := range series {
		c, err := s.NextUp(ctx, v, sid)
		if err != nil || c == nil || skip[c.ID] {
			continue
		}
		out = append(out, *c)
		if len(out) >= 20 {
			break
		}
	}
	return out, nil
}

// suggestions : films non vus partageant les genres des œuvres regardées ou favorites du profil.
func (s *Store) suggestions(ctx context.Context, v Viewer) ([]Card, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT g.value, COUNT(*) n FROM user_state st JOIN items i ON i.id = COALESCE(
		(SELECT parent_id FROM items WHERE id=st.item_id AND kind='episode'), st.item_id), json_each(i.genres) g
		WHERE st.user_id=? AND (st.watched=1 OR st.favorite=1) GROUP BY g.value ORDER BY n DESC LIMIT 3`, v.UserID)
	if err != nil {
		return nil, err
	}
	var genres []string
	for rows.Next() {
		var g string
		var n int
		if rows.Scan(&g, &n) == nil {
			genres = append(genres, g)
		}
	}
	rows.Close()
	if len(genres) == 0 {
		return nil, nil
	}
	gj, _ := json.Marshal(genres)
	where, args := v.Where("i")
	rows, err = s.DB.QueryContext(ctx, "SELECT "+cardColumns+" FROM items i"+cardJoins+" WHERE i.kind='movie' AND "+presentMovie+" AND "+where+
		` AND COALESCE(st.watched,0)=0 AND COALESCE(st.position,0)=0
		AND EXISTS (SELECT 1 FROM json_each(i.genres) g WHERE g.value IN (SELECT value FROM json_each(?)))
		ORDER BY COALESCE(i.rating,0) DESC LIMIT 20`, append(append([]any{v.UserID}, args...), string(gj))...)
	if err != nil {
		return nil, err
	}
	return scanCards(rows)
}

// SearchResults : résultats regroupés.
type SearchResults struct {
	Movies   []Card   `json:"movies"`
	Series   []Card   `json:"series"`
	Episodes []Card   `json:"episodes"`
	People   []string `json:"people"`
	Fuzzy    bool     `json:"fuzzy"` // résultats approchants (faute de frappe probable)
}

// ftsQuery transforme une saisie libre en requête FTS5 sûre : chaque mot devient un préfixe entre guillemets.
func ftsQuery(q string) string {
	var parts []string
	for _, w := range strings.Fields(parse.Normalize(q)) {
		w = strings.Map(func(r rune) rune {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				return r
			}
			return -1
		}, w)
		if w != "" {
			parts = append(parts, `"`+w+`"*`)
		}
	}
	return strings.Join(parts, " ")
}

// Search : recherche locale (aucun accès disque aux vidéos, aucune requête Internet).
func (s *Store) Search(ctx context.Context, v Viewer, q string, limit int) (*SearchResults, error) {
	res := &SearchResults{Movies: []Card{}, Series: []Card{}, Episodes: []Card{}, People: []string{}}
	fq := ftsQuery(q)
	if fq == "" {
		return res, nil
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	where, args := v.Where("i")
	rows, err := s.DB.QueryContext(ctx, "SELECT "+cardColumns+" FROM items_fts JOIN items i ON i.id=items_fts.rowid"+cardJoins+
		" WHERE items_fts MATCH ? AND "+where+" AND ((i.kind='movie' AND "+presentMovie+") OR (i.kind='series' AND "+presentSeries+
		") OR (i.kind='episode' AND "+presentMovie+")) ORDER BY bm25(items_fts, 10.0, 6.0, 2.0, 1.0) LIMIT ?",
		append(append([]any{v.UserID, fq}, args...), limit*3)...)
	if err != nil {
		return nil, err
	}
	cards, err := scanCards(rows)
	if err != nil {
		return nil, err
	}
	if len(cards) == 0 {
		cards, err = s.fuzzy(ctx, v, q, limit)
		if err != nil {
			return nil, err
		}
		res.Fuzzy = len(cards) > 0
	}
	for _, c := range cards {
		switch c.Kind {
		case "movie":
			if len(res.Movies) < limit {
				res.Movies = append(res.Movies, c)
			}
		case "series":
			if len(res.Series) < limit {
				res.Series = append(res.Series, c)
			}
		case "episode":
			if len(res.Episodes) < limit {
				res.Episodes = append(res.Episodes, c)
			}
		}
	}
	s.fillSeriesCounts(ctx, v, res.Series)
	res.People = s.matchPeople(ctx, fq, q)
	return res, nil
}

// matchPeople : noms dont un mot commence par chacun des mots saisis (sans accents).
func (s *Store) matchPeople(ctx context.Context, fq, q string) []string {
	rows, err := s.DB.QueryContext(ctx, `SELECT DISTINCT c.name FROM items_fts JOIN credits c ON c.item_id=items_fts.rowid
		WHERE items_fts MATCH ? LIMIT 400`, "people : ("+fq+")")
	if err != nil {
		return []string{}
	}
	defer rows.Close()
	words := strings.Fields(parse.Normalize(q))
	out := []string{}
	for rows.Next() {
		var name string
		if rows.Scan(&name) != nil {
			continue
		}
		nameWords := strings.Fields(parse.Normalize(name))
		all := true
		for _, w := range words {
			hit := false
			for _, nw := range nameWords {
				hit = hit || strings.HasPrefix(nw, w)
			}
			all = all && hit
		}
		if all {
			out = append(out, name)
			if len(out) >= 8 {
				break
			}
		}
	}
	return out
}

// fuzzy : tolérance aux fautes de frappe par similarité de trigrammes sur les titres (films et séries).
func (s *Store) fuzzy(ctx context.Context, v Viewer, q string, limit int) ([]Card, error) {
	nq := parse.Normalize(q)
	if len([]rune(nq)) < 3 {
		return nil, nil
	}
	where, args := v.Where("i")
	rows, err := s.DB.QueryContext(ctx, "SELECT i.id, i.title, i.original_title FROM items i WHERE i.kind IN ('movie','series') AND "+where, args...)
	if err != nil {
		return nil, err
	}
	type scored struct {
		id    int64
		score float64
	}
	var hits []scored
	qt := trigrams(nq)
	for rows.Next() {
		var id int64
		var t, o string
		if rows.Scan(&id, &t, &o) != nil {
			continue
		}
		sc := max(similarity(qt, nq, parse.Normalize(t)), similarity(qt, nq, parse.Normalize(o)))
		if sc >= 0.3 {
			hits = append(hits, scored{id, sc})
		}
	}
	rows.Close()
	sort.Slice(hits, func(i, j int) bool { return hits[i].score > hits[j].score })
	if len(hits) > limit {
		hits = hits[:limit]
	}
	if len(hits) == 0 {
		return nil, nil
	}
	var out []Card
	for _, h := range hits {
		rows, err := s.DB.QueryContext(ctx, "SELECT "+cardColumns+" FROM items i"+cardJoins+" WHERE i.id=? AND ((i.kind='movie' AND "+
			presentMovie+") OR (i.kind='series' AND "+presentSeries+"))", v.UserID, h.id)
		if err != nil {
			return nil, err
		}
		cs, err := scanCards(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, cs...)
	}
	return out, nil
}

func trigrams(s string) map[string]bool {
	r := []rune("  " + s + " ")
	m := map[string]bool{}
	for i := 0; i+3 <= len(r); i++ {
		m[string(r[i:i+3])] = true
	}
	return m
}

// similarity : meilleur score entre le titre entier et ses fenêtres de même longueur que la requête
// (« matrx » doit retrouver « The Matrix Reloaded »).
func similarity(qt map[string]bool, q, title string) float64 {
	if title == "" {
		return 0
	}
	best := jaccard(qt, trigrams(title))
	words := strings.Fields(title)
	n := len(strings.Fields(q))
	for i := 0; i+n <= len(words); i++ {
		if sc := jaccard(qt, trigrams(strings.Join(words[i:i+n], " "))); sc > best {
			best = sc
		}
	}
	return best
}

func jaccard(a, b map[string]bool) float64 {
	inter := 0
	for k := range a {
		if b[k] {
			inter++
		}
	}
	union := len(a) + len(b) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}
