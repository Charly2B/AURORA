package watcher

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"aurora/internal/db"
)

// Surveillance réelle (inotify sous Linux, kqueue sous macOS) d'un dossier temporaire.
func TestWatcher(t *testing.T) {
	root := t.TempDir()
	films := filepath.Join(root, "Films")
	os.MkdirAll(filepath.Join(films, "Ancien (2001)"), 0o755)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d, err := db.Open(ctx, filepath.Join(root, "data", "aurora.db"), filepath.Join(root, "data", "backups"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if _, err := d.ExecContext(ctx, `INSERT INTO libraries(id, name, kind, path, created_at) VALUES (7, 'Films', 'movies', ?, 0)`, films); err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	var got []int64
	enabled := true
	w := &Watcher{DB: d.DB, Enabled: func(context.Context) bool { mu.Lock(); defer mu.Unlock(); return enabled }, Settle: 300 * time.Millisecond, MaxWait: 5 * time.Second, Enqueue: func(ids ...int64) {
		mu.Lock()
		got = append(got, ids...)
		mu.Unlock()
	}}
	if err := w.Start(ctx); err != nil {
		t.Fatal(err)
	}
	w.Sync(ctx)
	if st := w.Status(); !st.Active || st.Dirs != 2 {
		t.Fatalf("racine et sous-dossier attendus : %+v", st)
	}
	scans := func() []int64 {
		mu.Lock()
		defer mu.Unlock()
		return append([]int64(nil), got...)
	}
	waitScans := func(n int) {
		t.Helper()
		deadline := time.Now().Add(8 * time.Second)
		for len(scans()) < n {
			if time.Now().After(deadline) {
				t.Fatalf("%d analyse(s) attendue(s), obtenu %v", n, scans())
			}
			time.Sleep(50 * time.Millisecond)
		}
	}

	// Bruit DSM (miniatures @eaDir) et fichiers sans intérêt : aucune analyse.
	os.MkdirAll(filepath.Join(films, "Ancien (2001)", "@eaDir"), 0o755)
	os.WriteFile(filepath.Join(films, "Ancien (2001)", "notes.txt"), []byte("x"), 0o644)
	time.Sleep(3 * time.Second)
	if s := scans(); len(s) != 0 {
		t.Fatalf("analyse déclenchée par du bruit : %v", s)
	}

	// Nouveau film dans un nouveau dossier : une seule analyse, après le calme, malgré plusieurs écritures.
	dir := filepath.Join(films, "Nouveau (2024)")
	os.MkdirAll(dir, 0o755)
	f, _ := os.Create(filepath.Join(dir, "Nouveau.2024.mkv"))
	for i := 0; i < 5; i++ {
		f.Write(make([]byte, 1024))
		time.Sleep(100 * time.Millisecond)
	}
	f.Close()
	waitScans(1)
	time.Sleep(3 * time.Second)
	if s := scans(); len(s) != 1 || s[0] != 7 {
		t.Fatalf("une seule analyse de la bibliothèque 7 attendue : %v", s)
	}
	if st := w.Status(); st.Dirs != 3 {
		t.Fatalf("le nouveau dossier doit être surveillé : %+v", st)
	}

	// Fichier ajouté plus tard dans ce nouveau dossier : vu grâce au watch ajouté à la volée.
	os.WriteFile(filepath.Join(dir, "Nouveau.2024.fr.srt"), []byte("1"), 0o644)
	waitScans(2)

	// Suppression d'un dossier : analyse (les fichiers disparus sont masqués) et watch oublié.
	os.RemoveAll(filepath.Join(films, "Ancien (2001)"))
	waitScans(3)
	if st := w.Status(); st.Dirs != 2 {
		t.Fatalf("dossier supprimé encore suivi : %+v", st)
	}

	// Watch abandonné par le noyau sans évènement (démontage d'un disque, IN_UNMOUNT) : Sync le voit dans
	// WatchList et resurveille la racine.
	w.mu.Lock()
	w.w.Remove(films)
	w.mu.Unlock()
	w.Sync(ctx)
	if st := w.Status(); st.Dirs != 2 {
		t.Fatalf("racine abandonnée non resurveillée : %+v", st)
	}
	os.WriteFile(filepath.Join(films, "Racine.2023.mkv"), []byte("x"), 0o644)
	waitScans(4)

	// Réglage désactivé : plus aucune surveillance.
	mu.Lock()
	enabled = false
	mu.Unlock()
	w.Sync(ctx)
	if st := w.Status(); st.Active || st.Dirs != 0 {
		t.Fatalf("surveillance non retirée : %+v", st)
	}
}
