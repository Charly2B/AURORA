package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Les candidats IMDb restent associables même sans fiche TMDB ; une suggestion ne modifie jamais la base.
func TestIMDbFallbackCandidatesAndMatch(t *testing.T) {
	e := newEnv(t)
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/imdb/missing film.json", r.URL.Path == "/imdb/tt1234567.json":
			fmt.Fprint(w, `{"d":[{"id":"tt1234567","l":"Missing Film","qid":"movie","q":"feature","y":2024}]}`)
		case r.URL.Path == "/imdb/missing series.json", r.URL.Path == "/imdb/tt7654321.json":
			fmt.Fprint(w, `{"d":[{"id":"tt7654321","l":"Missing Series","qid":"tvSeries","q":"TV series","y":2023}]}`)
		case r.URL.Path == "/3/search/movie", r.URL.Path == "/3/search/tv":
			fmt.Fprint(w, `{"results":[]}`)
		case r.URL.Path == "/3/find/tt1234567", r.URL.Path == "/3/find/tt7654321":
			fmt.Fprint(w, `{"movie_results":[],"tv_results":[]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer fake.Close()
	e.s.TMDBBase, e.s.TMDBImages, e.s.IMDbBase = fake.URL+"/3", fake.URL+"/images", fake.URL+"/imdb"
	admin := e.client()
	admin.do("POST", "/api/v1/setup/admin", map[string]string{"username": "admin", "password": "motdepasse1"}, nil)
	if _, err := e.s.DB.Exec(`INSERT INTO settings(key,value) VALUES ('tmdb_api_key','test-key');
		INSERT INTO libraries(id,name,kind,path,created_at) VALUES (9,'L','movies','/x',0);
		INSERT INTO items(id,library_id,kind,match_key,title,sort_title,added_at,updated_at)
		VALUES (77,9,'movie','film','Local Film','local film',0,0),(78,9,'series','series','Local Series','local series',0,0)`); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		id     int
		query  string
		imdbID string
		title  string
	}{
		{77, "missing%20film", "tt1234567", "Missing Film"},
		{78, "missing%20series", "tt7654321", "Missing Series"},
	} {
		t.Run(tc.title, func(t *testing.T) {
			var results []struct {
				ID     int    `json:"id"`
				IMDbID string `json:"imdbId"`
				Title  string `json:"title"`
			}
			if status := admin.do("GET", fmt.Sprintf("/api/v1/admin/items/%d/candidates?q=%s", tc.id, tc.query), nil, &results); status != 200 || len(results) != 1 || results[0].IMDbID != tc.imdbID || results[0].Title != tc.title {
				t.Fatalf("candidats IMDb : %d %+v", status, results)
			}
			var state string
			e.s.DB.QueryRow(`SELECT match_state FROM items WHERE id=?`, tc.id).Scan(&state)
			if state != "local" {
				t.Fatalf("recherche a modifié l'œuvre : %s", state)
			}
			var resolved struct {
				IMDbID string `json:"imdbId"`
			}
			if status := admin.do("GET", fmt.Sprintf("/api/v1/admin/items/%d/resolve?url=https://www.imdb.com/title/%s/", tc.id, tc.imdbID), nil, &resolved); status != 200 || resolved.IMDbID != tc.imdbID {
				t.Fatalf("lien IMDb sans fiche TMDB : %d %+v", status, resolved)
			}
			var applied struct {
				ItemID int `json:"itemId"`
			}
			if status := admin.do("POST", fmt.Sprintf("/api/v1/admin/items/%d/match", tc.id), map[string]any{"imdbId": tc.imdbID}, &applied); status != 200 || applied.ItemID != tc.id {
				t.Fatalf("association IMDb : %d %+v", status, applied)
			}
			var title, imdbID string
			var age *int
			if err := e.s.DB.QueryRow(`SELECT title, imdb_id, match_state, min_age FROM items WHERE id=?`, tc.id).Scan(&title, &imdbID, &state, &age); err != nil {
				t.Fatal(err)
			}
			if title != tc.title || imdbID != tc.imdbID || state != "locked" || age != nil {
				t.Fatalf("association persistée : %s %s %s %v", title, imdbID, state, age)
			}
		})
	}
	// Un identifiant IMDb valide de film ne peut pas être appliqué à une série.
	if status := admin.do("POST", "/api/v1/admin/items/78/match", map[string]any{"imdbId": "tt1234567"}, nil); status != 422 {
		t.Fatalf("type IMDb incompatible : %d", status)
	}
	for _, body := range []map[string]any{{"imdbId": "../../etc/passwd"}, {"tmdbId": 603, "imdbId": "tt1234567"}, {}} {
		if status := admin.do("POST", "/api/v1/admin/items/77/match", body, nil); status != 400 {
			t.Fatalf("identifiant invalide : %d", status)
		}
	}
}
