package api

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"aurora/internal/catalog"
)

// Bibliothèque de films contenant aussi des épisodes (cas réel) : les épisodes vont dans Séries,
// y compris ceux qu'une version antérieure avait indexés comme films.
func TestMixedMoviesLibrary(t *testing.T) {
	e := newEnv(t)
	films := filepath.Join(e.media, "Films")
	genVideo(t, filepath.Join(films, "Agen kim S01E06.mkv"), "-c:a", "aac")
	genVideo(t, filepath.Join(films, "Agen kim S01E07.mkv"), "-c:a", "aac")
	admin := e.client()
	admin.do("POST", "/api/v1/setup/admin", map[string]string{"username": "admin", "password": "motdepasse1"}, nil)
	admin.do("POST", "/api/v1/admin/libraries", map[string]any{"kind": "movies", "path": films, "force": true}, nil)
	e.waitScan()

	var list struct {
		Items []catalog.Card `json:"items"`
		Total int            `json:"total"`
	}
	check := func(step string) {
		t.Helper()
		admin.do("GET", "/api/v1/movies", nil, &list)
		if list.Total != 2 {
			t.Fatalf("%s : seuls les 2 vrais films attendus dans Films, obtenu %d", step, list.Total)
		}
		admin.do("GET", "/api/v1/series", nil, &list)
		if list.Total != 1 || list.Items[0].Title != "Agen kim" || list.Items[0].Total != 2 {
			t.Fatalf("%s : série « Agen kim » de 2 épisodes attendue : %+v", step, list.Items)
		}
	}
	check("indexation")

	// Simule l'état laissé par la version précédente : chaque épisode rattaché à un « film ».
	var libID int64
	e.s.DB.QueryRow(`SELECT id FROM libraries`).Scan(&libID)
	rows, _ := e.s.DB.Query(`SELECT id FROM media_files WHERE path LIKE '%Agen kim%'`)
	var ids []int64
	for rows.Next() {
		var id int64
		rows.Scan(&id)
		ids = append(ids, id)
	}
	rows.Close()
	for i, id := range ids {
		res, _ := e.s.DB.Exec(`INSERT INTO items(library_id,kind,match_key,title,sort_title,added_at,updated_at) VALUES (?,'movie',?,?,?,0,0)`,
			libID, "old"+string(rune('a'+i)), "Agen kim S01E0X", "agen kim")
		mid, _ := res.LastInsertId()
		e.s.DB.Exec(`UPDATE media_files SET item_id=? WHERE id=?`, mid, id)
		if i == 0 { // progression enregistrée sur l'ancien « film »
			e.s.DB.Exec(`INSERT INTO user_state(user_id,item_id,position,duration,last_played_at,last_file_id,favorite) VALUES (1,?,12,20,100,?,1)`, mid, id)
		}
	}
	e.s.DB.Exec(`DELETE FROM items WHERE kind IN ('series','episode')`)
	admin.do("GET", "/api/v1/movies", nil, &list)
	if list.Total != 4 {
		t.Fatalf("préparation : 4 « films » attendus, %d", list.Total)
	}
	admin.do("POST", "/api/v1/admin/scan", map[string]any{}, nil)
	e.waitScan()
	check("reclassement")
	var orphans int
	e.s.DB.QueryRow(`SELECT COUNT(*) FROM items WHERE kind='movie' AND title LIKE 'Agen kim%'`).Scan(&orphans)
	if orphans != 0 {
		t.Fatalf("films vidés non retirés du catalogue : %d", orphans)
	}
	if _, err := os.Stat(filepath.Join(films, "Agen kim S01E06.mkv")); err != nil {
		t.Fatal("le fichier vidéo doit rester intact")
	}
	var pos float64
	e.s.DB.QueryRow(`SELECT st.position FROM user_state st JOIN media_files f ON f.item_id=st.item_id
		WHERE f.id=? AND st.user_id=1`, ids[0]).Scan(&pos)
	var other int
	e.s.DB.QueryRow(`SELECT COUNT(*) FROM user_state st JOIN media_files f ON f.item_id=st.item_id WHERE f.id=?`, ids[1]).Scan(&other)
	var fav bool
	e.s.DB.QueryRow(`SELECT favorite FROM user_state WHERE user_id=1 AND item_id=(SELECT id FROM items WHERE kind='series')`).Scan(&fav)
	if pos != 12 || other != 0 || !fav {
		t.Fatalf("report d'historique : position=%v (12 attendu), autre épisode=%d (0 attendu), favori série=%v", pos, other, fav)
	}

	// Rattachement manuel d'un épisode à un film : jamais défait par l'analyse.
	var movieID int64
	e.s.DB.QueryRow(`SELECT id FROM items WHERE kind='movie' LIMIT 1`).Scan(&movieID)
	if code := admin.do("POST", fmt.Sprintf("/api/v1/admin/files/%d/assign", ids[1]), map[string]any{"itemId": movieID}, nil); code != 200 {
		t.Fatalf("rattachement manuel : %d", code)
	}
	admin.do("POST", "/api/v1/admin/scan", map[string]any{}, nil)
	e.waitScan()
	var owner int64
	e.s.DB.QueryRow(`SELECT item_id FROM media_files WHERE id=?`, ids[1]).Scan(&owner)
	if owner != movieID {
		t.Fatalf("rattachement manuel défait par l'analyse : %d au lieu de %d", owner, movieID)
	}
}
