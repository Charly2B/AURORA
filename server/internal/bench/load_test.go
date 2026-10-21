//go:build bench

// Test de charge : indexation (parcours + identification) et temps de réponse du catalogue.
// Lancement : go test -tags bench -run TestLoad -v -timeout 30m ./internal/bench
// Les fichiers sont vides (0 octet) : l'analyse ffprobe n'est pas mesurée ici (elle dépend des disques),
// des résultats techniques plausibles sont injectés pour que les requêtes portent sur des données réalistes.
package bench

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"aurora/internal/catalog"
	"aurora/internal/db"
	"aurora/internal/scanner"
)

type scenario struct {
	name           string
	movies, series int
	epsPerSeries   int
}

var words = []string{"Ombre", "Lumière", "Nuit", "Été", "Rivière", "Dernier", "Voyage", "Secret", "Empire", "Cité",
	"Matrix", "Horizon", "Étoile", "Silence", "Mémoire", "Odyssée", "Tempête", "Royaume", "Chasseur", "Miroir"}

func title(r *rand.Rand, i int) string {
	return fmt.Sprintf("%s %s %d", words[r.Intn(len(words))], words[r.Intn(len(words))], i)
}

func mkfile(p string) {
	os.MkdirAll(filepath.Dir(p), 0o755)
	f, _ := os.Create(p)
	f.Close()
}

func TestLoad(t *testing.T) {
	scenarios := []scenario{
		{"1 000 films", 1000, 0, 0},
		{"5 000 films + 10 000 épisodes", 5000, 500, 20},
		{"50 000 fichiers (20 000 films + 30 000 épisodes)", 20000, 1000, 30},
	}
	fmt.Printf("\nMachine : %s/%s, %d cœurs logiques, Go %s\n", runtime.GOOS, runtime.GOARCH, runtime.NumCPU(), runtime.Version())
	for _, sc := range scenarios {
		run(t, sc)
	}
}

func timeit(f func()) time.Duration {
	s := time.Now()
	f()
	return time.Since(s)
}

func run(t *testing.T, sc scenario) {
	ctx := context.Background()
	root := t.TempDir()
	r := rand.New(rand.NewSource(42))
	movies := filepath.Join(root, "Films")
	series := filepath.Join(root, "Series")
	for i := 0; i < sc.movies; i++ {
		mkfile(filepath.Join(movies, fmt.Sprintf("%s (%d)", title(r, i), 1950+r.Intn(75)), fmt.Sprintf("film.%d.1080p.mkv", i)))
	}
	for s := 0; s < sc.series; s++ {
		name := title(r, s)
		for e := 1; e <= sc.epsPerSeries; e++ {
			mkfile(filepath.Join(series, name, fmt.Sprintf("Saison %d", (e-1)/10+1), fmt.Sprintf("%s.S%02dE%02d.mkv", name, (e-1)/10+1, (e-1)%10+1)))
		}
	}
	d, err := db.Open(ctx, filepath.Join(root, "aurora.db"), filepath.Join(root, "backups"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	d.Exec(`INSERT INTO libraries(id,name,kind,path,created_at) VALUES (1,'Films','movies',?,0),(2,'Séries','series',?,0)`, movies, series)
	sc2 := &scanner.Scanner{DB: d.DB, FFprobe: ""}
	runtime.GC()
	var m0 runtime.MemStats
	runtime.ReadMemStats(&m0)
	scan := timeit(func() {
		sc2.Enqueue(1, 2)
		time.Sleep(50 * time.Millisecond)
		for sc2.Current() != nil || len(sc2.Queued()) > 0 {
			time.Sleep(20 * time.Millisecond)
		}
	})
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	rescan := timeit(func() {
		sc2.Enqueue(1, 2)
		time.Sleep(50 * time.Millisecond)
		for sc2.Current() != nil || len(sc2.Queued()) > 0 {
			time.Sleep(20 * time.Millisecond)
		}
	})
	// Résultats techniques plausibles (sans ffprobe).
	d.Exec(`UPDATE media_files SET probe_state='ok', duration=5400 + (id % 3000), width=CASE id % 3 WHEN 0 THEN 3840 ELSE 1920 END,
		height=CASE id % 3 WHEN 0 THEN 2160 ELSE 1080 END, hdr=CASE id % 5 WHEN 0 THEN 'HDR10' ELSE '' END, video_codec='hevc',
		audio='[{"index":0,"stream":1,"codec":"eac3","channels":6,"language":"fre","default":true}]'`)
	d.Exec(`UPDATE items SET genres = CASE id % 4 WHEN 0 THEN '["Drame"]' WHEN 1 THEN '["Action","Aventure"]' WHEN 2 THEN '["Comédie"]' ELSE '["Science-Fiction"]' END,
		rating = (id % 90) / 10.0`)
	var userID int64 = 1
	d.Exec(`INSERT INTO users(id,username,password_hash,role,created_at) VALUES (1,'bench','x','admin',0)`)
	// Historique : 200 œuvres en cours.
	d.Exec(`INSERT INTO user_state(user_id,item_id,position,duration,last_played_at) SELECT 1, id, 600, 5400, id FROM items WHERE kind IN ('movie','episode') LIMIT 200`)
	st := &catalog.Store{DB: d.DB}
	v := catalog.Viewer{UserID: userID, AllLibraries: true}
	child := 10
	vc := catalog.Viewer{UserID: userID, AllLibraries: true, MaxAge: &child, HideUnrated: true}

	best := func(f func()) time.Duration { // meilleur de 5 (cache chaud)
		b := time.Hour
		for i := 0; i < 5; i++ {
			if d := timeit(f); d < b {
				b = d
			}
		}
		return b
	}
	var n int
	d.QueryRow(`SELECT COUNT(*) FROM media_files`).Scan(&n)
	fmt.Printf("\n=== %s — %d fichiers indexés\n", sc.name, n)
	fmt.Printf("  indexation initiale (parcours + identification) : %v\n", scan.Round(time.Millisecond))
	fmt.Printf("  nouvelle analyse sans changement (incrémentale) : %v\n", rescan.Round(time.Millisecond))
	fmt.Printf("  mémoire allouée pendant l'indexation             : %.1f Mo (tas après : %.1f Mo)\n",
		float64(m1.TotalAlloc-m0.TotalAlloc)/(1<<20), float64(m1.HeapAlloc)/(1<<20))
	fmt.Printf("  liste films, 1re page (tri titre)                : %v\n", best(func() { st.List(ctx, v, catalog.ListQuery{Kind: "movie", Sort: "title", Limit: 120}) }))
	fmt.Printf("  liste films, page 30 (tri qualité, filtre 4K)    : %v\n", best(func() {
		st.List(ctx, v, catalog.ListQuery{Kind: "movie", Sort: "quality", Resolution: "4K", Limit: 120, Offset: 3480})
	}))
	fmt.Printf("  liste séries (tri titre, compteurs)              : %v\n", best(func() { st.List(ctx, v, catalog.ListQuery{Kind: "series", Limit: 120}) }))
	fmt.Printf("  filtres disponibles (facettes films)             : %v\n", best(func() { st.Facets(ctx, v, "movie", 0) }))
	fmt.Printf("  accueil complet                                  : %v\n", best(func() { st.Home(ctx, v, nil) }))
	fmt.Printf("  accueil, profil enfant                           : %v\n", best(func() { st.Home(ctx, vc, nil) }))
	fmt.Printf("  recherche « ombre nuit »                         : %v\n", best(func() { st.Search(ctx, v, "ombre nuit", 20) }))
	fmt.Printf("  recherche avec faute « odysee tempete »          : %v\n", best(func() { st.Search(ctx, v, "odysee tempete", 20) }))
	fmt.Printf("  recherche sans résultat exact « matrxi » (approx.) : %v\n", best(func() { st.Search(ctx, v, "matrxi", 20) }))
	var anyMovie int64
	d.QueryRow(`SELECT id FROM items WHERE kind='movie' LIMIT 1 OFFSET 500`).Scan(&anyMovie)
	fmt.Printf("  fiche film                                       : %v\n", best(func() { st.Item(ctx, v, anyMovie) }))
	var dbSize int64
	if fi, err := os.Stat(filepath.Join(root, "aurora.db")); err == nil {
		dbSize = fi.Size()
	}
	fmt.Printf("  taille de la base                                : %.1f Mo\n", float64(dbSize)/(1<<20))
	_ = sql.ErrNoRows
}
