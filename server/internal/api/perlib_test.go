package api

import (
	"path/filepath"
	"testing"

	"aurora/internal/catalog"
)

// Les dossiers ne sont pas mélangés : une rangée d'accueil par dossier, compteurs par dossier.
func TestPerLibrarySections(t *testing.T) {
	e := newEnv(t)
	genVideo(t, filepath.Join(e.media, "Series 2021", "Autre (2021)", "Saison 1", "Autre.S01E01.mkv"), "-c:a", "aac")
	admin := e.client()
	admin.do("POST", "/api/v1/setup/admin", map[string]string{"username": "admin", "password": "motdepasse1"}, nil)
	admin.do("POST", "/api/v1/admin/libraries", map[string]any{"name": "Series en cours", "kind": "series", "path": filepath.Join(e.media, "Series")}, nil)
	admin.do("POST", "/api/v1/admin/libraries", map[string]any{"name": "Series 2021", "kind": "series", "path": filepath.Join(e.media, "Series 2021")}, nil)
	admin.do("POST", "/api/v1/admin/libraries", map[string]any{"name": "Films", "kind": "movies", "path": filepath.Join(e.media, "Films")}, nil)
	e.waitScan()

	var libs []catalog.LibraryRef
	admin.do("GET", "/api/v1/libraries", nil, &libs)
	if len(libs) != 3 || libs[0].Series != 1 || libs[1].Series != 1 || libs[2].Movies != 2 {
		t.Fatalf("compteurs par dossier : %+v", libs)
	}
	var home catalog.Home
	admin.do("GET", "/api/v1/home", nil, &home)
	titles := map[string]bool{}
	for _, s := range home.Sections {
		titles[s.Title] = true
	}
	if !titles["Series en cours"] || !titles["Series 2021"] || titles["Séries récemment ajoutées"] {
		t.Fatalf("une rangée par dossier de séries attendue : %v", titles)
	}
	if !titles["Films récemment ajoutés"] { // un seul dossier de films : titre générique conservé
		t.Fatalf("rangée films attendue : %v", titles)
	}
	var list struct {
		Total int `json:"total"`
	}
	admin.do("GET", "/api/v1/series?library="+itoa(libs[1].ID), nil, &list)
	if list.Total != 1 {
		t.Fatalf("filtre par dossier : %d", list.Total)
	}
}

func itoa(v int64) string {
	b := []byte{}
	if v == 0 {
		return "0"
	}
	for v > 0 {
		b = append([]byte{byte('0' + v%10)}, b...)
		v /= 10
	}
	return string(b)
}
