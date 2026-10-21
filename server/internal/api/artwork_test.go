package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aurora/internal/catalog"
)

// Choix d'affiche : faux serveur TMDB, chemins d'image validés, droits administrateur.
func TestArtworkPicker(t *testing.T) {
	e := newEnv(t)
	png := []byte("\x89PNG\r\n\x1a\nfake")
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/3/movie/123/images":
			fmt.Fprint(w, `{"posters":[{"file_path":"/affiche_fr.jpg","iso_639_1":"fr","vote_average":5.2,"width":1000,"height":1500},
				{"file_path":"/affiche_en.jpg","iso_639_1":"en"}],"backdrops":[{"file_path":"/fond_1.jpg","iso_639_1":null}]}`)
		case strings.HasPrefix(r.URL.Path, "/t/p/"):
			w.Write(png)
		default:
			http.NotFound(w, r)
		}
	}))
	defer fake.Close()
	e.s.TMDBBase, e.s.TMDBImages = fake.URL+"/3", fake.URL+"/t/p"

	admin := e.client()
	admin.do("POST", "/api/v1/setup/admin", map[string]string{"username": "admin", "password": "motdepasse1"}, nil)
	e.s.DB.Exec(`INSERT INTO settings(key,value) VALUES ('tmdb_api_key','cle-de-test')`)
	e.s.DB.Exec(`INSERT INTO libraries(id,name,kind,path,created_at) VALUES (9,'L','movies','/x',0)`)
	e.s.DB.Exec(`INSERT INTO items(id,library_id,kind,match_key,title,sort_title,tmdb_id,added_at,updated_at) VALUES (77,9,'movie','k','Film','film',123,0,0)`)

	var list struct {
		Posters   []map[string]any `json:"posters"`
		Backdrops []map[string]any `json:"backdrops"`
	}
	if code := admin.do("GET", "/api/v1/admin/items/77/artwork", nil, &list); code != 200 || len(list.Posters) != 2 || len(list.Backdrops) != 1 {
		t.Fatalf("liste : %d %+v", code, list)
	}
	var set map[string]string
	if code := admin.do("POST", "/api/v1/admin/items/77/artwork", map[string]string{"type": "poster", "path": "/affiche_en.jpg"}, &set); code != 200 {
		t.Fatalf("choix : %d", code)
	}
	var poster string
	e.s.DB.QueryRow(`SELECT poster FROM items WHERE id=77`).Scan(&poster)
	if poster == "" || set["url"] != catalog.ImageURL(poster) {
		t.Fatalf("affiche non enregistrée : %q %v", poster, set)
	}
	for _, bad := range []string{"../../etc/passwd", "http://evil/x.jpg", "/a b.jpg", "/x.svg"} {
		if code := admin.do("POST", "/api/v1/admin/items/77/artwork", map[string]string{"type": "poster", "path": bad}, nil); code != 400 {
			t.Fatalf("chemin refusé attendu pour %q : %d", bad, code)
		}
	}
	admin.do("POST", "/api/v1/admin/users", map[string]any{"username": "user", "password": "motdepasse2", "role": "user", "allLibraries": true}, nil)
	u := e.client()
	u.do("POST", "/api/v1/auth/login", map[string]string{"username": "user", "password": "motdepasse2"}, nil)
	if code := u.do("POST", "/api/v1/admin/items/77/artwork", map[string]string{"type": "poster", "path": "/affiche_fr.jpg"}, nil); code != 403 {
		t.Fatalf("réservé aux administrateurs : %d", code)
	}
}
