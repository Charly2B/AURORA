package api

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"aurora/internal/catalog"
)

// « Retirer de Continuer à regarder » : film et série masqués des rangées, progression et fichiers intacts,
// retour automatique à la lecture suivante, sans effet sur un autre profil.
func TestHideContinue(t *testing.T) {
	e := newEnv(t)
	admin := e.client()
	admin.do("POST", "/api/v1/setup/admin", map[string]string{"username": "admin", "password": "motdepasse1"}, nil)
	admin.do("POST", "/api/v1/admin/libraries", map[string]any{"kind": "movies", "path": filepath.Join(e.media, "Films")}, nil)
	admin.do("POST", "/api/v1/admin/libraries", map[string]any{"kind": "series", "path": filepath.Join(e.media, "Series")}, nil)
	e.waitScan()

	var movies struct{ Items []catalog.Card }
	admin.do("GET", "/api/v1/movies", nil, &movies)
	film := movies.Items[0]
	var d catalog.Detail
	admin.do("GET", fmt.Sprintf("/api/v1/items/%d", film.ID), nil, &d)
	admin.do("POST", fmt.Sprintf("/api/v1/items/%d/progress", film.ID), map[string]any{"fileId": d.Files[0].ID, "position": 12, "duration": 20}, nil)

	var series struct{ Items []catalog.Card }
	admin.do("GET", "/api/v1/series", nil, &series)
	var eps []catalog.Card
	admin.do("GET", fmt.Sprintf("/api/v1/items/%d/seasons/1", series.Items[0].ID), nil, &eps)
	var ep1 catalog.Detail
	admin.do("GET", fmt.Sprintf("/api/v1/items/%d", eps[0].ID), nil, &ep1)
	admin.do("POST", fmt.Sprintf("/api/v1/items/%d/progress", eps[0].ID), map[string]any{"fileId": ep1.Files[0].ID, "position": 19.9, "duration": 20}, nil)

	in := func(c *client, section string, id int64) bool {
		var home catalog.Home
		c.do("GET", "/api/v1/home", nil, &home)
		for _, s := range home.Sections {
			if s.ID == section {
				for _, it := range s.Items {
					if it.ID == id {
						return true
					}
				}
			}
		}
		return false
	}
	if !in(admin, "continue", film.ID) || !in(admin, "nextup", eps[1].ID) {
		t.Fatal("film en cours et épisode suivant attendus avant le retrait")
	}

	// Retrait du film et de la série (via l'épisode proposé : c'est la série entière qui est retirée).
	if c := admin.do("POST", fmt.Sprintf("/api/v1/items/%d/continue/hide", film.ID), map[string]any{}, nil); c != 200 {
		t.Fatalf("retrait du film : %d", c)
	}
	if c := admin.do("POST", fmt.Sprintf("/api/v1/items/%d/continue/hide", eps[1].ID), map[string]any{}, nil); c != 200 {
		t.Fatalf("retrait de la série : %d", c)
	}
	if in(admin, "continue", film.ID) || in(admin, "nextup", eps[1].ID) {
		t.Fatal("encore présents après le retrait")
	}
	var home catalog.Home
	admin.do("GET", "/api/v1/home", nil, &home)
	if home.Hero != nil && home.Hero.ID == film.ID {
		t.Fatal("film retiré remis en avant dans le bandeau")
	}
	admin.do("GET", fmt.Sprintf("/api/v1/items/%d", film.ID), nil, &d)
	if d.State.Position != 12 {
		t.Fatalf("progression perdue : %+v", d.State)
	}

	// Un autre profil n'est pas concerné.
	admin.do("POST", "/api/v1/admin/users", map[string]any{"username": "autre", "password": "motdepasse2", "role": "user", "allLibraries": true}, nil)
	other := e.client()
	other.do("POST", "/api/v1/auth/login", map[string]string{"username": "autre", "password": "motdepasse2"}, nil)
	other.do("POST", fmt.Sprintf("/api/v1/items/%d/progress", film.ID), map[string]any{"fileId": d.Files[0].ID, "position": 13, "duration": 20}, nil)
	if !in(other, "continue", film.ID) {
		t.Fatal("retrait appliqué à un autre profil")
	}

	// Nouvelle lecture (seconde suivante) : le film revient.
	time.Sleep(1100 * time.Millisecond)
	admin.do("POST", fmt.Sprintf("/api/v1/items/%d/progress", film.ID), map[string]any{"fileId": d.Files[0].ID, "position": 14, "duration": 20}, nil)
	if !in(admin, "continue", film.ID) {
		t.Fatal("le film relu ne revient pas")
	}

	// Œuvre inconnue : 404.
	if c := admin.do("POST", "/api/v1/items/999999/continue/hide", map[string]any{}, nil); c != 404 {
		t.Fatalf("œuvre inconnue : %d", c)
	}
}
