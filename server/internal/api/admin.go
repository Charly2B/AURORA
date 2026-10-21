package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"aurora/internal/auth"
	"aurora/internal/catalog"
	"aurora/internal/config"
	"aurora/internal/db"
	"aurora/internal/metadata"
	"aurora/internal/parse"
	"aurora/internal/scanner"
)

// --- Assistant de premier démarrage --------------------------------------------------------------

var setupSteps = map[string]bool{"language": true, "libraries": true, "metadata": true, "scan": true, "done": true}

func (s *Server) handleGetSetupStep(w http.ResponseWriter, r *http.Request, u *auth.User) {
	writeJSON(w, http.StatusOK, map[string]any{"step": s.setting(r.Context(), "setup_step", "language"),
		"language": s.setting(r.Context(), "language", "fr-FR")})
}

func (s *Server) handlePutSetupStep(w http.ResponseWriter, r *http.Request, u *auth.User) {
	var req struct {
		Step     string `json:"step"`
		Language string `json:"language"`
	}
	if err := readJSON(r, &req); err != nil {
		fail(w, r, err)
		return
	}
	if !setupSteps[req.Step] {
		fail(w, r, badRequest("étape inconnue"))
		return
	}
	ctx := r.Context()
	if req.Language != "" {
		if req.Language != "fr-FR" && req.Language != "en-US" {
			fail(w, r, badRequest("langue non prise en charge"))
			return
		}
		if err := s.setSetting(ctx, "language", req.Language); err != nil {
			fail(w, r, err)
			return
		}
	}
	if err := s.setSetting(ctx, "setup_step", req.Step); err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"step": req.Step})
}

// --- Exploration des dossiers (assistant) --------------------------------------------------------

// within : vrai si p est dans l'une des racines autorisées (liens symboliques résolus).
func (s *Server) within(p string) (string, bool) {
	real, err := filepath.EvalSymlinks(p)
	if err != nil {
		return "", false
	}
	for _, root := range s.Cfg.Roots() {
		rr, err := filepath.EvalSymlinks(root)
		if err != nil {
			continue
		}
		if real == rr || strings.HasPrefix(real, rr+string(filepath.Separator)) {
			return real, true
		}
	}
	return "", false
}

func readable(p string) bool {
	f, err := os.Open(p)
	if err != nil {
		return false
	}
	defer f.Close()
	_, err = f.Readdirnames(1)
	return err == nil || errors.Is(err, io.EOF)
}

func (s *Server) handleBrowse(w http.ResponseWriter, r *http.Request, u *auth.User) {
	p := r.URL.Query().Get("path")
	type entry struct {
		Name     string `json:"name"`
		Path     string `json:"path"`
		Readable bool   `json:"readable"`
	}
	if p == "" {
		var out []entry
		for _, root := range s.Cfg.Roots() {
			out = append(out, entry{filepath.Base(root), root, readable(root)})
		}
		writeJSON(w, http.StatusOK, map[string]any{"path": "", "parent": nil, "entries": out})
		return
	}
	if !filepath.IsAbs(p) {
		fail(w, r, badRequest("chemin absolu attendu"))
		return
	}
	real, ok := s.within(filepath.Clean(p))
	if !ok {
		fail(w, r, &httpError{http.StatusForbidden, "dossier hors des volumes autorisés ou inaccessible"})
		return
	}
	des, err := os.ReadDir(real)
	if err != nil {
		fail(w, r, &httpError{http.StatusForbidden, "lecture impossible : " + humanFSError(err) +
			". Accordez la lecture à l'utilisateur « aurora » (Panneau de configuration > Dossier partagé > Modifier > Permissions > Utilisateurs internes du système)."})
		return
	}
	out := []entry{}
	for _, d := range des {
		n := d.Name()
		if !d.IsDir() || strings.HasPrefix(n, ".") || strings.HasPrefix(n, "@") || strings.HasPrefix(n, "#") {
			continue
		}
		full := filepath.Join(real, n)
		out = append(out, entry{n, full, readable(full)})
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	var parent any
	if _, ok := s.within(filepath.Dir(real)); ok && filepath.Dir(real) != real {
		parent = filepath.Dir(real)
	}
	writeJSON(w, http.StatusOK, map[string]any{"path": real, "parent": parent, "entries": out})
}

// --- Bibliothèques -------------------------------------------------------------------------------

func (s *Server) handleAdminLibraries(w http.ResponseWriter, r *http.Request, u *auth.User) {
	rows, err := s.DB.QueryContext(r.Context(), `SELECT l.id, l.name, l.kind, l.path, l.created_at, COALESCE(l.last_scan_at,0), l.last_scan_status,
		(SELECT COUNT(*) FROM media_files f WHERE f.library_id=l.id AND f.present=1),
		(SELECT COUNT(*) FROM media_files f WHERE f.library_id=l.id AND f.present=0),
		(SELECT COUNT(*) FROM media_files f WHERE f.library_id=l.id AND f.present=1 AND f.item_id IS NULL),
		(SELECT COUNT(*) FROM media_files f WHERE f.library_id=l.id AND f.present=1 AND f.probe_state='pending')
		FROM libraries l ORDER BY l.name`)
	if err != nil {
		fail(w, r, err)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, created, last int64
		var name, kind, path, status string
		var files, missing, unident, pending int
		if rows.Scan(&id, &name, &kind, &path, &created, &last, &status, &files, &missing, &unident, &pending) == nil {
			lib := map[string]any{"id": id, "name": name, "kind": kind, "path": path, "createdAt": created,
				"lastScanAt": last, "lastScanStatus": status, "files": files, "missing": missing, "unidentified": unident,
				"pending": pending, "accessible": readable(path)}
			out = append(out, lib)
		}
	}
	rows.Close()
	// Bibliothèque dont le contenu contredit le type (ex. séries ajoutées comme films) : suggestion de correction.
	// Les noms sont lus en base (aucun accès disque) : épisodes reconnus parmi les fichiers présents.
	for _, lib := range out {
		if hint := s.kindHint(r.Context(), lib["id"].(int64), lib["kind"].(string), lib["path"].(string)); hint != "" {
			lib["suggestKind"] = hint
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) kindHint(ctx context.Context, id int64, kind, root string) string {
	rows, err := s.DB.QueryContext(ctx, `SELECT path FROM media_files WHERE library_id=? AND present=1 LIMIT 400`, id)
	if err != nil {
		return ""
	}
	defer rows.Close()
	eps, total := 0, 0
	for rows.Next() {
		var p string
		if rows.Scan(&p) != nil {
			continue
		}
		total++
		if _, ok := parse.ParseEpisode(root, p); ok {
			eps++
		}
	}
	// Des épisodes dans une bibliothèque de films sont rangés automatiquement en séries : pas d'alerte.
	// Des films dans une bibliothèque de séries restent non identifiés : là, le type doit être corrigé.
	if kind == "series" && total >= 3 && eps*5 < total {
		return "movies"
	}
	return ""
}

type libraryIn struct {
	Name  string `json:"name"`
	Kind  string `json:"kind"`
	Path  string `json:"path"`
	Force bool   `json:"force"` // ajouter malgré un contenu qui semble d'un autre type
}

// guessKind échantillonne les noms de vidéos d'un dossier : combien ressemblent à des épisodes (S01E03…) ?
// Lecture des noms uniquement (aucun fichier ouvert), 400 vidéos au plus.
func guessKind(ctx context.Context, root string) (episodes, total int) {
	seen := 0
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		seen++
		if ctx.Err() != nil || seen > 20000 { // borne : un partage immense n'est jamais parcouru en entier
			return fs.SkipAll
		}
		if err != nil {
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if p != root && scanner.SkipDir(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if !parse.IsVideo(p) {
			return nil
		}
		total++
		if _, ok := parse.ParseEpisode(root, p); ok {
			episodes++
		}
		if total >= 400 {
			return fs.SkipAll
		}
		return nil
	})
	return
}

// kindMismatch renvoie un message si le contenu du dossier contredit le type choisi.
func kindMismatch(ctx context.Context, kind, root string) (string, string, int, int) {
	eps, total := guessKind(ctx, root)
	switch {
	case kind == "movies" && eps >= 2 && eps*2 >= total:
		return fmt.Sprintf("Ce dossier semble contenir des séries : %d fichier(s) sur %d sont nommés comme des épisodes (S01E01…).", eps, total), "series", eps, total
	case kind == "series" && total >= 3 && eps*5 < total:
		return fmt.Sprintf("Ce dossier semble contenir des films : seulement %d fichier(s) sur %d sont nommés comme des épisodes.", eps, total), "movies", eps, total
	}
	return "", "", eps, total
}

func (s *Server) validateLibraryPath(ctx context.Context, p string, exceptID int64) (string, error) {
	if !filepath.IsAbs(p) {
		return "", badRequest("chemin absolu attendu")
	}
	real, ok := s.within(filepath.Clean(p))
	if !ok {
		return "", badRequest("ce dossier n'existe pas ou n'est pas dans un volume autorisé")
	}
	if st, err := os.Stat(real); err != nil || !st.IsDir() {
		return "", badRequest("ce chemin n'est pas un dossier")
	}
	if !readable(real) {
		return "", badRequest("AURORA n'a pas le droit de lire ce dossier. Dans DSM : Panneau de configuration > Dossier partagé > " +
			"Modifier > Permissions > « Utilisateurs internes du système » > aurora : Lecture seule.")
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT id, path FROM libraries WHERE id<>?`, exceptID)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var other string
		rows.Scan(&id, &other)
		sep := string(filepath.Separator)
		if other == real || strings.HasPrefix(real, other+sep) || strings.HasPrefix(other, real+sep) {
			return "", badRequest("ce dossier recoupe la bibliothèque existante « %s »", other)
		}
	}
	return real, nil
}

func (s *Server) handleCreateLibrary(w http.ResponseWriter, r *http.Request, u *auth.User) {
	var in libraryIn
	if err := readJSON(r, &in); err != nil {
		fail(w, r, err)
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Kind != "movies" && in.Kind != "series" {
		fail(w, r, badRequest("type de bibliothèque invalide"))
		return
	}
	ctx := r.Context()
	real, err := s.validateLibraryPath(ctx, in.Path, 0)
	if err != nil {
		fail(w, r, err)
		return
	}
	if in.Name == "" {
		in.Name = filepath.Base(real)
	}
	if !in.Force {
		if msg, suggest, eps, total := kindMismatch(ctx, in.Kind, real); msg != "" {
			writeJSON(w, http.StatusConflict, map[string]any{"error": msg, "suggestKind": suggest, "episodes": eps, "total": total})
			return
		}
	}
	res, err := s.DB.ExecContext(ctx, `INSERT INTO libraries(name, kind, path, created_at) VALUES (?,?,?,?)`, in.Name, in.Kind, real, time.Now().Unix())
	if err != nil {
		fail(w, r, err)
		return
	}
	id, _ := res.LastInsertId()
	s.Scanner.Enqueue(id)
	s.syncWatcher()
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "path": real})
}

// handleUpdateLibrary renomme une bibliothèque et/ou change son type (films <-> séries). Changer de type
// efface uniquement l'identification faite par AURORA (œuvres, progression associée) puis relance une
// analyse ; les fichiers vidéo ne sont jamais touchés et leur analyse technique est conservée.
func (s *Server) handleUpdateLibrary(w http.ResponseWriter, r *http.Request, u *auth.User) {
	id, err := pathID(r, "id")
	if err != nil {
		fail(w, r, err)
		return
	}
	var in libraryIn
	if err := readJSON(r, &in); err != nil {
		fail(w, r, err)
		return
	}
	ctx := r.Context()
	var curName, curKind string
	if err := s.DB.QueryRowContext(ctx, `SELECT name, kind FROM libraries WHERE id=?`, id).Scan(&curName, &curKind); err != nil {
		fail(w, r, catalog.ErrNotFound)
		return
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = curName
	}
	if in.Kind == "" || in.Kind == curKind {
		if _, err := s.DB.ExecContext(ctx, `UPDATE libraries SET name=? WHERE id=?`, name, id); err != nil {
			fail(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}
	if in.Kind != "movies" && in.Kind != "series" {
		fail(w, r, badRequest("type de bibliothèque invalide"))
		return
	}
	var backup string
	err = s.Scanner.Exclusive(id, func() error {
		// Filet de sécurité : progression, favoris et corrections de cette bibliothèque vont être effacés du
		// catalogue ; la base complète est sauvegardée avant (restaurable depuis Maintenance).
		backup = "avant-changement-type-" + time.Now().Format("20060102-150405.000") + ".db"
		if _, err := s.DB.BackupTo(ctx, backup); err != nil {
			return err
		}
		tx, err := s.DB.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		for _, q := range []string{
			`DELETE FROM items_fts WHERE rowid IN (SELECT id FROM items WHERE library_id=?)`,
			`UPDATE media_files SET item_id=NULL, version_label='', edition='', manual=0 WHERE library_id=?`,
			`DELETE FROM items WHERE library_id=?`, // lignes de catalogue seulement (progression, crédits en cascade)
		} {
			if _, err := tx.ExecContext(ctx, q, id); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, `UPDATE libraries SET name=?, kind=? WHERE id=?`, name, in.Kind, id); err != nil {
			return err
		}
		return tx.Commit()
	})
	if errors.Is(err, scanner.ErrBusy) {
		fail(w, r, &httpError{http.StatusConflict, "analyse en cours ou prévue sur cette bibliothèque : réessayez quand elle sera terminée"})
		return
	}
	if err != nil {
		fail(w, r, err)
		return
	}
	s.Scanner.Enqueue(id)
	s.syncWatcher()
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "rescan": true, "backup": backup})
}

// handleDeleteLibrary retire la bibliothèque du catalogue AURORA. Les fichiers vidéo ne sont jamais touchés.
func (s *Server) handleDeleteLibrary(w http.ResponseWriter, r *http.Request, u *auth.User) {
	id, err := pathID(r, "id")
	if err != nil {
		fail(w, r, err)
		return
	}
	if cur := s.Scanner.Current(); cur != nil && cur.LibraryID == id {
		fail(w, r, &httpError{http.StatusConflict, "analyse en cours sur cette bibliothèque : arrêtez-la d'abord"})
		return
	}
	ctx := r.Context()
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		fail(w, r, err)
		return
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM items_fts WHERE rowid IN (SELECT id FROM items WHERE library_id=?)`, id); err != nil {
		fail(w, r, err)
		return
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM libraries WHERE id=?`, id); err != nil {
		fail(w, r, err)
		return
	}
	if err := tx.Commit(); err != nil {
		fail(w, r, err)
		return
	}
	s.syncWatcher()
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleScan(w http.ResponseWriter, r *http.Request, u *auth.User) {
	var req struct {
		LibraryID int64 `json:"libraryId"`
	}
	readJSON(r, &req)
	var ids []int64
	if req.LibraryID > 0 {
		ids = []int64{req.LibraryID}
	} else {
		rows, err := s.DB.QueryContext(r.Context(), `SELECT id FROM libraries ORDER BY id`)
		if err != nil {
			fail(w, r, err)
			return
		}
		for rows.Next() {
			var id int64
			rows.Scan(&id)
			ids = append(ids, id)
		}
		rows.Close()
	}
	s.Scanner.Enqueue(ids...)
	writeJSON(w, http.StatusAccepted, map[string]any{"queued": ids})
}

func (s *Server) handleScanCancel(w http.ResponseWriter, r *http.Request, u *auth.User) {
	writeJSON(w, http.StatusOK, map[string]any{"cancelled": s.Scanner.Cancel()})
}

func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request, u *auth.User) {
	rows, err := s.DB.QueryContext(r.Context(), `SELECT t.id, t.kind, COALESCE(l.name,''), t.status, t.started_at, COALESCE(t.ended_at,0),
		t.total, t.done, t.errors, t.message FROM tasks t LEFT JOIN libraries l ON l.id=t.library_id ORDER BY t.id DESC LIMIT 50`)
	if err != nil {
		fail(w, r, err)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, started, ended int64
		var kind, lib, status, msg string
		var total, done, errs int
		if rows.Scan(&id, &kind, &lib, &status, &started, &ended, &total, &done, &errs, &msg) == nil {
			out = append(out, map[string]any{"id": id, "kind": kind, "library": lib, "status": status, "startedAt": started,
				"endedAt": ended, "total": total, "done": done, "errors": errs, "message": msg})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"tasks": out, "current": s.Scanner.Current(), "queued": s.Scanner.Queued()})
}

func (s *Server) handleTaskErrors(w http.ResponseWriter, r *http.Request, u *auth.User) {
	id, err := pathID(r, "id")
	if err != nil {
		fail(w, r, err)
		return
	}
	rows, err := s.DB.QueryContext(r.Context(), `SELECT path, message, at FROM task_errors WHERE task_id=? ORDER BY at LIMIT 500`, id)
	if err != nil {
		fail(w, r, err)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var p, m string
		var at int64
		if rows.Scan(&p, &m, &at) == nil {
			out = append(out, map[string]any{"path": p, "message": m, "at": at})
		}
	}
	writeJSON(w, http.StatusOK, out)
}

// --- Correction des identifications --------------------------------------------------------------

func (s *Server) handleUnidentified(w http.ResponseWriter, r *http.Request, u *auth.User) {
	ctx := r.Context()
	files := []map[string]any{}
	rows, err := s.DB.QueryContext(ctx, `SELECT f.id, f.path, l.name, l.kind, l.path FROM media_files f JOIN libraries l ON l.id=f.library_id
		WHERE f.present=1 AND f.item_id IS NULL ORDER BY f.path LIMIT 500`)
	if err != nil {
		fail(w, r, err)
		return
	}
	for rows.Next() {
		var id int64
		var p, lib, kind, root string
		if rows.Scan(&id, &p, &lib, &kind, &root) == nil {
			rel, _ := filepath.Rel(root, p)
			files = append(files, map[string]any{"id": id, "path": rel, "library": lib, "kind": kind})
		}
	}
	rows.Close()
	items := []map[string]any{}
	rows, err = s.DB.QueryContext(ctx, `SELECT i.id, i.kind, i.title, COALESCE(i.year,0), i.match_state, l.name FROM items i JOIN libraries l ON l.id=i.library_id
		WHERE i.kind IN ('movie','series') AND i.match_state IN ('ambiguous','failed')
		AND (EXISTS (SELECT 1 FROM media_files f WHERE f.item_id=i.id AND f.present=1)
		  OR EXISTS (SELECT 1 FROM items e JOIN media_files f ON f.item_id=e.id WHERE e.parent_id=i.id AND f.present=1))
		ORDER BY i.sort_title LIMIT 500`)
	if err != nil {
		fail(w, r, err)
		return
	}
	for rows.Next() {
		var id int64
		var kind, title, state, lib string
		var year int
		if rows.Scan(&id, &kind, &title, &year, &state, &lib) == nil {
			items = append(items, map[string]any{"id": id, "kind": kind, "title": title, "year": year, "state": state, "library": lib})
		}
	}
	rows.Close()
	writeJSON(w, http.StatusOK, map[string]any{"files": files, "items": items})
}

// handleAssignFile rattache manuellement un fichier : à une œuvre existante, ou à un titre/épisode saisi.
func (s *Server) handleAssignFile(w http.ResponseWriter, r *http.Request, u *auth.User) {
	id, err := pathID(r, "id")
	if err != nil {
		fail(w, r, err)
		return
	}
	var req struct {
		ItemID  int64  `json:"itemId"`
		Title   string `json:"title"`
		Year    int    `json:"year"`
		Season  *int   `json:"season"`
		Episode int    `json:"episode"`
	}
	if err := readJSON(r, &req); err != nil {
		fail(w, r, err)
		return
	}
	ctx := r.Context()
	var lib scanner.Library
	var path string
	if err := s.DB.QueryRowContext(ctx, `SELECT l.id, l.name, l.kind, l.path, f.path FROM media_files f JOIN libraries l ON l.id=f.library_id WHERE f.id=?`, id).
		Scan(&lib.ID, &lib.Name, &lib.Kind, &lib.Path, &path); err != nil {
		fail(w, r, catalog.ErrNotFound)
		return
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		fail(w, r, err)
		return
	}
	defer tx.Rollback()
	var target int64
	switch {
	case req.ItemID > 0:
		var okKind bool
		tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM items WHERE id=? AND library_id=? AND kind IN ('movie','episode'))`, req.ItemID, lib.ID).Scan(&okKind)
		if !okKind {
			fail(w, r, badRequest("œuvre cible invalide (même bibliothèque, film ou épisode)"))
			return
		}
		target = req.ItemID
	case strings.TrimSpace(req.Title) != "":
		// On réutilise l'identification standard en lui fournissant un nom normalisé.
		title := strings.TrimSpace(req.Title)
		fake := filepath.Join(lib.Path, fmt.Sprintf("%s (%d)", title, req.Year), "x.mkv")
		if req.Year == 0 {
			fake = filepath.Join(lib.Path, title, "x.mkv")
		}
		if lib.Kind == "series" {
			if req.Season == nil || req.Episode <= 0 {
				fail(w, r, badRequest("saison et épisode requis"))
				return
			}
			fake = filepath.Join(lib.Path, title, fmt.Sprintf("x.S%02dE%02d.mkv", *req.Season, req.Episode))
		}
		ids, err := scanner.Identify(ctx, tx, lib, id, fake)
		if err != nil || len(ids) == 0 {
			fail(w, r, badRequest("identification impossible avec ces informations"))
			return
		}
		target = ids[len(ids)-1]
		for _, it := range ids {
			if err := catalog.ReindexItem(ctx, tx, it); err != nil {
				fail(w, r, err)
				return
			}
		}
	default:
		fail(w, r, badRequest("indiquez une œuvre ou un titre"))
		return
	}
	// Le libellé de version reste celui du vrai nom de fichier.
	m := parse.ParseMovie(lib.Path, path)
	if _, err := tx.ExecContext(ctx, `UPDATE media_files SET item_id=?, version_label=?, edition=?, manual=1 WHERE id=?`,
		target, parse.VersionLabel(m.Tags, m.Edition), m.Edition, id); err != nil {
		fail(w, r, err)
		return
	}
	if err := tx.Commit(); err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"itemId": target})
}

func (s *Server) tmdb(ctx context.Context) *metadata.Client {
	key := s.setting(ctx, "tmdb_api_key", "")
	if key == "" {
		return nil
	}
	c := metadata.NewClient(key, s.setting(ctx, "language", "fr-FR"), s.ImageDir)
	if s.TMDBBase != "" {
		c.BaseURL, c.ImageURL = s.TMDBBase, s.TMDBImages
	}
	return c
}

func (s *Server) handleCandidates(w http.ResponseWriter, r *http.Request, u *auth.User) {
	id, err := pathID(r, "id")
	if err != nil {
		fail(w, r, err)
		return
	}
	ctx := r.Context()
	c := s.tmdb(ctx)
	if c == nil {
		fail(w, r, badRequest("aucune clé TMDB : configurez-la dans Administration > Métadonnées"))
		return
	}
	var kind, title string
	var year int
	if err := s.DB.QueryRowContext(ctx, `SELECT kind, title, COALESCE(year,0) FROM items WHERE id=?`, id).Scan(&kind, &title, &year); err != nil {
		fail(w, r, catalog.ErrNotFound)
		return
	}
	q := r.URL.Query().Get("q")
	if q == "" {
		q = title
	}
	if y := qInt(r, "year"); y > 0 {
		year = y
	} else if r.URL.Query().Has("q") {
		year = 0
	}
	tk := "movie"
	if kind == "series" {
		tk = "tv"
	}
	res, err := c.Search(ctx, tk, q, year)
	if err != nil {
		fail(w, r, &httpError{http.StatusBadGateway, err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// handleMatch applique une fiche TMDB choisie par l'administrateur ; la correspondance est verrouillée.
func (s *Server) handleMatch(w http.ResponseWriter, r *http.Request, u *auth.User) {
	id, err := pathID(r, "id")
	if err != nil {
		fail(w, r, err)
		return
	}
	var req struct {
		TmdbID int `json:"tmdbId"`
	}
	if err := readJSON(r, &req); err != nil || req.TmdbID <= 0 {
		fail(w, r, badRequest("identifiant TMDB invalide"))
		return
	}
	ctx := r.Context()
	c := s.tmdb(ctx)
	if c == nil {
		fail(w, r, badRequest("aucune clé TMDB configurée"))
		return
	}
	var kind string
	if err := s.DB.QueryRowContext(ctx, `SELECT kind FROM items WHERE id=?`, id).Scan(&kind); err != nil {
		fail(w, r, catalog.ErrNotFound)
		return
	}
	var target int64
	switch kind {
	case "movie":
		target, err = s.Enricher.ApplyMovie(ctx, c, id, req.TmdbID, "locked")
	case "series":
		target, err = s.Enricher.ApplySeries(ctx, c, id, req.TmdbID, "locked")
	default:
		err = badRequest("seuls les films et les séries peuvent être associés")
	}
	if err != nil {
		var he *httpError
		if errors.As(err, &he) {
			fail(w, r, err)
			return
		}
		fail(w, r, &httpError{http.StatusBadGateway, err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"itemId": target})
}

// handleEditItem : correction manuelle du titre/année ; verrouille l'œuvre contre toute modification automatique.
func (s *Server) handleEditItem(w http.ResponseWriter, r *http.Request, u *auth.User) {
	id, err := pathID(r, "id")
	if err != nil {
		fail(w, r, err)
		return
	}
	var req struct {
		Title    string `json:"title"`
		Year     int    `json:"year"`
		Overview string `json:"overview"`
	}
	if err := readJSON(r, &req); err != nil {
		fail(w, r, err)
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" || len(req.Title) > 300 {
		fail(w, r, badRequest("titre invalide"))
		return
	}
	var year any
	if req.Year > 1800 && req.Year < 2200 {
		year = req.Year
	}
	ctx := r.Context()
	res, err := s.DB.ExecContext(ctx, `UPDATE items SET title=?, sort_title=?, year=?, overview=CASE WHEN ?<>'' THEN ? ELSE overview END,
		match_state='locked', updated_at=? WHERE id=?`, req.Title, parse.SortTitle(req.Title), year, req.Overview, req.Overview, time.Now().Unix(), id)
	if err != nil {
		fail(w, r, err)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		fail(w, r, catalog.ErrNotFound)
		return
	}
	catalog.ReindexItem(ctx, s.DB, id)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// --- Choix d'affiche / de fond ------------------------------------------------------------------

// artworkTarget renvoie le type TMDB et l'identifiant d'une œuvre associée (film ou série).
func (s *Server) artworkTarget(ctx context.Context, id int64) (string, int, error) {
	var kind string
	var tmdb sql.NullInt64
	if err := s.DB.QueryRowContext(ctx, `SELECT kind, tmdb_id FROM items WHERE id=?`, id).Scan(&kind, &tmdb); err != nil {
		return "", 0, catalog.ErrNotFound
	}
	if kind != "movie" && kind != "series" {
		return "", 0, badRequest("seuls les films et les séries ont une affiche à choisir")
	}
	if !tmdb.Valid {
		return "", 0, badRequest("associez d'abord la bonne fiche TMDB (« Corriger l'identification »)")
	}
	if kind == "series" {
		return "tv", int(tmdb.Int64), nil
	}
	return "movie", int(tmdb.Int64), nil
}

func (s *Server) handleArtworkList(w http.ResponseWriter, r *http.Request, u *auth.User) {
	id, err := pathID(r, "id")
	if err != nil {
		fail(w, r, err)
		return
	}
	ctx := r.Context()
	c := s.tmdb(ctx)
	if c == nil {
		fail(w, r, badRequest("aucune clé TMDB configurée"))
		return
	}
	kind, tmdbID, err := s.artworkTarget(ctx, id)
	if err != nil {
		fail(w, r, err)
		return
	}
	posters, backdrops, err := c.Images(ctx, kind, tmdbID)
	if err != nil {
		fail(w, r, &httpError{http.StatusBadGateway, err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"posters": posters, "backdrops": backdrops})
}

// tmdbImagePath : chemin d'image TMDB strict (aucune URL arbitraire ne peut être téléchargée par le NAS).
var tmdbImagePath = regexp.MustCompile(`^/[A-Za-z0-9_-]{5,64}\.(jpg|jpeg|png)$`)

// handleArtworkSet applique l'affiche ou le fond choisi. Le choix est conservé par les mises à jour
// automatiques (elles ne repassent pas sur une œuvre associée) ; une nouvelle identification manuelle
// (autre œuvre) remplace naturellement affiche et fond.
func (s *Server) handleArtworkSet(w http.ResponseWriter, r *http.Request, u *auth.User) {
	id, err := pathID(r, "id")
	if err != nil {
		fail(w, r, err)
		return
	}
	var req struct {
		Type string `json:"type"` // poster | backdrop
		Path string `json:"path"`
	}
	if err := readJSON(r, &req); err != nil {
		fail(w, r, err)
		return
	}
	if (req.Type != "poster" && req.Type != "backdrop") || !tmdbImagePath.MatchString(req.Path) {
		fail(w, r, badRequest("image invalide"))
		return
	}
	ctx := r.Context()
	c := s.tmdb(ctx)
	if c == nil {
		fail(w, r, badRequest("aucune clé TMDB configurée"))
		return
	}
	if _, _, err := s.artworkTarget(ctx, id); err != nil {
		fail(w, r, err)
		return
	}
	size := "w500"
	if req.Type == "backdrop" {
		size = "w1280"
	}
	name, err := c.Image(ctx, size, req.Path)
	if err != nil {
		fail(w, r, &httpError{http.StatusBadGateway, "téléchargement de l'image impossible : " + err.Error()})
		return
	}
	col := "poster"
	if req.Type == "backdrop" {
		col = "backdrop"
	}
	if _, err := s.DB.ExecContext(ctx, `UPDATE items SET `+col+`=?, updated_at=? WHERE id=?`, name, time.Now().Unix(), id); err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"url": catalog.ImageURL(name)})
}

// --- Réglages ------------------------------------------------------------------------------------

func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request, u *auth.User) {
	ctx := r.Context()
	writeJSON(w, http.StatusOK, map[string]any{
		"tmdbConfigured":    s.setting(ctx, "tmdb_api_key", "") != "",
		"language":          s.setting(ctx, "language", "fr-FR"),
		"scanIntervalHours": atoiDef(s.setting(ctx, "scan_interval_hours", "6"), 6),
		"probeJobs":         atoiDef(s.setting(ctx, "probe_jobs", "2"), 2),
		"maxTranscodes":     atoiDef(s.setting(ctx, "max_transcodes", "1"), 1),
		"allow4kTranscode":  s.setting(ctx, "allow_4k_transcode", "0") == "1",
		"watchEnabled":      s.setting(ctx, "watch_enabled", "1") != "0",
	})
}

// WatchEnabled : réglage « détection immédiate des ajouts » (activé par défaut, comme Plex).
func (s *Server) WatchEnabled(ctx context.Context) bool {
	return s.setting(ctx, "watch_enabled", "1") != "0"
}

// syncWatcher réaligne la surveillance des dossiers après un changement de bibliothèques ou de réglage.
// En tâche de fond : le parcours d'une grosse bibliothèque prend quelques secondes.
func (s *Server) syncWatcher() {
	if s.Watcher == nil {
		return
	}
	go s.Watcher.Sync(context.Background())
}

func atoiDef(s string, def int) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

func (s *Server) handlePutSettings(w http.ResponseWriter, r *http.Request, u *auth.User) {
	var req struct {
		TmdbAPIKey        *string `json:"tmdbApiKey"`
		Language          *string `json:"language"`
		ScanIntervalHours *int    `json:"scanIntervalHours"`
		ProbeJobs         *int    `json:"probeJobs"`
		MaxTranscodes     *int    `json:"maxTranscodes"`
		Allow4kTranscode  *bool   `json:"allow4kTranscode"`
		WatchEnabled      *bool   `json:"watchEnabled"`
	}
	if err := readJSON(r, &req); err != nil {
		fail(w, r, err)
		return
	}
	ctx := r.Context()
	if req.TmdbAPIKey != nil {
		key := strings.TrimSpace(*req.TmdbAPIKey)
		if key != "" {
			c := metadata.NewClient(key, "fr-FR", s.ImageDir)
			vctx, cancel := context.WithTimeout(ctx, 15*time.Second)
			err := c.Validate(vctx)
			cancel()
			if err != nil {
				fail(w, r, badRequest("clé TMDB non validée : %s", err.Error()))
				return
			}
		}
		if err := s.setSetting(ctx, "tmdb_api_key", key); err != nil {
			fail(w, r, err)
			return
		}
	}
	if req.Language != nil {
		if *req.Language != "fr-FR" && *req.Language != "en-US" {
			fail(w, r, badRequest("langue non prise en charge"))
			return
		}
		s.setSetting(ctx, "language", *req.Language)
	}
	if req.WatchEnabled != nil {
		v := "0"
		if *req.WatchEnabled {
			v = "1"
		}
		s.setSetting(ctx, "watch_enabled", v)
		s.syncWatcher()
	}
	if req.ScanIntervalHours != nil {
		if *req.ScanIntervalHours < 0 || *req.ScanIntervalHours > 168 {
			fail(w, r, badRequest("intervalle entre 0 (désactivé) et 168 heures"))
			return
		}
		s.setSetting(ctx, "scan_interval_hours", strconv.Itoa(*req.ScanIntervalHours))
	}
	if req.ProbeJobs != nil {
		if *req.ProbeJobs < 1 || *req.ProbeJobs > 4 {
			fail(w, r, badRequest("analyses simultanées : de 1 à 4"))
			return
		}
		s.setSetting(ctx, "probe_jobs", strconv.Itoa(*req.ProbeJobs))
		s.Scanner.SetProbeJobs(*req.ProbeJobs)
	}
	if req.MaxTranscodes != nil {
		if *req.MaxTranscodes < 0 || *req.MaxTranscodes > 3 {
			fail(w, r, badRequest("conversions simultanées : de 0 à 3"))
			return
		}
		s.setSetting(ctx, "max_transcodes", strconv.Itoa(*req.MaxTranscodes))
		s.Play.SetMaxTranscodes(*req.MaxTranscodes)
	}
	if req.Allow4kTranscode != nil {
		v := "0"
		if *req.Allow4kTranscode {
			v = "1"
		}
		s.setSetting(ctx, "allow_4k_transcode", v)
	}
	s.handleGetSettings(w, r, u)
}

// --- Sauvegardes ---------------------------------------------------------------------------------

var backupName = regexp.MustCompile(`^[A-Za-z0-9._-]+\.db$`)

func (s *Server) handleBackups(w http.ResponseWriter, r *http.Request, u *auth.User) {
	list, err := s.DB.ListBackups()
	if err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleCreateBackup(w http.ResponseWriter, r *http.Request, u *auth.User) {
	name := "aurora-" + time.Now().Format("20060102-150405") + ".db"
	p, err := s.DB.BackupTo(r.Context(), name)
	if err != nil {
		fail(w, r, err)
		return
	}
	st, _ := os.Stat(p)
	var size int64
	if st != nil {
		size = st.Size()
	}
	writeJSON(w, http.StatusCreated, db.BackupInfo{Name: name, Size: size, ModTime: time.Now().Unix()})
}

func (s *Server) handleDownloadBackup(w http.ResponseWriter, r *http.Request, u *auth.User) {
	name := r.PathValue("name")
	if !backupName.MatchString(name) {
		http.NotFound(w, r)
		return
	}
	p := filepath.Join(s.DB.BackupsDir, name)
	if _, err := os.Stat(p); err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, p)
}

func (s *Server) handleRestoreBackup(w http.ResponseWriter, r *http.Request, u *auth.User) {
	name := r.PathValue("name")
	if !backupName.MatchString(name) {
		fail(w, r, badRequest("nom invalide"))
		return
	}
	// Filet de sécurité : l'état actuel est sauvegardé avant de programmer la restauration.
	if _, err := s.DB.BackupTo(r.Context(), "avant-restauration-"+time.Now().Format("20060102-150405")+".db"); err != nil {
		fail(w, r, err)
		return
	}
	if err := s.DB.ScheduleRestore(name); err != nil {
		fail(w, r, badRequest("%s", err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true,
		"message": "Restauration programmée : redémarrez AURORA depuis le Centre de paquets pour l'appliquer."})
}

// --- Tableau de bord, journaux, diagnostics ------------------------------------------------------

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request, u *auth.User) {
	ctx := r.Context()
	count := func(q string) int {
		var n int
		s.DB.QueryRowContext(ctx, q).Scan(&n)
		return n
	}
	var lastScan sql.NullInt64
	s.DB.QueryRowContext(ctx, `SELECT MAX(last_scan_at) FROM libraries`).Scan(&lastScan)
	tc, rx := s.Play.Count()
	var errorsList []map[string]any
	rows, err := s.DB.QueryContext(ctx, `SELECT t.id, COALESCE(l.name,''), t.status, t.message, t.errors, COALESCE(t.ended_at, t.started_at)
		FROM tasks t LEFT JOIN libraries l ON l.id=t.library_id WHERE t.status IN ('failed','interrupted') OR t.errors > 0 ORDER BY t.id DESC LIMIT 5`)
	if err == nil {
		for rows.Next() {
			var id, at int64
			var lib, status, msg string
			var n int
			if rows.Scan(&id, &lib, &status, &msg, &n, &at) == nil {
				errorsList = append(errorsList, map[string]any{"taskId": id, "library": lib, "status": status, "message": msg, "errors": n, "at": at})
			}
		}
		rows.Close()
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"version":      config.Version,
		"startedAt":    s.StartedAt.Unix(),
		"movies":       count(`SELECT COUNT(*) FROM items i WHERE kind='movie' AND EXISTS (SELECT 1 FROM media_files f WHERE f.item_id=i.id AND f.present=1)`),
		"series":       count(`SELECT COUNT(*) FROM items i WHERE kind='series' AND EXISTS (SELECT 1 FROM items e JOIN media_files f ON f.item_id=e.id WHERE e.parent_id=i.id AND f.present=1)`),
		"episodes":     count(`SELECT COUNT(*) FROM items i WHERE kind='episode' AND EXISTS (SELECT 1 FROM media_files f WHERE f.item_id=i.id AND f.present=1)`),
		"files":        count(`SELECT COUNT(*) FROM media_files WHERE present=1`),
		"missing":      count(`SELECT COUNT(*) FROM media_files WHERE present=0`),
		"unidentified": count(`SELECT COUNT(*) FROM media_files WHERE present=1 AND item_id IS NULL`) + count(`SELECT COUNT(*) FROM items WHERE match_state IN ('ambiguous')`),
		"probeErrors":  count(`SELECT COUNT(*) FROM media_files WHERE present=1 AND probe_state='error'`),
		"pending":      count(`SELECT COUNT(*) FROM media_files WHERE present=1 AND probe_state='pending'`),
		"libraries":    count(`SELECT COUNT(*) FROM libraries`),
		"users":        count(`SELECT COUNT(*) FROM users`),
		"lastScanAt":   lastScan.Int64,
		"scan":         s.Scanner.Current(),
		"transcodes":   tc,
		"remuxes":      rx,
		"ffmpeg":       s.Tools.FFmpeg != "",
		"ffprobe":      s.Tools.FFprobe != "",
		"tmdb":         s.setting(ctx, "tmdb_api_key", "") != "",
		"errors":       errorsList,
	})
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request, u *auth.User) {
	if s.Log == nil {
		writeJSON(w, http.StatusOK, map[string]any{"text": "Journal envoyé sur la sortie standard (mode développement)."})
		return
	}
	t, err := s.Log.Tail(256 << 10)
	if err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"text": t})
}

func (s *Server) handleDiagnostics(w http.ResponseWriter, r *http.Request, u *auth.User) {
	ctx := r.Context()
	integrity, err := s.DB.IntegrityCheck(ctx)
	if err != nil {
		integrity = "erreur : " + err.Error()
	}
	schema, _ := s.DB.SchemaVersion(ctx)
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	dbSize := int64(0)
	if st, err := os.Stat(s.DB.Path); err == nil {
		dbSize = st.Size()
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"version": config.Version, "go": runtime.Version(), "os": runtime.GOOS + "/" + runtime.GOARCH, "cpus": runtime.NumCPU(),
		"goroutines": runtime.NumGoroutine(), "memoryMB": ms.Sys / (1 << 20), "heapMB": ms.HeapAlloc / (1 << 20),
		"dataDir": s.Cfg.DataDir, "tmpDir": s.Cfg.TmpDir, "browseRoots": s.Cfg.Roots(), "listen": s.Cfg.Listen,
		"dbIntegrity": integrity, "schemaVersion": schema, "dbSize": dbSize,
		"tools": s.Tools, "tonemap": s.Play.HasTonemap,
		"watch": func() any {
			if s.Watcher == nil {
				return map[string]any{"active": false, "unavailable": true}
			}
			return s.Watcher.Status()
		}(),
		"limits": func() map[string]any {
			l := s.Play.CurrentLimits()
			return map[string]any{"maxTranscodes": l.MaxTranscodes, "maxRemux": l.MaxRemux, "transcodeThreads": l.TranscodeThreads}
		}(),
	})
}

// handleEvents : flux SSE (progression des analyses).
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request, u *auth.User) {
	fl, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "flux non pris en charge")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Accel-Buffering", "no") // reverse proxy nginx de DSM : pas de mise en tampon
	ch, unsub := s.Hub.Subscribe()
	defer unsub()
	if cur := s.Scanner.Current(); cur != nil {
		b, _ := json.Marshal(map[string]any{"type": "scan", "data": cur})
		fmt.Fprintf(w, "data: %s\n\n", b)
	}
	fl.Flush()
	tick := time.NewTicker(25 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case b, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", b)
			fl.Flush()
		case <-tick.C:
			fmt.Fprint(w, ": ping\n\n")
			fl.Flush()
		}
	}
}
