package api

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"aurora/internal/auth"
	"aurora/internal/catalog"
	"aurora/internal/playback"
	"aurora/internal/probe"
	"aurora/internal/scanner"
)

type mediaRow struct {
	ID          int64
	ItemID      int64
	Path        string
	Mtime       int64
	Container   string
	Duration    float64
	VideoStream sql.NullInt64
	VideoCodec  string
	Profile     string
	PixFmt      string
	Height      int
	HDR         string
	Audio       []probe.Audio
	Subs        []probe.Subtitle // internes + externes, chemins inclus (usage serveur)
	Chapters    []probe.Chapter
	ProbeState  string
	Present     bool
	LibPath     string
}

// loadFile charge un fichier et vérifie que le profil a le droit de voir l'œuvre associée.
func (s *Server) loadFile(ctx context.Context, u *auth.User, fileID int64) (*mediaRow, error) {
	var m mediaRow
	var itemID sql.NullInt64
	var audio, subs, ext, chapters string
	err := s.DB.QueryRowContext(ctx, `SELECT f.id, f.item_id, f.path, f.mtime, f.container, COALESCE(f.duration,0), f.video_stream,
		f.video_codec, f.video_profile, f.pix_fmt, COALESCE(f.height,0), f.hdr, f.audio, f.subtitles, f.external_subs, f.chapters,
		f.probe_state, f.present, l.path
		FROM media_files f JOIN libraries l ON l.id=f.library_id WHERE f.id=?`, fileID).Scan(&m.ID, &itemID, &m.Path, &m.Mtime,
		&m.Container, &m.Duration, &m.VideoStream, &m.VideoCodec, &m.Profile, &m.PixFmt, &m.Height, &m.HDR, &audio, &subs, &ext,
		&chapters, &m.ProbeState, &m.Present, &m.LibPath)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, catalog.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if !itemID.Valid {
		if !u.IsAdmin() { // fichier non identifié : visible des seuls administrateurs
			return nil, catalog.ErrNotFound
		}
	} else if !s.Store.ItemVisible(ctx, viewer(u), itemID.Int64) {
		return nil, catalog.ErrNotFound
	}
	m.ItemID = itemID.Int64
	json.Unmarshal([]byte(audio), &m.Audio)
	m.Subs = catalog.AllSubtitles(subs, ext)
	json.Unmarshal([]byte(chapters), &m.Chapters)
	return &m, nil
}

type playRequest struct {
	FileID        int64         `json:"fileId"`
	Caps          playback.Caps `json:"caps"`
	AudioIndex    *int          `json:"audioIndex"`
	SubtitleIndex *int          `json:"subtitleIndex"`
	Start         float64       `json:"start"`
	ForceMode     string        `json:"forceMode"` // diagnostic : "remux" ou "transcode"
	External      bool          `json:"external"`  // demande un lien signé pour un lecteur externe
}

type subtitleOut struct {
	Index    int    `json:"index"`
	Label    string `json:"label"`
	Language string `json:"language"`
	Codec    string `json:"codec"`
	Forced   bool   `json:"forced"`
	Default  bool   `json:"default"`
	Text     bool   `json:"text"`
	External bool   `json:"external"`
	URL      string `json:"url,omitempty"`
}

// handlePlay arbitre le mode de lecture et, si nécessaire, démarre une session ffmpeg.
func (s *Server) handlePlay(w http.ResponseWriter, r *http.Request, u *auth.User) {
	var req playRequest
	if err := readJSON(r, &req); err != nil {
		fail(w, r, err)
		return
	}
	ctx := r.Context()
	m, err := s.loadFile(ctx, u, req.FileID)
	if err != nil {
		fail(w, r, err)
		return
	}
	if !m.Present {
		fail(w, r, &httpError{http.StatusGone, "ce fichier est introuvable sur le NAS (déplacé, supprimé ou disque indisponible)"})
		return
	}
	if _, err := os.Stat(m.Path); err != nil {
		fail(w, r, &httpError{http.StatusGone, "le fichier vidéo n'est pas accessible : " + humanFSError(err)})
		return
	}
	// Fichier jamais analysé (analyse en cours ou ffprobe absent) : analyse immédiate si possible.
	if m.ProbeState != "ok" && s.Tools.FFprobe != "" {
		if res, err := probe.Run(ctx, s.Tools.FFprobe, m.Path); err == nil {
			scanner.SaveProbe(ctx, s.DB, m.ID, res)
			if m, err = s.loadFile(ctx, u, req.FileID); err != nil {
				fail(w, r, err)
				return
			}
		}
	}
	if req.Start < 0 || (m.Duration > 0 && req.Start > m.Duration) {
		req.Start = 0
	}

	f := playback.File{Path: m.Path, Container: m.Container, VideoCodec: m.VideoCodec, VideoProfile: m.Profile,
		PixFmt: m.PixFmt, Height: m.Height, HDR: m.HDR, Audio: m.Audio}
	opt := playback.Options{AudioIndex: -1, BurnSubtitle: false, Allow4KEncode: s.setting(ctx, "allow_4k_transcode", "0") == "1"}
	if req.AudioIndex != nil {
		opt.AudioIndex = *req.AudioIndex
	}
	burnStream := -1
	if req.SubtitleIndex != nil && *req.SubtitleIndex >= 0 && *req.SubtitleIndex < len(m.Subs) {
		if sub := m.Subs[*req.SubtitleIndex]; !sub.Text && sub.Stream >= 0 {
			opt.BurnSubtitle, burnStream = true, sub.Stream
		}
	}
	d := playback.Decide(f, req.Caps, opt)
	switch req.ForceMode {
	case "remux":
		if d.Mode == playback.ModeDirect {
			d.Mode, d.Reason = playback.ModeRemux, "Remux forcé (diagnostic)."
		}
	case "transcode":
		d.Mode, d.CopyAudio, d.Reason = playback.ModeTranscode, false, "Conversion forcée (diagnostic)."
	}
	if !m.VideoStream.Valid && d.Mode != playback.ModeDirect {
		fail(w, r, &httpError{http.StatusUnprocessableEntity, "aucun flux vidéo exploitable dans ce fichier"})
		return
	}
	if d.Mode == playback.ModeTranscode {
		if tooMuch, why := playback.TooExpensive(f, opt); tooMuch {
			fail(w, r, &httpError{http.StatusUnprocessableEntity, why})
			return
		}
	}

	resp := map[string]any{
		"mode": d.Mode, "reason": d.Reason, "duration": m.Duration, "chapters": m.Chapters, "itemId": m.ItemID,
		"fileId": m.ID, "audio": m.Audio, "audioStream": d.AudioStream,
	}
	if req.External { // repli lecteur externe (VLC sur la box) : lien seulement sur demande
		resp["signedUrl"] = s.signedURL(ctx, m.ID, u.ID)
	}
	offset := 0.0
	if d.Mode == playback.ModeDirect {
		resp["url"] = fmt.Sprintf("/api/v1/files/%d/stream", m.ID)
		resp["start"] = req.Start
	} else {
		if d.Mode == playback.ModeRemux && req.Start > 0 {
			req.Start = playback.KeyframeBefore(ctx, s.Tools.FFprobe, m.Path, req.Start)
		}
		sess, err := s.Play.Start(playback.StartRequest{UserID: u.ID, FileID: m.ID, Path: m.Path, Decision: d, File: f,
			VideoStream: int(m.VideoStream.Int64), Start: req.Start, BurnSubIdx: burnStream})
		if err != nil {
			if errors.Is(err, playback.ErrLimit) {
				fail(w, r, &httpError{http.StatusServiceUnavailable, err.Error()})
				return
			}
			fail(w, r, &httpError{http.StatusInternalServerError, err.Error()})
			return
		}
		offset = req.Start
		resp["url"] = fmt.Sprintf("/api/v1/hls/%s/index.m3u8", sess.ID)
		resp["sessionId"] = sess.ID
		resp["start"] = 0
	}
	resp["offset"] = offset
	subs := make([]subtitleOut, 0, len(m.Subs))
	for _, sb := range m.Subs {
		o := subtitleOut{Index: sb.Index, Language: sb.Language, Codec: sb.Codec, Forced: sb.Forced, Default: sb.Default,
			Text: sb.Text, External: sb.External != "", Label: subtitleLabel(sb)}
		if sb.Text {
			o.URL = fmt.Sprintf("/api/v1/files/%d/subtitles/%d?offset=%s", m.ID, sb.Index, strconv.FormatFloat(offset, 'f', 3, 64))
		}
		subs = append(subs, o)
	}
	resp["subtitles"] = subs
	resp["burnedSubtitle"] = burnStream >= 0
	writeJSON(w, http.StatusOK, resp)
}

var langNames = map[string]string{
	"fre": "Français", "fra": "Français", "fr": "Français", "eng": "Anglais", "en": "Anglais", "ger": "Allemand",
	"deu": "Allemand", "de": "Allemand", "spa": "Espagnol", "es": "Espagnol", "ita": "Italien", "it": "Italien",
	"jpn": "Japonais", "ja": "Japonais", "por": "Portugais", "pt": "Portugais", "dut": "Néerlandais", "nld": "Néerlandais",
	"nl": "Néerlandais", "kor": "Coréen", "ko": "Coréen", "chi": "Chinois", "zho": "Chinois", "zh": "Chinois",
	"rus": "Russe", "ru": "Russe", "ara": "Arabe", "ar": "Arabe",
}

func subtitleLabel(sb probe.Subtitle) string {
	l := langNames[sb.Language]
	if l == "" {
		l = strings.ToUpper(sb.Language)
	}
	if l == "" {
		l = "Inconnu"
	}
	var extra []string
	if sb.Title != "" && !strings.EqualFold(sb.Title, l) {
		extra = append(extra, sb.Title)
	}
	if sb.Forced {
		extra = append(extra, "forcés")
	}
	if !sb.Text {
		extra = append(extra, "image")
	}
	if sb.External != "" {
		extra = append(extra, "externe")
	}
	if len(extra) > 0 {
		l += " (" + strings.Join(extra, ", ") + ")"
	}
	return l
}

// subtitleCacheKey identifie la source réelle du sous-titre, jamais sa position dans la liste (qui change
// quand un fichier .srt est ajouté ou retiré : le cache aurait alors servi une autre piste).
func subtitleCacheKey(m *mediaRow, sb probe.Subtitle) string {
	src := fmt.Sprintf("v2|%d|%d|%s|stream:%d", m.ID, m.Mtime, m.Path, sb.Stream)
	if sb.External != "" {
		src = "v2|ext|" + sb.External
		if st, err := os.Stat(sb.External); err == nil {
			src += fmt.Sprintf("|%d|%d", st.Size(), st.ModTime().UnixNano())
		}
	}
	h := sha256.Sum256([]byte(src))
	return hex.EncodeToString(h[:16])
}

// safeSidecar : un sous-titre externe n'est servi que s'il est un fichier régulier (pas un lien
// symbolique) situé dans la bibliothèque. Défense en profondeur, en plus du filtrage à l'indexation.
func safeSidecar(p, libRoot string) bool {
	st, err := os.Lstat(p)
	if err != nil || !st.Mode().IsRegular() {
		return false
	}
	rel, err := filepath.Rel(libRoot, p)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func humanFSError(err error) string {
	switch {
	case errors.Is(err, os.ErrNotExist):
		return "fichier absent"
	case errors.Is(err, os.ErrPermission):
		return "permission refusée à l'utilisateur système d'AURORA"
	}
	return "erreur d'accès"
}

var contentTypes = map[string]string{
	".mp4": "video/mp4", ".m4v": "video/mp4", ".mov": "video/quicktime", ".webm": "video/webm",
	".mkv": "video/x-matroska", ".ts": "video/mp2t", ".m2ts": "video/mp2t", ".avi": "video/x-msvideo",
}

// handleDirectStream : Direct Play, avec prise en charge des requêtes partielles (Range) pour la recherche.
func (s *Server) handleDirectStream(w http.ResponseWriter, r *http.Request, u *auth.User) {
	id, err := pathID(r, "id")
	if err != nil {
		fail(w, r, err)
		return
	}
	s.serveFile(w, r, u, id)
}

// signedTTL : la durée d'un long film ; au-delà, le lecteur externe redemande un lien.
const signedTTL = 4 * time.Hour

func (s *Server) streamSecret() []byte {
	s.streamKeyOnce.Do(func() {
		s.streamKey = make([]byte, 32)
		rand.Read(s.streamKey)
	})
	return s.streamKey
}

// streamSig signe « fichier|profil|expiration|empreinte du mot de passe » : changer le mot de passe
// (ou sa réinitialisation par un administrateur) révoque immédiatement tous les liens du profil.
func (s *Server) streamSig(fileID, userID, exp int64, passwordHash string) string {
	mac := hmac.New(sha256.New, s.streamSecret())
	fmt.Fprintf(mac, "%d|%d|%d|%s", fileID, userID, exp, passwordHash)
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *Server) passwordHash(ctx context.Context, userID int64) string {
	var h string
	s.DB.QueryRowContext(ctx, `SELECT password_hash FROM users WHERE id=?`, userID).Scan(&h)
	return h
}

// signedURL : lien de lecture directe valable signedTTL, pour un profil et un fichier.
func (s *Server) signedURL(ctx context.Context, fileID, userID int64) string {
	exp := time.Now().Add(signedTTL).Unix()
	return fmt.Sprintf("/api/v1/files/%d/signed?u=%d&e=%d&s=%s", fileID, userID, exp,
		s.streamSig(fileID, userID, exp, s.passwordHash(ctx, userID)))
}

// handleSignedStream sert un fichier sur présentation d'un lien signé valide. Les droits du profil sont
// revérifiés à chaque requête (un profil restreint entre-temps perd l'accès).
func (s *Server) handleSignedStream(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		fail(w, r, err)
		return
	}
	q := r.URL.Query()
	uid, _ := strconv.ParseInt(q.Get("u"), 10, 64)
	exp, _ := strconv.ParseInt(q.Get("e"), 10, 64)
	ph := s.passwordHash(r.Context(), uid)
	want := s.streamSig(id, uid, exp, ph)
	if uid <= 0 || ph == "" || exp < time.Now().Unix() || !hmac.Equal([]byte(want), []byte(q.Get("s"))) {
		writeError(w, http.StatusForbidden, "lien de lecture invalide ou expiré")
		return
	}
	u, err := s.userByID(r.Context(), uid)
	if err != nil {
		writeError(w, http.StatusForbidden, "lien de lecture invalide ou expiré")
		return
	}
	s.serveFile(w, r, u, id)
}

func (s *Server) userByID(ctx context.Context, id int64) (*auth.User, error) {
	var u auth.User
	var maxAge sql.NullInt64
	err := s.DB.QueryRowContext(ctx, `SELECT id, username, role, max_age, hide_unrated, all_libraries, prefs FROM users WHERE id=?`, id).
		Scan(&u.ID, &u.Username, &u.Role, &maxAge, &u.HideUnrated, &u.AllLibraries, &u.Prefs)
	if err != nil {
		return nil, err
	}
	if maxAge.Valid {
		v := int(maxAge.Int64)
		u.MaxAge = &v
	}
	return &u, nil
}

func (s *Server) serveFile(w http.ResponseWriter, r *http.Request, u *auth.User, id int64) {
	m, err := s.loadFile(r.Context(), u, id)
	if err != nil {
		fail(w, r, err)
		return
	}
	f, err := os.Open(m.Path)
	if err != nil {
		fail(w, r, &httpError{http.StatusGone, "le fichier vidéo n'est pas accessible : " + humanFSError(err)})
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		fail(w, r, err)
		return
	}
	ct := contentTypes[strings.ToLower(filepath.Ext(m.Path))]
	if ct == "" {
		ct = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "private, no-transform")
	http.ServeContent(w, r, "", st.ModTime(), f)
}

func (s *Server) handleHLS(w http.ResponseWriter, r *http.Request, u *auth.User) {
	sess := s.Play.Get(r.PathValue("session"), u.ID)
	if sess == nil {
		writeError(w, http.StatusNotFound, "session de lecture expirée")
		return
	}
	name := r.PathValue("name")
	p, seg, err := sess.FilePath(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if sess.Purged(seg) {
		writeError(w, http.StatusGone, "segment déjà lu et effacé : relancez la lecture à cette position")
		return
	}
	sess.Touch(seg)
	timeout := 25 * time.Second
	if name == "index.m3u8" {
		timeout = 45 * time.Second // démarrage de ffmpeg + premier segment (plus long en conversion)
	}
	if err := sess.WaitFile(r.Context(), p, timeout); err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	switch {
	case name == "index.m3u8":
		// Pas de http.ServeFile ici : sa date Last-Modified est à la seconde, or ffmpeg réécrit la playlist
		// plusieurs fois par seconde. Le navigateur recevait alors un 304 et gardait une playlist périmée
		// (lecture bloquée après les premiers segments — constaté en test).
		b, err := os.ReadFile(p)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(b)
		return
	case name == "init.mp4":
		w.Header().Set("Content-Type", "video/mp4")
	default:
		w.Header().Set("Content-Type", "video/iso.segment")
		w.Header().Set("Cache-Control", "private, max-age=3600")
	}
	http.ServeFile(w, r, p)
}

func (s *Server) handleStopSession(w http.ResponseWriter, r *http.Request, u *auth.User) {
	s.Play.Stop(r.PathValue("session"), u.ID)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleSubtitle convertit une piste textuelle en WebVTT (mise en cache), décalée si la lecture HLS
// a démarré en cours de film.
func (s *Server) handleSubtitle(w http.ResponseWriter, r *http.Request, u *auth.User) {
	id, err := pathID(r, "id")
	if err != nil {
		fail(w, r, err)
		return
	}
	idx, err := strconv.Atoi(strings.TrimSuffix(r.PathValue("index"), ".vtt"))
	if err != nil || idx < 0 {
		fail(w, r, badRequest("piste invalide"))
		return
	}
	m, err := s.loadFile(r.Context(), u, id)
	if err != nil {
		fail(w, r, err)
		return
	}
	if idx >= len(m.Subs) {
		fail(w, r, catalog.ErrNotFound)
		return
	}
	sb := m.Subs[idx]
	if sb.External != "" && !safeSidecar(sb.External, m.LibPath) {
		fail(w, r, catalog.ErrNotFound)
		return
	}
	if !sb.Text {
		fail(w, r, &httpError{http.StatusUnsupportedMediaType, "sous-titre image : affichage possible uniquement par incrustation"})
		return
	}
	vtt, err := s.Extract.SubtitleVTT(r.Context(), subtitleCacheKey(m, sb), m.Path, sb.Stream, sb.External)
	if err != nil {
		fail(w, r, &httpError{http.StatusUnprocessableEntity, err.Error()})
		return
	}
	offset, _ := strconv.ParseFloat(r.URL.Query().Get("offset"), 64)
	w.Header().Set("Content-Type", "text/vtt; charset=utf-8")
	w.Header().Set("Cache-Control", "private, max-age=300")
	w.Write(playback.ShiftVTT(vtt, offset))
}
