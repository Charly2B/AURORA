package playback

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Limits : garde-fous de charge pour le NAS.
type Limits struct {
	MaxTranscodes    int // conversions vidéo simultanées (défaut 1)
	MaxRemux         int // remux simultanés (défaut 4)
	TranscodeThreads int // fils ffmpeg par conversion (défaut : moitié des cœurs logiques)
}

// Manager : sessions HLS en cours (une par lecture remux/transcodage).
type Manager struct {
	FFmpeg  string
	TmpRoot string // dossier de travail des segments ; vidé au démarrage
	Limits  Limits
	// Filters disponibles dans ffmpeg (zscale/tonemap pour le HDR -> SDR).
	HasTonemap bool

	mu       sync.Mutex
	sessions map[string]*Session
}

// Session : un processus ffmpeg qui produit une playlist HLS fMP4.
type Session struct {
	ID      string
	UserID  int64
	FileID  int64
	Mode    Mode
	Offset  float64
	Dir     string
	cmd     *exec.Cmd // protégé par mu
	cancel  context.CancelFunc
	started chan struct{} // fermé quand le lancement de ffmpeg a abouti ou échoué
	done    chan struct{} // fermé quand ffmpeg s'est terminé
	err     string
	mu      sync.Mutex
	lastSeg int // dernier segment demandé par le client
	purged  int // segments d'index < purged effacés (déjà vus)
	lastHit time.Time
	paused  bool
}

// ErrLimit : trop de sessions simultanées.
var ErrLimit = errors.New("limite atteinte")

// NewManager prépare le dossier de travail (nettoyé des restes d'une exécution précédente).
func NewManager(ffmpeg, tmpRoot string, lim Limits) (*Manager, error) {
	if lim.MaxTranscodes < 0 {
		lim.MaxTranscodes = 1
	}
	if lim.MaxRemux <= 0 {
		lim.MaxRemux = 4
	}
	if lim.TranscodeThreads <= 0 {
		lim.TranscodeThreads = max(1, runtime.NumCPU()/2)
	}
	dir := filepath.Join(tmpRoot, "hls")
	// Les segments sont des fichiers temporaires créés par AURORA lui-même dans son dossier de travail.
	if err := os.RemoveAll(dir); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}
	m := &Manager{FFmpeg: ffmpeg, TmpRoot: dir, Limits: lim, sessions: map[string]*Session{}}
	go m.janitor()
	return m, nil
}

// CurrentLimits renvoie une copie des limites (lecture concurrente sûre).
func (m *Manager) CurrentLimits() Limits {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Limits
}

// SetMaxTranscodes modifie la limite de conversions simultanées (0 = conversions interdites).
func (m *Manager) SetMaxTranscodes(n int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Limits.MaxTranscodes = n
}

// Count renvoie le nombre de sessions actives par mode.
func (m *Manager) Count() (transcodes, remux int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.sessions {
		if s.Mode == ModeTranscode {
			transcodes++
		} else {
			remux++
		}
	}
	return
}

// StartRequest : paramètres d'une nouvelle session.
type StartRequest struct {
	UserID      int64
	FileID      int64
	Path        string
	Decision    Decision
	File        File
	VideoStream int
	Start       float64
	BurnSubIdx  int // index du flux de sous-titres image à incruster, -1 = aucun
}

// Start lance ffmpeg. Une session précédente du même utilisateur sur le même fichier est arrêtée
// (changement de position ou de piste).
func (m *Manager) Start(req StartRequest) (*Session, error) {
	if m.FFmpeg == "" {
		return nil, errors.New("ffmpeg est introuvable sur le serveur : seule la lecture directe est possible")
	}
	m.mu.Lock()
	for id, s := range m.sessions {
		if s.UserID == req.UserID && s.FileID == req.FileID {
			delete(m.sessions, id)
			go s.stop()
		}
	}
	tc, rx := 0, 0
	for _, s := range m.sessions {
		if s.Mode == ModeTranscode {
			tc++
		} else {
			rx++
		}
	}
	if req.Decision.Mode == ModeTranscode && m.Limits.MaxTranscodes == 0 {
		m.mu.Unlock()
		return nil, fmt.Errorf("%w : les conversions vidéo sont désactivées dans les réglages", ErrLimit)
	}
	if req.Decision.Mode == ModeTranscode && tc >= m.Limits.MaxTranscodes {
		m.mu.Unlock()
		return nil, fmt.Errorf("%w : %d conversion(s) vidéo déjà en cours, le NAS ne peut pas en assurer davantage", ErrLimit, tc)
	}
	if req.Decision.Mode == ModeRemux && rx >= m.Limits.MaxRemux {
		m.mu.Unlock()
		return nil, fmt.Errorf("%w : %d flux déjà en cours", ErrLimit, rx)
	}
	b := make([]byte, 12)
	rand.Read(b)
	// Tout ce dont stop() a besoin existe avant que la session soit visible des autres requêtes :
	// un arrêt concurrent ne peut plus laisser un ffmpeg orphelin.
	ctx, cancel := context.WithCancel(context.Background())
	s := &Session{ID: hex.EncodeToString(b), UserID: req.UserID, FileID: req.FileID, Mode: req.Decision.Mode,
		Offset: req.Start, done: make(chan struct{}), started: make(chan struct{}), cancel: cancel,
		lastHit: time.Now(), lastSeg: -1}
	s.Dir = filepath.Join(m.TmpRoot, s.ID)
	m.sessions[s.ID] = s
	m.mu.Unlock()

	fail := func(err error) (*Session, error) {
		close(s.started)
		close(s.done)
		m.remove(s.ID)
		return nil, err
	}
	if err := os.MkdirAll(s.Dir, 0o750); err != nil {
		return fail(err)
	}
	cmd := exec.CommandContext(ctx, m.FFmpeg, m.buildArgs(req, s.Dir)...)
	cmd.Dir = s.Dir
	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		// Contexte annulé entre-temps (session déjà arrêtée) ou ffmpeg introuvable.
		return fail(fmt.Errorf("démarrage de ffmpeg impossible : %w", err))
	}
	s.mu.Lock()
	s.cmd = cmd
	s.mu.Unlock()
	close(s.started)
	slog.Info("session de lecture", "session", s.ID, "mode", s.Mode, "file", req.FileID, "start", req.Start)
	go func() {
		var last []string
		sc := bufio.NewScanner(stderr)
		for sc.Scan() {
			last = append(last, sc.Text())
			if len(last) > 5 {
				last = last[1:]
			}
		}
		err := cmd.Wait()
		s.mu.Lock()
		if err != nil && ctx.Err() == nil {
			s.err = strings.Join(last, " | ")
			if s.err == "" {
				s.err = err.Error()
			}
			slog.Warn("ffmpeg terminé en erreur", "session", s.ID, "err", s.err)
		}
		s.mu.Unlock()
		close(s.done)
	}()
	return s, nil
}

func (m *Manager) buildArgs(req StartRequest, dir string) []string {
	f := req.File
	a := []string{"-hide_banner", "-loglevel", "error", "-nostdin"}
	if req.Start > 0 {
		a = append(a, "-ss", strconv.FormatFloat(req.Start, 'f', 3, 64))
	}
	a = append(a, "-i", req.Path, "-map", fmt.Sprintf("0:%d", req.VideoStream))
	if req.Decision.AudioStream >= 0 {
		a = append(a, "-map", fmt.Sprintf("0:%d", req.Decision.AudioStream))
	}
	switch req.Decision.Mode {
	case ModeTranscode:
		var vf []string
		if f.HDR != "" && m.HasTonemap {
			// HDR -> SDR : sans cette conversion, les couleurs apparaîtraient délavées sur un écran SDR.
			vf = append(vf, "zscale=t=linear:npl=100", "format=gbrpf32le", "zscale=p=bt709",
				"tonemap=tonemap=hable:desat=0", "zscale=t=bt709:m=bt709:r=tv")
		}
		vf = append(vf, "scale=-2:'min(1080,ih)'", "format=yuv420p")
		if req.BurnSubIdx >= 0 {
			// Incrustation d'un sous-titre image : le filtre overlay remplace la chaîne -vf.
			fc := fmt.Sprintf("[0:%d][0:%d]overlay,%s[v]", req.VideoStream, req.BurnSubIdx, strings.Join(vf, ","))
			a = replaceVideoMap(a, req.VideoStream)
			a = append(a, "-filter_complex", fc)
		} else {
			a = append(a, "-vf", strings.Join(vf, ","))
		}
		a = append(a, "-c:v", "libx264", "-preset", "veryfast", "-crf", "21", "-maxrate", "12M", "-bufsize", "24M",
			"-profile:v", "high", "-g", "96", "-keyint_min", "96", "-sc_threshold", "0",
			"-threads", strconv.Itoa(m.Limits.TranscodeThreads))
	default:
		a = append(a, "-c:v", "copy")
		if f.VideoCodec == "hevc" {
			a = append(a, "-tag:v", "hvc1") // requis par Safari pour le HEVC en fMP4
		}
	}
	if req.Decision.AudioStream >= 0 {
		if req.Decision.CopyAudio {
			a = append(a, "-c:a", "copy")
		} else {
			a = append(a, "-c:a", "aac", "-b:a", "192k", "-ac", "2")
		}
	}
	a = append(a, "-sn", "-dn", "-map_metadata", "-1", "-map_chapters", "-1",
		"-f", "hls", "-hls_time", "4", "-hls_playlist_type", "event", "-hls_segment_type", "fmp4",
		"-hls_fmp4_init_filename", "init.mp4", "-hls_segment_filename", "seg%05d.m4s",
		"-hls_flags", "independent_segments+temp_file", "index.m3u8")
	return a
}

// replaceVideoMap remplace « -map 0:<video> » par « -map [v] » quand un filter_complex produit la vidéo.
func replaceVideoMap(a []string, videoStream int) []string {
	want := fmt.Sprintf("0:%d", videoStream)
	for i := 0; i+1 < len(a); i++ {
		if a[i] == "-map" && a[i+1] == want {
			a[i+1] = "[v]"
			break
		}
	}
	return a
}

// Get renvoie une session si elle appartient à l'utilisateur.
func (m *Manager) Get(id string, userID int64) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.sessions[id]
	if s == nil || s.UserID != userID {
		return nil
	}
	return s
}

// Stop arrête une session.
func (m *Manager) Stop(id string, userID int64) {
	m.mu.Lock()
	s := m.sessions[id]
	if s != nil && s.UserID == userID {
		delete(m.sessions, id)
	} else {
		s = nil
	}
	m.mu.Unlock()
	if s != nil {
		s.stop()
	}
}

// StopAll arrête toutes les sessions (arrêt du serveur).
func (m *Manager) StopAll() {
	m.mu.Lock()
	all := m.sessions
	m.sessions = map[string]*Session{}
	m.mu.Unlock()
	for _, s := range all {
		s.stop()
	}
}

func (m *Manager) remove(id string) {
	m.mu.Lock()
	s := m.sessions[id]
	delete(m.sessions, id)
	m.mu.Unlock()
	if s != nil {
		s.stop()
	}
}

func (s *Session) stop() {
	s.cancel() // CommandContext tue ffmpeg ; un lancement pas encore fait échouera immédiatement
	select {
	case <-s.started:
	case <-time.After(10 * time.Second):
	}
	s.mu.Lock()
	cmd, paused := s.cmd, s.paused
	s.paused = false
	s.mu.Unlock()
	if cmd != nil && cmd.Process != nil && paused {
		cmd.Process.Signal(syscall.SIGCONT) // un processus suspendu doit reprendre pour mourir proprement
	}
	select {
	case <-s.done:
	case <-time.After(10 * time.Second):
	}
	os.RemoveAll(s.Dir) // dossier temporaire créé par cette session, dans le dossier de travail d'AURORA
}

// Purged indique si le segment n a déjà été effacé (lu il y a longtemps).
func (s *Session) Purged(n int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return n >= 0 && n < s.purged
}

var segRe = regexp.MustCompile(`^seg(\d{5})\.m4s$`)

// FilePath valide un nom de fichier de session (playlist, init, segment) et renvoie son chemin.
// Le nom est contraint par motif : aucune traversée de chemin possible.
func (s *Session) FilePath(name string) (string, int, error) {
	if name == "index.m3u8" || name == "init.mp4" {
		return filepath.Join(s.Dir, name), -1, nil
	}
	m := segRe.FindStringSubmatch(name)
	if m == nil {
		return "", -1, errors.New("nom invalide")
	}
	n, _ := strconv.Atoi(m[1])
	return filepath.Join(s.Dir, name), n, nil
}

// Touch note l'activité du client (segment demandé) pour la régulation et le nettoyage.
func (s *Session) Touch(seg int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastHit = time.Now()
	if seg > s.lastSeg {
		s.lastSeg = seg
	}
}

// Err renvoie l'erreur ffmpeg éventuelle.
func (s *Session) Err() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

func (s *Session) ctxAlive() bool {
	select {
	case <-s.started:
		return true
	default:
		return false
	}
}

// Finished indique si ffmpeg a terminé.
func (s *Session) Finished() bool {
	select {
	case <-s.done:
		return true
	default:
		return false
	}
}

// WaitFile attend qu'un fichier soit produit par ffmpeg (playlist ou segment).
func (s *Session) WaitFile(ctx context.Context, path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if st, err := os.Stat(path); err == nil && st.Size() > 0 {
			if filepath.Base(path) != "index.m3u8" || playlistHasSegment(path) {
				return nil
			}
		}
		if s.Finished() {
			if st, err := os.Stat(path); err == nil && st.Size() > 0 {
				return nil
			}
			if e := s.Err(); e != "" {
				return fmt.Errorf("échec de ffmpeg : %s", e)
			}
			return errors.New("segment indisponible")
		}
		if time.Now().After(deadline) {
			return errors.New("délai dépassé : le NAS ne produit pas la vidéo assez vite")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(150 * time.Millisecond):
		}
	}
}

func playlistHasSegment(p string) bool {
	b, err := os.ReadFile(p)
	return err == nil && strings.Contains(string(b), ".m4s")
}

// Régulation : ffmpeg est suspendu (SIGSTOP) quand il a trop d'avance sur le lecteur, et repris (SIGCONT)
// quand le lecteur se rapproche. Les segments déjà lus loin derrière sont effacés. Ainsi un remux de 50 Go
// ne remplit pas le disque et une conversion ne consomme du CPU que pour ce qui sera regardé.
const (
	aheadPause  = 45 // segments d'avance (≈ 3 min à 4 s) au-delà desquels ffmpeg est suspendu
	aheadResume = 20
	keepBehind  = 40 // segments conservés derrière la position de lecture
	idleTimeout = 90 * time.Second
)

func (m *Manager) janitor() {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for range t.C {
		m.mu.Lock()
		var all []*Session
		for _, s := range m.sessions {
			all = append(all, s)
		}
		m.mu.Unlock()
		for _, s := range all {
			s.mu.Lock()
			idle := time.Since(s.lastHit)
			last := s.lastSeg
			s.mu.Unlock()
			if idle > idleTimeout {
				slog.Info("session inactive arrêtée", "session", s.ID)
				go m.remove(s.ID) // l'arrêt peut attendre ffmpeg : ne pas bloquer la régulation des autres sessions
				continue
			}
			produced := countSegments(s.Dir)
			s.mu.Lock()
			if s.cmd != nil && s.cmd.Process != nil && !s.Finished() && s.ctxAlive() {
				ahead := produced - 1 - last
				if !s.paused && ahead > aheadPause {
					s.cmd.Process.Signal(syscall.SIGSTOP)
					s.paused = true
				} else if s.paused && ahead < aheadResume {
					s.cmd.Process.Signal(syscall.SIGCONT)
					s.paused = false
				}
			}
			s.mu.Unlock()
			if floor := last - keepBehind; floor > 0 {
				s.mu.Lock()
				from := s.purged
				if floor > s.purged {
					s.purged = floor
				}
				s.mu.Unlock()
				for n := from; n < floor; n++ {
					os.Remove(filepath.Join(s.Dir, fmt.Sprintf("seg%05d.m4s", n)))
				}
			}
		}
	}
}

func countSegments(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	maxN := -1
	for _, e := range entries {
		if m := segRe.FindStringSubmatch(e.Name()); m != nil {
			if n, _ := strconv.Atoi(m[1]); n > maxN {
				maxN = n
			}
		}
	}
	return maxN + 1
}
