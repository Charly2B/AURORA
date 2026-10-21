// Package watcher : détection immédiate des ajouts, comme Plex (« analyser quand un changement est détecté »).
//
// inotify (via fsnotify) surveille chaque dossier des bibliothèques. Un changement pertinent (vidéo,
// sous-titre, dossier) programme une analyse incrémentale de la bibliothèque concernée après un temps
// de calme : une copie de film produit des écritures en continu, l'analyse part quand elle est finie.
//
// inotify ne surveille pas récursivement : un watch par dossier, ajouté aussi pour chaque nouveau dossier.
// Le nombre de watches est plafonné par le noyau (fs.inotify.max_user_watches) : si la limite est atteinte,
// la surveillance devient partielle, c'est signalé, et l'analyse périodique continue de tout couvrir.
package watcher

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"

	"aurora/internal/parse"
	"aurora/internal/scanner"
)

// Watcher surveille les dossiers des bibliothèques et programme leur analyse.
type Watcher struct {
	DB      *sql.DB
	Enqueue func(libIDs ...int64)
	// Enabled : réglage relu à chaque Sync (deux bascules rapides ne peuvent pas laisser un état périmé).
	Enabled func(ctx context.Context) bool
	// Settle : calme requis après le dernier changement ; MaxWait : délai maximal (copie interminable).
	Settle, MaxWait time.Duration

	syncMu  sync.Mutex // un seul Sync à la fois
	mu      sync.Mutex
	w       *fsnotify.Watcher
	roots   map[int64]string // bibliothèque → dossier racine surveillé
	devs    map[int64]uint64 // bibliothèque → périphérique de la racine (disque rebranché = autre montage)
	dirs    map[string]int64 // dossier surveillé → bibliothèque
	pending map[int64]*pendingScan
	limited bool      // limite inotify atteinte : surveillance partielle
	retried time.Time // dernier nouvel essai d'une surveillance partielle
}

type pendingScan struct{ first, last time.Time }

// Status : état affiché dans le diagnostic.
type Status struct {
	Active  bool `json:"active"`
	Dirs    int  `json:"dirs"`
	Limited bool `json:"limited"`
}

// Start ouvre inotify et traite les évènements jusqu'à l'annulation de ctx.
func (w *Watcher) Start(ctx context.Context) error {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	w.mu.Lock()
	w.w = fw
	w.roots, w.devs, w.dirs, w.pending = map[int64]string{}, map[int64]uint64{}, map[string]int64{}, map[int64]*pendingScan{}
	if w.Settle == 0 {
		w.Settle = 20 * time.Second
	}
	if w.MaxWait == 0 {
		w.MaxWait = 10 * time.Minute
	}
	w.mu.Unlock()
	go w.loop(ctx)
	return nil
}

func (w *Watcher) loop(ctx context.Context) {
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	defer w.w.Close()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-w.w.Events:
			if !ok {
				return
			}
			w.handle(ev)
		case err, ok := <-w.w.Errors:
			if !ok {
				return
			}
			// Débordement de la file du noyau : des évènements sont perdus, on analyse tout par précaution.
			slog.Warn("surveillance des dossiers", "err", err)
			if errors.Is(err, fsnotify.ErrEventOverflow) {
				w.mu.Lock()
				for id := range w.roots {
					w.mark(id)
				}
				w.mu.Unlock()
			}
		case <-tick.C:
			w.flush()
		}
	}
}

// handle trie un évènement : dossier technique ignoré, nouveau dossier surveillé, bibliothèque à analyser.
func (w *Watcher) handle(ev fsnotify.Event) {
	if ev.Op == fsnotify.Chmod {
		return // droits ou dates : rien de nouveau à indexer
	}
	w.mu.Lock()
	libID, root := w.libraryOf(ev.Name)
	w.mu.Unlock()
	if libID == 0 {
		return
	}
	rel, err := filepath.Rel(root, ev.Name)
	if err != nil {
		return
	}
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		if part != "." && scanner.SkipDir(part) { // « . » = la racine elle-même (disque débranché)
			return // @eaDir (miniatures DSM), #recycle, .AppleDouble… : bruit, pas de contenu
		}
	}
	w.mu.Lock()
	_, wasDir := w.dirs[ev.Name]
	if wasDir && (ev.Has(fsnotify.Remove) || ev.Has(fsnotify.Rename)) {
		w.forget(ev.Name) // le noyau a déjà retiré le watch
	}
	w.mu.Unlock()
	isDir := false
	if ev.Has(fsnotify.Create) {
		if st, err := os.Stat(ev.Name); err == nil && st.IsDir() {
			isDir = true
			w.addTree(ev.Name, libID) // les fichiers déjà copiés dedans seront vus par l'analyse
		}
	}
	ext := strings.ToLower(filepath.Ext(ev.Name))
	if isDir || wasDir || parse.VideoExtensions[ext] || parse.SubtitleExtensions[ext] {
		w.mu.Lock()
		w.mark(libID)
		w.mu.Unlock()
	}
}

// libraryOf : bibliothèque dont la racine contient path (appelé sous verrou).
func (w *Watcher) libraryOf(path string) (int64, string) {
	var best int64
	bestRoot := ""
	for id, root := range w.roots {
		if (path == root || strings.HasPrefix(path, root+string(filepath.Separator))) && len(root) > len(bestRoot) {
			best, bestRoot = id, root
		}
	}
	return best, bestRoot
}

func (w *Watcher) mark(libID int64) {
	now := time.Now()
	if p := w.pending[libID]; p != nil {
		p.last = now
		return
	}
	w.pending[libID] = &pendingScan{first: now, last: now}
}

// flush lance les analyses dont le calme est atteint (ou qui attendent depuis trop longtemps).
func (w *Watcher) flush() {
	now := time.Now()
	var ids []int64
	w.mu.Lock()
	for id, p := range w.pending {
		if now.Sub(p.last) >= w.Settle || now.Sub(p.first) >= w.MaxWait {
			ids = append(ids, id)
			delete(w.pending, id)
		}
	}
	w.mu.Unlock()
	if len(ids) > 0 {
		slog.Info("changement détecté : analyse programmée", "libraries", ids)
		w.Enqueue(ids...)
	}
}

// addTree surveille dir et ses sous-dossiers. Le parcours (long sur une grosse bibliothèque) se fait hors
// verrou : seuls les ajouts de watches le prennent, pour ne pas bloquer les évènements ni le diagnostic.
func (w *Watcher) addTree(dir string, libID int64) {
	var list []string
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		if path != dir && scanner.SkipDir(d.Name()) {
			return filepath.SkipDir
		}
		list = append(list, path)
		return nil
	})
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.roots[libID] == "" {
		return // bibliothèque retirée pendant le parcours
	}
	for _, path := range list {
		if _, ok := w.dirs[path]; ok {
			continue
		}
		if err := w.w.Add(path); err != nil {
			if errors.Is(err, syscall.ENOSPC) || errors.Is(err, syscall.EMFILE) {
				if !w.limited {
					slog.Warn("limite de surveillance du système atteinte : détection immédiate partielle, l'analyse périodique reste active",
						"dossiers", len(w.dirs))
				}
				w.limited = true
				return
			}
			continue // dossier illisible ou déjà disparu : ignoré, comme pour l'analyse
		}
		w.dirs[path] = libID
	}
}

// forget retire dir et ses sous-dossiers du suivi (appelé sous verrou).
func (w *Watcher) forget(dir string) {
	for d := range w.dirs {
		if d == dir || strings.HasPrefix(d, dir+string(filepath.Separator)) {
			w.w.Remove(d) // erreur ignorée : le noyau l'a souvent déjà retiré
			delete(w.dirs, d)
		}
	}
}

// Sync aligne la surveillance sur les bibliothèques et sur l'état réel du noyau : nouvelles, supprimées,
// réglage désactivé, disque débranché puis rebranché, surveillance partielle à retenter.
func (w *Watcher) Sync(ctx context.Context) {
	if w.w == nil {
		return
	}
	w.syncMu.Lock()
	defer w.syncMu.Unlock()
	enabled := w.Enabled == nil || w.Enabled(ctx)
	libs := map[int64]string{}
	if enabled {
		rows, err := w.DB.QueryContext(ctx, `SELECT id, path FROM libraries`)
		if err != nil {
			return
		}
		for rows.Next() {
			var id int64
			var p string
			if rows.Scan(&id, &p) == nil {
				libs[id] = filepath.Clean(p)
			}
		}
		rows.Close()
	}

	w.mu.Lock()
	// Démontage (IN_UNMOUNT/IN_IGNORED) : fsnotify retire ses watches sans évènement. On recale donc w.dirs
	// sur la liste réelle, sinon un disque rebranché paraîtrait encore surveillé.
	kernel := map[string]bool{}
	for _, p := range w.w.WatchList() {
		kernel[p] = true
	}
	for d := range w.dirs {
		if !kernel[d] {
			delete(w.dirs, d)
		}
	}
	for id, root := range w.roots {
		if libs[id] != root {
			w.forget(root)
			delete(w.roots, id)
			delete(w.devs, id)
			delete(w.pending, id)
		}
	}
	// Surveillance partielle : nouvel essai complet au plus une fois par heure (une bibliothèque a pu être
	// retirée), car il reparcourt tous les dossiers et réveillerait les disques à chaque Sync.
	retry := w.limited && time.Since(w.retried) >= time.Hour
	if retry {
		w.limited = false
		w.retried = time.Now()
	}
	type todo struct {
		id   int64
		root string
	}
	var add []todo
	for id, root := range libs {
		w.roots[id] = root
		st, err := os.Stat(root)
		if err != nil || !st.IsDir() {
			w.forget(root) // disque débranché : réessayé au prochain Sync
			continue
		}
		// Racine sur un autre périphérique qu'au dernier passage : disque rebranché par-dessus un point de
		// montage resté surveillé. Les anciens watches ne voient rien du nouveau montage.
		if dev := deviceOf(st); w.devs[id] != dev {
			w.forget(root)
			w.devs[id] = dev
		}
		if _, watched := w.dirs[root]; !watched || retry {
			add = append(add, todo{id, root})
		}
	}
	w.mu.Unlock()
	for _, t := range add {
		w.addTree(t.root, t.id)
	}
}

// Status renvoie l'état de la surveillance.
func (w *Watcher) Status() Status {
	w.mu.Lock()
	defer w.mu.Unlock()
	return Status{Active: len(w.dirs) > 0, Dirs: len(w.dirs), Limited: w.limited}
}
