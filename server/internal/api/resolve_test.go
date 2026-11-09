package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// Lien collé quand TMDB ne trouve pas l'œuvre : TMDB direct, IMDb via /find, type vérifié, droits admin.
func TestResolveLink(t *testing.T) {
	e := newEnv(t)
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/3/tv/219937":
			fmt.Fprint(w, `{"id":219937,"name":"Hunter","original_name":"Hunter","first_air_date":"2023-03-01","overview":"Résumé","poster_path":"/h.jpg"}`)
		case "/3/movie/603":
			fmt.Fprint(w, `{"id":603,"title":"Matrix","original_title":"The Matrix","release_date":"1999-03-31","overview":"Neo","poster_path":"/m.jpg"}`)
		case "/3/find/tt0944947":
			fmt.Fprint(w, `{"movie_results":[],"tv_results":[{"id":219937}]}`)
		case "/3/find/tt1480055":
			fmt.Fprint(w, `{"movie_results":[],"tv_results":[],"tv_episode_results":[{"id":63056,"show_id":219937}]}`)
		case "/3/find/tt0000001":
			fmt.Fprint(w, `{"movie_results":[],"tv_results":[]}`)
		case "/imdb/tt0000001.json":
			fmt.Fprint(w, `{"d":[]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer fake.Close()
	e.s.TMDBBase, e.s.TMDBImages = fake.URL+"/3", fake.URL+"/t/p"
	e.s.IMDbBase = fake.URL + "/imdb"

	admin := e.client()
	admin.do("POST", "/api/v1/setup/admin", map[string]string{"username": "admin", "password": "motdepasse1"}, nil)
	e.s.DB.Exec(`INSERT INTO settings(key,value) VALUES ('tmdb_api_key','cle-de-test')`)
	e.s.DB.Exec(`INSERT INTO libraries(id,name,kind,path,created_at) VALUES (9,'L','series','/x',0)`)
	e.s.DB.Exec(`INSERT INTO items(id,library_id,kind,match_key,title,sort_title,added_at,updated_at) VALUES (77,9,'series','k','Hunter','hunter',0,0)`)

	resolve := func(c *client, link string, out any) int {
		return c.do("GET", "/api/v1/admin/items/77/resolve?url="+url.QueryEscape(link), nil, out)
	}
	var res struct {
		ID     int    `json:"id"`
		Title  string `json:"title"`
		Year   int    `json:"year"`
		Poster string `json:"poster"`
	}
	if c := resolve(admin, "https://www.themoviedb.org/tv/219937-hunter", &res); c != 200 || res.ID != 219937 || res.Title != "Hunter" || res.Year != 2023 || res.Poster == "" {
		t.Fatalf("lien TMDB : %d %+v", c, res)
	}
	res.ID = 0
	if c := resolve(admin, "https://www.imdb.com/title/tt0944947/", &res); c != 200 || res.ID != 219937 {
		t.Fatalf("lien IMDb : %d %+v", c, res)
	}
	res.ID = 0
	if c := resolve(admin, "imdb.com/title/tt1480055", &res); c != 200 || res.ID != 219937 {
		t.Fatalf("épisode IMDb sans https:// → série : %d %+v", c, res)
	}
	if c := resolve(admin, "https://www.themoviedb.org/tv/1", nil); c != 422 {
		t.Fatalf("fiche TMDB inexistante : %d", c)
	}
	var e1 struct{ Error string }
	if c := resolve(admin, "https://www.themoviedb.org/movie/603-the-matrix", &e1); c != 422 || e1.Error == "" {
		t.Fatalf("film pour une série : %d %+v", c, e1)
	}
	if c := resolve(admin, "https://www.imdb.com/title/tt0000001/", nil); c != 422 {
		t.Fatalf("IMDb inconnu : %d", c)
	}
	for _, bad := range []string{"https://evil.example/tv/1", "pas un lien", "http://127.0.0.1:8890/tv/1"} {
		if c := resolve(admin, bad, nil); c != 400 {
			t.Fatalf("lien refusé attendu pour %q : %d", bad, c)
		}
	}
	// Associer ensuite par /match reste l'étape de confirmation existante ; un simple profil n'a pas accès.
	admin.do("POST", "/api/v1/admin/users", map[string]any{"username": "user", "password": "motdepasse2", "role": "user", "allLibraries": true}, nil)
	u := e.client()
	u.do("POST", "/api/v1/auth/login", map[string]string{"username": "user", "password": "motdepasse2"}, nil)
	if c := resolve(u, "https://www.themoviedb.org/tv/219937", nil); c != 403 {
		t.Fatalf("profil non administrateur : %d", c)
	}
}
