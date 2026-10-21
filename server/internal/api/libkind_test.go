package api

import (
	"fmt"
	"path/filepath"
	"testing"

	"aurora/internal/catalog"
)

// Scénario réel : un dossier de séries ajouté par erreur comme bibliothèque de films.
func TestLibraryKindMismatchAndChange(t *testing.T) {
	e := newEnv(t)
	admin := e.client()
	admin.do("POST", "/api/v1/setup/admin", map[string]string{"username": "admin", "password": "motdepasse1"}, nil)
	series := filepath.Join(e.media, "Series")

	// 1. Avertissement : le dossier contient des épisodes, le type proposé est « séries ».
	var warn map[string]any
	if code := admin.do("POST", "/api/v1/admin/libraries", map[string]any{"kind": "movies", "path": series}, &warn); code != 409 || warn["suggestKind"] != "series" {
		t.Fatalf("avertissement attendu : %d %v", code, warn)
	}
	// 2. L'utilisateur force « films » malgré l'avertissement.
	var lib struct {
		ID int64 `json:"id"`
	}
	if code := admin.do("POST", "/api/v1/admin/libraries", map[string]any{"kind": "movies", "path": series, "force": true}, &lib); code != 201 {
		t.Fatalf("ajout forcé : %d", code)
	}
	e.waitScan()
	var list struct {
		Items []catalog.Card `json:"items"`
		Total int            `json:"total"`
	}
	// Depuis 0.1.2, même dans une bibliothèque « films », les épisodes sont rangés en séries.
	admin.do("GET", "/api/v1/movies", nil, &list)
	if list.Total != 0 {
		t.Fatalf("aucun épisode ne doit apparaître comme film : %d", list.Total)
	}
	// 3. Correction : changement de type, réidentification sans toucher aux fichiers.
	if code := admin.do("PUT", fmt.Sprintf("/api/v1/admin/libraries/%d", lib.ID), map[string]any{"kind": "series"}, nil); code != 200 {
		t.Fatalf("changement de type : %d", code)
	}
	e.waitScan()
	admin.do("GET", "/api/v1/movies", nil, &list)
	if list.Total != 0 {
		t.Fatalf("plus aucun film attendu : %d", list.Total)
	}
	admin.do("GET", "/api/v1/series", nil, &list)
	if list.Total != 1 || list.Items[0].Total != 2 {
		t.Fatalf("1 série de 2 épisodes attendue : %+v", list)
	}
	var probed int
	e.s.DB.QueryRow(`SELECT COUNT(*) FROM media_files WHERE probe_state='ok' AND library_id=?`, lib.ID).Scan(&probed)
	if probed != 2 {
		t.Fatalf("l'analyse technique doit être conservée : %d", probed)
	}
	// 4. Un vrai dossier de films ajouté comme films : aucun avertissement.
	if code := admin.do("POST", "/api/v1/admin/libraries", map[string]any{"kind": "movies", "path": filepath.Join(e.media, "Films")}, nil); code != 201 {
		t.Fatalf("dossier de films : %d", code)
	}
}
