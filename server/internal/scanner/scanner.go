// Package scanner indexe les bibliothèques : parcours incrémental, détection des ajouts / suppressions /
// déplacements, identification par le nom, puis analyse ffprobe en arrière-plan avec concurrence limitée.
// Il ne modifie, ne déplace et ne supprime jamais un fichier vidéo : il ne fait que lire.
package scanner

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"aurora/internal/catalog"
	"aurora/internal/events"
	"aurora/internal/parse"
	"aurora/internal/probe"
)

// Library : bibliothèque à analyser.
type Library struct {
	ID   int64
	Name string
	Kind string // movies | series
	Path string
}

// Progress : état publié pendant une analyse.
type Progress struct {
	TaskID    int64  `json:"taskId"`
	LibraryID int64  `json:"libraryId"`
	Library   string `json:"library"`
	Phase     string `json:"phase"` // walk | identify | probe | metadata | done
	Done      int    `json:"done"`
	Total     int    `json:"total"`
	Errors    int    `json:"errors"`
	Current   string `json:"current"`
	Status    string `json:"status"` // running | done | failed | cancelled
	Message   string `json:"message"`
}

// AfterScanFunc : appelé après l'analyse d'une bibliothèque (ex. enrichissement des métadonnées).
type AfterScanFunc func(ctx context.Context, lib Library, report func(Progress)) error

// Scanner coordonne les analyses. Une seule analyse à la fois : préserver les disques et le CPU du NAS
// compte plus que la vitesse.
type Scanner struct {
	DB        *sql.DB
	FFprobe   string
	Hub       *events.Hub
	AfterScan AfterScanFunc

	probeJobs atomic.Int32 // analyses ffprobe simultanées (modifiable à chaud depuis les réglages)

	mu       sync.Mutex
	running  *runState
	queue    []int64
	reserved map[int64]bool // bibliothèques en cours de modification (Exclusive) : jamais mises en file
}

type runState struct {
	progress Progress
	cancel   context.CancelFunc
}

// SetProbeJobs règle le nombre d'analyses ffprobe simultanées.
func (s *Scanner) SetProbeJobs(n int) { s.probeJobs.Store(int32(n)) }

// Exclusive exécute fn pendant qu'aucune analyse de la bibliothèque n'est en cours ni en attente, et
// empêche qu'une analyse de cette bibliothèque démarre pendant fn. La bibliothèque est seulement
// « réservée » : le verrou du planificateur n'est pas tenu pendant fn (qui ouvre une transaction),
// sinon une analyse d'une autre bibliothèque, elle-même en transaction, attendrait ce verrou.
func (s *Scanner) Exclusive(libID int64, fn func() error) error {
	s.mu.Lock()
	busy := s.running != nil && s.running.progress.LibraryID == libID
	for _, q := range s.queue {
		busy = busy || q == libID
	}
	if busy || s.reserved[libID] {
		s.mu.Unlock()
		return ErrBusy
	}
	if s.reserved == nil {
		s.reserved = map[int64]bool{}
	}
	s.reserved[libID] = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.reserved, libID)
		s.mu.Unlock()
	}()
	return fn()
}

// ErrBusy : une analyse est déjà en cours.
var ErrBusy = errors.New("une analyse est déjà en cours")

// RecoverInterrupted marque comme interrompues les tâches restées « running » après un arrêt brutal.
// Les fichiers non encore analysés gardent probe_state='pending' et seront repris à la prochaine analyse.
func (s *Scanner) RecoverInterrupted(ctx context.Context) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE tasks SET status='interrupted', ended_at=? WHERE status='running'`, time.Now().Unix())
	return err
}

// Current renvoie la progression en cours, ou nil.
func (s *Scanner) Current() *Progress {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running == nil {
		return nil
	}
	p := s.running.progress
	return &p
}

// Queued renvoie les bibliothèques en attente.
func (s *Scanner) Queued() []int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]int64(nil), s.queue...)
}

// Cancel interrompt l'analyse en cours (les fichiers restants seront analysés à la prochaine analyse)
// et vide la file d'attente.
func (s *Scanner) Cancel() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queue = nil
	if s.running == nil {
		return false
	}
	s.running.cancel()
	return true
}

// Enqueue ajoute des bibliothèques à la file et démarre le traitement si nécessaire.
func (s *Scanner) Enqueue(libIDs ...int64) {
	s.mu.Lock()
	for _, id := range libIDs {
		if s.reserved[id] {
			continue // en cours de modification : l'appelant relance l'analyse ensuite
		}
		dup := false
		for _, q := range s.queue {
			dup = dup || q == id
		}
		// En cours d'analyse : on ne l'ignore pas, on la reprogramme (un ajout détecté pendant l'analyse
		// peut tomber dans un dossier déjà parcouru). La file empêche les doublons.
		if !dup {
			s.queue = append(s.queue, id)
		}
	}
	start := s.running == nil && len(s.queue) > 0
	if start {
		s.running = &runState{} // réservé avant le lancement pour éviter deux boucles concurrentes
	}
	s.mu.Unlock()
	if start {
		go s.loop()
	}
}

func (s *Scanner) loop() {
	for {
		s.mu.Lock()
		if len(s.queue) == 0 {
			s.running = nil
			s.mu.Unlock()
			return
		}
		id := s.queue[0]
		s.queue = s.queue[1:]
		ctx, cancel := context.WithCancel(context.Background())
		s.running = &runState{cancel: cancel, progress: Progress{LibraryID: id, Status: "running", Phase: "walk"}}
		s.mu.Unlock()

		lib, err := s.loadLibrary(ctx, id)
		if err != nil {
			slog.Error("bibliothèque introuvable", "id", id, "err", err)
			cancel()
			continue
		}
		s.scan(ctx, lib)
		cancel()
	}
}

func (s *Scanner) loadLibrary(ctx context.Context, id int64) (Library, error) {
	var l Library
	err := s.DB.QueryRowContext(ctx, `SELECT id, name, kind, path FROM libraries WHERE id = ?`, id).
		Scan(&l.ID, &l.Name, &l.Kind, &l.Path)
	return l, err
}

// publish met à jour la progression et la diffuse, au plus ~4 fois par seconde sauf changement de phase/état.
func (s *Scanner) publish(p Progress, force bool) {
	s.mu.Lock()
	if s.running != nil {
		s.running.progress = p
	}
	s.mu.Unlock()
	if s.Hub != nil && (force || throttle.allow()) {
		s.Hub.Publish("scan", p)
	}
}

type throttler struct {
	mu   sync.Mutex
	last time.Time
}

var throttle = &throttler{}

func (t *throttler) allow() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if time.Since(t.last) < 250*time.Millisecond {
		return false
	}
	t.last = time.Now()
	return true
}

type existingFile struct {
	id         int64
	size       int64
	mtime      int64
	present    bool
	itemID     sql.NullInt64
	probeState string
	extSubs    string
}

type foundFile struct {
	path  string
	size  int64
	mtime int64
}

func (s *Scanner) scan(ctx context.Context, lib Library) {
	start := time.Now()
	res, err := s.DB.ExecContext(ctx, `INSERT INTO tasks(kind, library_id, status, started_at, message) VALUES ('scan', ?, 'running', ?, ?)`,
		lib.ID, start.Unix(), lib.Name)
	if err != nil {
		slog.Error("création de tâche impossible", "err", err)
		return
	}
	taskID, _ := res.LastInsertId()
	p := Progress{TaskID: taskID, LibraryID: lib.ID, Library: lib.Name, Phase: "walk", Status: "running"}
	s.publish(p, true)

	var errs atomic.Int64 // incrémenté depuis plusieurs goroutines d'analyse
	logErr := func(path string, e error) {
		errs.Add(1)
		s.DB.ExecContext(context.Background(), `INSERT INTO task_errors(task_id, path, message, at) VALUES (?,?,?,?)`,
			taskID, path, e.Error(), time.Now().Unix())
	}

	status, message := "done", ""
	if err := s.scanLibrary(ctx, lib, &p, logErr); err != nil {
		if errors.Is(err, context.Canceled) {
			status, message = "cancelled", "Analyse interrompue ; elle reprendra là où elle s'est arrêtée."
		} else {
			status, message = "failed", err.Error()
		}
	}
	if status == "done" && s.AfterScan != nil {
		p.Phase = "metadata"
		s.publish(p, true)
		if err := s.AfterScan(ctx, lib, func(mp Progress) {
			p.Done, p.Total, p.Current = mp.Done, mp.Total, mp.Current
			s.publish(p, false)
		}); err != nil {
			if errors.Is(err, context.Canceled) {
				status, message = "cancelled", "Récupération des métadonnées interrompue."
			} else {
				message = "Métadonnées : " + err.Error()
			}
		}
	}
	errCount := int(errs.Load())
	p.Errors = errCount
	if message == "" && errCount > 0 {
		message = fmt.Sprintf("%d fichier(s) en erreur", errCount)
	}
	now := time.Now().Unix()
	bg := context.Background()
	s.DB.ExecContext(bg, `UPDATE tasks SET status=?, ended_at=?, total=?, done=?, errors=?, message=? WHERE id=?`,
		status, now, p.Total, p.Done, errCount, message, taskID)
	s.DB.ExecContext(bg, `UPDATE libraries SET last_scan_at=?, last_scan_status=? WHERE id=?`, now, status, lib.ID)
	p.Phase, p.Status, p.Message, p.Current = "done", status, message, ""
	s.publish(p, true)
	slog.Info("analyse terminée", "library", lib.Name, "status", status, "errors", errCount, "duration", time.Since(start).Round(time.Millisecond))
}

// skipDir : dossiers techniques Synology / macOS / Windows à ignorer.
// SkipDir indique un dossier technique à ignorer (exporté pour la détection du type de bibliothèque).
func SkipDir(name string) bool { return skipDir(name) }

func skipDir(name string) bool {
	if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "@") || strings.HasPrefix(name, "#") {
		return true // .AppleDouble, @eaDir (miniatures Synology), #recycle, #snapshot…
	}
	switch strings.ToLower(name) {
	case "$recycle.bin", "system volume information", "lost+found", "extras", "featurettes", "behind the scenes",
		"deleted scenes", "interviews", "trailers", "bonus":
		return true
	}
	return false
}

func (s *Scanner) scanLibrary(ctx context.Context, lib Library, p *Progress, logErr func(string, error)) error {
	st, err := os.Stat(lib.Path)
	if err != nil {
		return fmt.Errorf("dossier inaccessible (%s) : vérifiez qu'il existe et que l'utilisateur système « aurora » a le droit de lecture", lib.Path)
	}
	if !st.IsDir() {
		return fmt.Errorf("%s n'est pas un dossier", lib.Path)
	}

	// 1. Parcours : uniquement des stat(), aucun fichier vidéo n'est lu à ce stade.
	var found []foundFile
	subsByDir := map[string][]string{}
	walkErr := filepath.WalkDir(lib.Path, func(path string, d fs.DirEntry, err error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			logErr(path, err)
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if path != lib.Path && skipDir(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if parse.SubtitleExtensions[ext] {
			// Fichiers réguliers uniquement : un lien symbolique « Film.fr.vtt -> aurora.db » exposerait
			// n'importe quel fichier lisible par l'utilisateur système.
			if d.Type().IsRegular() {
				subsByDir[filepath.Dir(path)] = append(subsByDir[filepath.Dir(path)], path)
			}
			return nil
		}
		if !parse.IsVideo(path) || !d.Type().IsRegular() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			logErr(path, err)
			return nil
		}
		found = append(found, foundFile{path, info.Size(), info.ModTime().Unix()})
		if len(found)%200 == 0 {
			p.Done, p.Current = len(found), filepath.Base(path)
			s.publish(*p, false)
		}
		return nil
	})
	if walkErr != nil {
		return walkErr
	}

	// 2. Comparaison avec la base.
	existing := map[string]*existingFile{}
	rows, err := s.DB.QueryContext(ctx, `SELECT id, path, size, mtime, present, item_id, probe_state, external_subs
		FROM media_files WHERE library_id = ?`, lib.ID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var e existingFile
		var path string
		if err := rows.Scan(&e.id, &path, &e.size, &e.mtime, &e.present, &e.itemID, &e.probeState, &e.extSubs); err != nil {
			rows.Close()
			return err
		}
		existing[path] = &e
	}
	rows.Close()

	seen := map[string]bool{}
	var added []foundFile
	var toIdentify []int64
	now := time.Now().Unix()
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, f := range found {
		seen[f.path] = true
		e := existing[f.path]
		if e == nil {
			added = append(added, f)
			continue
		}
		subs := externalSubsJSON(f.path, subsByDir[filepath.Dir(f.path)])
		switch {
		case e.size != f.size || e.mtime != f.mtime:
			// Contenu modifié : nouvelle analyse technique.
			_, err = tx.ExecContext(ctx, `UPDATE media_files SET size=?, mtime=?, present=1, missing_since=NULL,
				probe_state='pending', probe_error='', external_subs=? WHERE id=?`, f.size, f.mtime, subs, e.id)
		case !e.present || subs != e.extSubs:
			_, err = tx.ExecContext(ctx, `UPDATE media_files SET present=1, missing_since=NULL, external_subs=? WHERE id=?`, subs, e.id)
		}
		if err != nil {
			return err
		}
		if !e.itemID.Valid {
			toIdentify = append(toIdentify, e.id)
		}
	}

	// Fichiers disparus. Un fichier ajouté de même nom et même taille est considéré comme déplacé :
	// on garde la même ligne, donc l'historique et la progression de lecture.
	// Les autres sont marqués absents, jamais supprimés : s'ils reviennent (disque démonté…), tout est conservé.
	byKey := map[string][]int{}
	for i, a := range added {
		k := filepath.Base(a.path) + "|" + fmt.Sprint(a.size)
		byKey[k] = append(byKey[k], i)
	}
	moved := map[int]bool{}
	var missing []string
	for path, e := range existing {
		if !seen[path] && e.present {
			missing = append(missing, path)
		}
	}
	sort.Strings(missing)
	for _, mp := range missing {
		e := existing[mp]
		k := filepath.Base(mp) + "|" + fmt.Sprint(e.size)
		if idx := byKey[k]; len(idx) > 0 {
			a := added[idx[0]]
			byKey[k] = idx[1:]
			moved[idx[0]] = true
			if _, err := tx.ExecContext(ctx, `UPDATE media_files SET path=?, mtime=?, present=1, missing_since=NULL, external_subs=? WHERE id=?`,
				a.path, a.mtime, externalSubsJSON(a.path, subsByDir[filepath.Dir(a.path)]), e.id); err != nil {
				return err
			}
			slog.Info("déplacement détecté", "from", mp, "to", a.path)
			continue
		}
		if _, err := tx.ExecContext(ctx, `UPDATE media_files SET present=0, missing_since=? WHERE id=?`, now, e.id); err != nil {
			return err
		}
	}
	for i, a := range added {
		if moved[i] {
			continue
		}
		res, err := tx.ExecContext(ctx, `INSERT INTO media_files(library_id, path, size, mtime, external_subs, added_at) VALUES (?,?,?,?,?,?)`,
			lib.ID, a.path, a.size, a.mtime, externalSubsJSON(a.path, subsByDir[filepath.Dir(a.path)]), now)
		if err != nil {
			return err
		}
		id, _ := res.LastInsertId()
		toIdentify = append(toIdentify, id)
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	// 2 bis. Épisodes indexés à tort comme films (versions antérieures) : reclassés avec leur historique.
	var reclass map[int64]int64
	if lib.Kind == "movies" {
		if reclass, err = s.misfiledEpisodes(ctx, lib); err != nil {
			return err
		}
		for id := range reclass {
			toIdentify = append(toIdentify, id)
		}
	}

	// 3. Identification par le nom (rapide, sans lecture de fichier).
	if len(toIdentify) > 0 {
		p.Phase, p.Done, p.Total = "identify", 0, len(toIdentify)
		s.publish(*p, true)
		if err := s.identifyAll(ctx, lib, toIdentify, reclass, p); err != nil {
			return err
		}
	}

	// 4. Analyse technique (ffprobe) des fichiers en attente, y compris ceux d'une analyse interrompue.
	return s.probePending(ctx, lib, p, logErr)
}

// misfiledEpisodes repère les fichiers nommés comme des épisodes mais rattachés à un « film » par une
// version antérieure. Exclus : fichiers rattachés à la main, films verrouillés ou déjà associés à TMDB.
// Renvoie fichier -> ancien film ; l'écriture se fait dans la transaction d'identification.
func (s *Scanner) misfiledEpisodes(ctx context.Context, lib Library) (map[int64]int64, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT f.id, f.path, f.item_id FROM media_files f JOIN items i ON i.id=f.item_id
		WHERE f.library_id=? AND f.manual=0 AND i.kind='movie' AND i.match_state NOT IN ('locked','matched')`, lib.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]int64{}
	for rows.Next() {
		var id, item int64
		var path string
		if rows.Scan(&id, &path, &item) != nil {
			continue
		}
		if _, ok := parse.ParseEpisodeInMovies(lib.Path, path); ok {
			out[id] = item
		}
	}
	return out, rows.Err()
}

// identifyAll identifie les fichiers en une seule transaction. reclass (fichier -> ancien film) : fichiers
// reclassés ; leur historique de lecture est reporté sur l'épisode, puis les films vidés sont retirés.
func (s *Scanner) identifyAll(ctx context.Context, lib Library, ids []int64, reclass map[int64]int64, p *Progress) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	touched := map[int64]bool{}
	emptied := map[int64]bool{}
	for i, id := range ids {
		var path string
		if err := tx.QueryRowContext(ctx, `SELECT path FROM media_files WHERE id=?`, id).Scan(&path); err != nil {
			return err
		}
		old, isReclass := reclass[id]
		if isReclass {
			if _, err := tx.ExecContext(ctx, `UPDATE media_files SET item_id=NULL, manual=0 WHERE id=?`, id); err != nil {
				return err
			}
		}
		itemIDs, err := Identify(ctx, tx, lib, id, path)
		if err != nil {
			return err
		}
		if isReclass && len(itemIDs) == 2 { // [série, épisode]
			shared := 0
			for _, o := range reclass {
				if o == old {
					shared++
				}
			}
			if err := carryHistory(ctx, tx, old, itemIDs[0], itemIDs[1], id, shared == 1); err != nil {
				return err
			}
			emptied[old] = true
		}
		for _, it := range itemIDs {
			touched[it] = true
		}
		if i%100 == 0 {
			p.Done, p.Current = i, filepath.Base(path)
			s.publish(*p, false)
		}
	}
	for id := range touched {
		if err := catalog.ReindexItem(ctx, tx, id); err != nil {
			return err
		}
	}
	// Films vidés par le reclassement : lignes de catalogue retirées (aucun fichier n'est concerné).
	for id := range emptied {
		var n int
		tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM media_files WHERE item_id=?`, id).Scan(&n)
		if n > 0 {
			continue
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM items_fts WHERE rowid=?`, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM items WHERE id=? AND kind='movie'`, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// carryHistory reporte l'historique d'un ancien « film » sur l'épisode reclassé. Plusieurs épisodes
// pouvaient partager le même faux film : la progression et le statut vu ne vont qu'à l'épisode réellement
// lu (last_file_id), ou à l'épisode unique ; favoris et « Ma liste » vont à la série.
func carryHistory(ctx context.Context, tx catalog.Execer, from, series, episode, fileID int64, single bool) error {
	if _, err := tx.ExecContext(ctx, `INSERT INTO user_state(user_id, item_id, position, duration, watched, play_count,
		last_played_at, last_file_id)
		SELECT user_id, ?, position, duration, watched, play_count, last_played_at, last_file_id
		FROM user_state WHERE item_id=? AND (last_file_id=? OR (? AND last_file_id IS NULL))
		ON CONFLICT(user_id, item_id) DO UPDATE SET
		  watched = MAX(watched, excluded.watched), play_count = play_count + excluded.play_count,
		  position = CASE WHEN COALESCE(excluded.last_played_at,0) > COALESCE(last_played_at,0) THEN excluded.position ELSE position END,
		  last_played_at = MAX(COALESCE(last_played_at,0), COALESCE(excluded.last_played_at,0))`,
		episode, from, fileID, single); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO user_state(user_id, item_id, favorite, watchlist, watchlist_at)
		SELECT user_id, ?, favorite, watchlist, watchlist_at FROM user_state WHERE item_id=? AND (favorite=1 OR watchlist=1)
		ON CONFLICT(user_id, item_id) DO UPDATE SET favorite = MAX(favorite, excluded.favorite),
		  watchlist = MAX(watchlist, excluded.watchlist), watchlist_at = COALESCE(watchlist_at, excluded.watchlist_at)`,
		series, from)
	return err
}

// Identify rattache un fichier à une œuvre d'après son nom. Renvoie les items créés ou modifiés.
// Un fichier de série dont le numéro d'épisode n'est pas reconnu reste non identifié (item_id NULL)
// plutôt que d'être rattaché au hasard : il apparaît dans « Contenus non identifiés ».
func Identify(ctx context.Context, tx catalog.Execer, lib Library, fileID int64, path string) ([]int64, error) {
	now := time.Now().Unix()
	stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	switch lib.Kind {
	case "movies":
		// Un fichier nommé comme un épisode (S01E06) rangé parmi des films devient un épisode de sa série :
		// les bibliothèques mêlant films et séries sont courantes, et un épisode n'est jamais un film.
		if ep, ok := parse.ParseEpisodeInMovies(lib.Path, path); ok {
			return identifyEpisode(ctx, tx, lib, fileID, ep, now)
		}
		m := parse.ParseMovie(lib.Path, path)
		if m.Title == "" {
			m.Title = stem
		}
		key := parse.Normalize(m.Title) + "|" + yearKey(m.Year)
		itemID, err := getOrCreate(ctx, tx, lib.ID, "movie", nil, key, m.Title, m.Year, now, nil)
		if err != nil {
			return nil, err
		}
		_, err = tx.ExecContext(ctx, `UPDATE media_files SET item_id=?, version_label=?, edition=? WHERE id=?`,
			itemID, parse.VersionLabel(m.Tags, m.Edition), m.Edition, fileID)
		return []int64{itemID}, err
	case "series":
		ep, ok := parse.ParseEpisode(lib.Path, path)
		if !ok {
			return nil, nil
		}
		return identifyEpisode(ctx, tx, lib, fileID, ep, now)
	}
	return nil, fmt.Errorf("type de bibliothèque inconnu : %s", lib.Kind)
}

func identifyEpisode(ctx context.Context, tx catalog.Execer, lib Library, fileID int64, ep parse.Episode, now int64) ([]int64, error) {
	seriesKey := parse.Normalize(ep.Series) + "|" + yearKey(ep.Year)
	seriesID, err := getOrCreate(ctx, tx, lib.ID, "series", nil, seriesKey, ep.Series, ep.Year, now, nil)
	if err != nil {
		return nil, err
	}
	title := ep.Title
	if title == "" {
		title = fmt.Sprintf("Épisode %d", ep.Episode)
	}
	epKey := fmt.Sprintf("%d|%d|%d", seriesID, ep.Season, ep.Episode)
	epID, err := getOrCreate(ctx, tx, lib.ID, "episode", &seriesID, epKey, title, 0, now, &ep)
	if err != nil {
		return nil, err
	}
	_, err = tx.ExecContext(ctx, `UPDATE media_files SET item_id=?, version_label=? WHERE id=?`,
		epID, parse.VersionLabel(ep.Tags, ""), fileID)
	return []int64{seriesID, epID}, err
}

func yearKey(y int) string {
	if y == 0 {
		return ""
	}
	return fmt.Sprint(y)
}

func getOrCreate(ctx context.Context, tx catalog.Execer, libID int64, kind string, parent *int64, key, title string,
	year int, now int64, ep *parse.Episode) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `SELECT id FROM items WHERE library_id=? AND kind=? AND match_key=?`, libID, kind, key).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	var yearV, season, episode, episodeEnd any
	if year > 0 {
		yearV = year
	}
	if ep != nil {
		season, episode = ep.Season, ep.Episode
		if ep.EpisodeEnd > 0 {
			episodeEnd = ep.EpisodeEnd
		}
	}
	res, err := tx.ExecContext(ctx, `INSERT INTO items(library_id, kind, parent_id, match_key, title, sort_title, year,
		season, episode, episode_end, added_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		libID, kind, parent, key, title, parse.SortTitle(title), yearV, season, episode, episodeEnd, now, now)
	if err != nil {
		return 0, err
	}
	id, err = res.LastInsertId()
	if err == nil && parent != nil {
		// Classification héritée de la série dès la création (contrôle parental).
		_, err = tx.ExecContext(ctx, `UPDATE items SET content_rating=(SELECT content_rating FROM items WHERE id=?),
			min_age=(SELECT min_age FROM items WHERE id=?) WHERE id=?`, *parent, *parent, id)
	}
	return id, err
}

type probeJob struct {
	id   int64
	path string
}

func (s *Scanner) probePending(ctx context.Context, lib Library, p *Progress, logErr func(string, error)) error {
	rows, err := s.DB.QueryContext(ctx, `SELECT id, path FROM media_files WHERE library_id=? AND present=1 AND probe_state='pending' ORDER BY id`, lib.ID)
	if err != nil {
		return err
	}
	var jobs []probeJob
	for rows.Next() {
		var j probeJob
		if rows.Scan(&j.id, &j.path) == nil {
			jobs = append(jobs, j)
		}
	}
	rows.Close()
	if len(jobs) == 0 {
		return nil
	}
	if s.FFprobe == "" {
		p.Message = "ffprobe est introuvable : l'analyse technique (codecs, pistes) est reportée."
		s.publish(*p, true)
		return nil
	}
	p.Phase, p.Done, p.Total = "probe", 0, len(jobs)
	s.publish(*p, true)

	n := int(s.probeJobs.Load())
	if n <= 0 {
		n = 2
	}
	ch := make(chan probeJob)
	var wg sync.WaitGroup
	var done atomic.Int64
	var mu sync.Mutex
	for w := 0; w < n; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range ch {
				r, err := probe.Run(ctx, s.FFprobe, j.path)
				if ctx.Err() != nil {
					return // annulé : le fichier reste « pending » et sera repris
				}
				if err != nil {
					logErr(j.path, err)
					s.DB.ExecContext(context.Background(), `UPDATE media_files SET probe_state='error', probe_error=?, probed_at=? WHERE id=?`,
						err.Error(), time.Now().Unix(), j.id)
				} else if err := SaveProbe(context.Background(), s.DB, j.id, r); err != nil {
					logErr(j.path, err)
				}
				d := done.Add(1)
				mu.Lock()
				p.Done, p.Current = int(d), filepath.Base(j.path)
				s.publish(*p, false)
				mu.Unlock()
			}
		}()
	}
feed:
	for _, j := range jobs {
		select {
		case ch <- j:
		case <-ctx.Done():
			break feed
		}
	}
	close(ch)
	wg.Wait()
	return ctx.Err()
}

// SaveProbe enregistre le résultat d'une analyse ffprobe.
func SaveProbe(ctx context.Context, db catalog.Execer, fileID int64, r *probe.Result) error {
	audio, _ := json.Marshal(r.Audio)
	subs, _ := json.Marshal(r.Subtitles)
	chapters, _ := json.Marshal(r.Chapters)
	_, err := db.ExecContext(ctx, `UPDATE media_files SET container=?, duration=?, bitrate=?, video_stream=?, video_codec=?, video_profile=?,
		pix_fmt=?, width=?, height=?, hdr=?, audio=?, subtitles=?, chapters=?, probe_state='ok', probe_error='', probed_at=?
		WHERE id=?`, r.Container, r.Duration, r.Bitrate, nullStream(r.VideoStream), r.VideoCodec, r.VideoProfile, r.PixFmt, r.Width, r.Height, r.HDR,
		string(audio), string(subs), string(chapters), time.Now().Unix(), fileID)
	return err
}

// externalSubsJSON liste les sous-titres voisins appartenant à la vidéo : même nom de base, suivi
// éventuellement d'un code de langue / « forced » (Film.fr.srt, Film.en.forced.srt).
func externalSubsJSON(video string, candidates []string) string {
	base := strings.TrimSuffix(filepath.Base(video), filepath.Ext(video))
	var out []probe.Subtitle
	for _, c := range candidates {
		name := filepath.Base(c)
		stem := strings.TrimSuffix(name, filepath.Ext(name))
		if stem != base && !strings.HasPrefix(stem, base+".") {
			continue
		}
		rest := strings.TrimPrefix(strings.TrimPrefix(stem, base), ".")
		sub := probe.Subtitle{Stream: -1, External: c, Codec: strings.TrimPrefix(strings.ToLower(filepath.Ext(c)), "."), Text: true}
		for _, part := range strings.Split(strings.ToLower(rest), ".") {
			switch {
			case part == "forced":
				sub.Forced = true
			case part == "default":
				sub.Default = true
			case part == "sdh" || part == "cc" || part == "hi":
				sub.Title = "SDH"
			case len(part) == 2 || len(part) == 3:
				sub.Language = part
			}
		}
		out = append(out, sub)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].External < out[j].External })
	if len(out) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(out)
	return string(b)
}

func nullStream(i int) any {
	if i < 0 {
		return nil
	}
	return i
}
