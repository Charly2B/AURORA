package catalog

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"aurora/internal/db"
)

func searchStore(t *testing.T) *Store {
	t.Helper()
	root := t.TempDir()
	d, err := db.Open(context.Background(), filepath.Join(root, "catalog.db"), filepath.Join(root, "backups"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	for _, query := range []string{
		`INSERT INTO users(id, username, password_hash, role, created_at) VALUES (1, 'viewer', '', 'user', 0)`,
		`INSERT INTO libraries(id, name, kind, path, created_at) VALUES (1, 'Films', 'movies', '/films', 0), (2, 'Privée', 'movies', '/private', 0)`,
		`INSERT INTO user_libraries(user_id, library_id) VALUES (1, 1)`,
	} {
		if _, err := d.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	return &Store{DB: d.DB}
}

func searchItem(t *testing.T, s *Store, kind, title, original string, library int, present bool) int64 {
	t.Helper()
	r, err := s.DB.Exec(`INSERT INTO items(library_id, kind, match_key, title, sort_title, original_title, added_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 0, 0)`, library, kind, title, title, title, original)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := r.LastInsertId()
	if err := ReindexItem(context.Background(), s.DB, id); err != nil {
		t.Fatal(err)
	}
	fileItem := id
	if kind == "series" {
		r, err = s.DB.Exec(`INSERT INTO items(library_id, kind, parent_id, match_key, title, sort_title, season, episode, added_at, updated_at)
			VALUES (?, 'episode', ?, ?, 'Épisode 1', 'episode 1', 1, 1, 0, 0)`, library, id, fmt.Sprintf("%d|1|1", id))
		if err != nil {
			t.Fatal(err)
		}
		fileItem, _ = r.LastInsertId()
	}
	if _, err := s.DB.Exec(`INSERT INTO media_files(library_id, item_id, path, size, mtime, present, added_at)
		VALUES (?, ?, ?, 1, 0, ?, 0)`, library, fileItem, fmt.Sprintf("/media/%d.mkv", id), present); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestSearchSmallTypos(t *testing.T) {
	for _, tc := range []struct {
		name, kind, title, original, query string
	}{
		{"pluriel manquant", "movie", "Cars", "", "Car"},
		{"lettre supprimée", "movie", "Cars", "", "Cas"},
		{"lettre ajoutée", "movie", "Cars", "", "Carts"},
		{"lettre remplacée", "movie", "Cars", "", "Cers"},
		{"lettres inversées", "movie", "Cars", "", "Cras"},
		{"série courte", "series", "Lost", "", "Lots"},
		{"mot dans un titre", "movie", "The Matrix Reloaded", "", "Matrxi"},
		{"plusieurs mots et accents", "movie", "Le château ambulant", "", "chatau ambulant"},
		{"titre original", "movie", "Le Voyage de Chihiro", "Spirited Away", "Spirted Away"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := searchStore(t)
			id := searchItem(t, s, tc.kind, tc.title, tc.original, 1, true)
			res, err := s.Search(context.Background(), Viewer{UserID: 1, AllLibraries: true}, tc.query, 20)
			if err != nil {
				t.Fatal(err)
			}
			cards := append(res.Movies, res.Series...)
			if len(cards) != 1 || cards[0].ID != id {
				t.Fatalf("%q doit trouver %q : %+v", tc.query, tc.title, res)
			}
		})
	}
}

func TestSearchCompletesExactResultsWithNearbyTitles(t *testing.T) {
	s := searchStore(t)
	exact := searchItem(t, s, "movie", "Matrxi Chronicles", "", 1, true)
	near := searchItem(t, s, "movie", "The Matrix", "", 1, true)
	res, err := s.Search(context.Background(), Viewer{UserID: 1, AllLibraries: true}, "Matrxi", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Movies) != 2 || res.Movies[0].ID != exact || res.Movies[1].ID != near || res.Fuzzy {
		t.Fatalf("résultat exact puis titre proche attendus, sans doublon : %+v", res)
	}
	res, err = s.Search(context.Background(), Viewer{UserID: 1, AllLibraries: true}, "Matrxi", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Movies) != 1 || res.Movies[0].ID != exact || res.Fuzzy {
		t.Fatalf("limite respectée, sans annoncer une suggestion non affichée : %+v", res)
	}
}

func TestSearchNearbyTitlesAreRankedAndVisibleBeforeLimit(t *testing.T) {
	s := searchStore(t)
	searchItem(t, s, "movie", "Cras", "", 1, false)
	searchItem(t, s, "movie", "Cras Privé", "", 2, true)
	searchItem(t, s, "movie", "Cras Adulte", "", 1, true)
	if _, err := s.DB.Exec(`UPDATE items SET min_age=18 WHERE title='Cras Adulte'`); err != nil {
		t.Fatal(err)
	}
	best := searchItem(t, s, "movie", "Crabs", "", 1, true)
	searchItem(t, s, "movie", "Cars", "", 1, true)
	age := 12
	res, err := s.Search(context.Background(), Viewer{UserID: 1, MaxAge: &age}, "Cras", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Movies) != 1 || res.Movies[0].ID != best || !res.Fuzzy {
		t.Fatalf("meilleur film visible attendu malgré les candidats cachés : %+v", res)
	}
}

func TestSearchDoesNotSuggestUnrelatedTitles(t *testing.T) {
	s := searchStore(t)
	searchItem(t, s, "movie", "Cars", "", 1, true)
	res, err := s.Search(context.Background(), Viewer{UserID: 1, AllLibraries: true}, "Zebra", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Movies) != 0 || res.Fuzzy {
		t.Fatalf("aucune suggestion sans titre proche : %+v", res)
	}
}
