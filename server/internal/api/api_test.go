package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"aurora/internal/auth"
	"aurora/internal/catalog"
	"aurora/internal/config"
	"aurora/internal/db"
	"aurora/internal/events"
	"aurora/internal/metadata"
	"aurora/internal/playback"
	"aurora/internal/scanner"
	"aurora/internal/tools"
)

// Test d'intégration : vraie base SQLite, vrais fichiers vidéo générés par ffmpeg, vrai serveur HTTP.
// Ignoré si ffmpeg/ffprobe sont absents de la machine de test.

type env struct {
	t     *testing.T
	srv   *httptest.Server
	s     *Server
	media string
}

func genVideo(t *testing.T, path string, args ...string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(path), 0o755)
	base := []string{"-hide_banner", "-loglevel", "error", "-y", "-f", "lavfi", "-i", "testsrc2=size=320x240:rate=24:duration=20",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=48000:duration=20", "-c:v", "libx264", "-preset", "ultrafast",
		"-pix_fmt", "yuv420p", "-g", "48"}
	cmd := exec.Command("ffmpeg", append(append(base, args...), path)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("ffmpeg : %v %s", err, out)
	}
}

func newEnv(t *testing.T) *env {
	t.Helper()
	tl := tools.Locate("", "")
	if tl.FFmpeg == "" || tl.FFprobe == "" {
		t.Skip("ffmpeg/ffprobe absents : test d'intégration ignoré")
	}
	root := t.TempDir()
	media := filepath.Join(root, "media")
	genVideo(t, filepath.Join(media, "Films", "Amélie (2001)", "Amelie.2001.1080p.mp4"), "-c:a", "aac", "-movflags", "+faststart")
	genVideo(t, filepath.Join(media, "Films", "Le.Parrain.1972.720p.mkv"), "-c:a", "ac3")
	genVideo(t, filepath.Join(media, "Series", "Ma Serie", "Saison 1", "Ma.Serie.S01E01.mkv"), "-c:a", "aac")
	genVideo(t, filepath.Join(media, "Series", "Ma Serie", "Saison 1", "Ma.Serie.S01E02.mkv"), "-c:a", "aac")

	ctx := context.Background()
	cfg := &config.Config{DataDir: filepath.Join(root, "data"), TmpDir: filepath.Join(root, "tmp"), BrowseRoots: []string{media}}
	d, err := db.Open(ctx, filepath.Join(cfg.DataDir, "aurora.db"), filepath.Join(cfg.DataDir, "backups"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	play, err := playback.NewManager(tl.FFmpeg, cfg.TmpDir, playback.Limits{MaxTranscodes: 1, MaxRemux: 2})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(play.StopAll)
	hub := events.NewHub()
	enr := &metadata.Enricher{DB: d.DB, Client: func(context.Context) *metadata.Client { return nil }}
	sc := &scanner.Scanner{DB: d.DB, FFprobe: tl.FFprobe, Hub: hub}
	sc.SetProbeJobs(2)
	s := &Server{Cfg: cfg, DB: d, Store: &catalog.Store{DB: d.DB}, Sessions: &auth.Sessions{DB: d.DB}, Limiter: auth.NewLimiter(),
		Scanner: sc, Enricher: enr, Play: play, Extract: playback.NewExtractor(tl.FFmpeg, filepath.Join(cfg.DataDir, "cache")),
		Tools: tl, Hub: hub, ImageDir: filepath.Join(cfg.DataDir, "cache", "images"), StartedAt: time.Now()}
	srv := httptest.NewServer(s.Handler())
	t.Cleanup(srv.Close)
	return &env{t: t, srv: srv, s: s, media: media}
}

type client struct {
	e *env
	c *http.Client
}

func (e *env) client() *client {
	jar, _ := cookiejar.New(nil)
	return &client{e, &http.Client{Jar: jar}}
}

func (c *client) do(method, path string, body any, out any) int {
	c.e.t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, c.e.srv.URL+path, rd)
	req.Header.Set("X-Aurora", "1")
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.c.Do(req)
	if err != nil {
		c.e.t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if out != nil {
		if err := json.Unmarshal(b, out); err != nil {
			c.e.t.Fatalf("%s %s : réponse illisible (%d) %s", method, path, resp.StatusCode, b)
		}
	}
	return resp.StatusCode
}

func (e *env) waitScan() {
	e.t.Helper()
	deadline := time.Now().Add(60 * time.Second)
	time.Sleep(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		var running int
		e.s.DB.QueryRow(`SELECT COUNT(*) FROM tasks WHERE status='running'`).Scan(&running)
		if e.s.Scanner.Current() == nil && len(e.s.Scanner.Queued()) == 0 && running == 0 {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	e.t.Fatal("analyse trop longue")
}

func TestEndToEnd(t *testing.T) {
	e := newEnv(t)
	admin := e.client()

	// 1. Assistant : création du premier administrateur, puis refus d'un second.
	if code := admin.do("POST", "/api/v1/setup/admin", map[string]string{"username": "admin", "password": "motdepasse1"}, nil); code != 201 {
		t.Fatalf("création admin : %d", code)
	}
	if code := e.client().do("POST", "/api/v1/setup/admin", map[string]string{"username": "pirate", "password": "motdepasse1"}, nil); code != 409 {
		t.Fatalf("second admin : %d, attendu 409", code)
	}

	// 2. CSRF : requête modifiante sans en-tête refusée.
	req, _ := http.NewRequest("POST", e.srv.URL+"/api/v1/admin/scan", strings.NewReader("{}"))
	if resp, _ := admin.c.Do(req); resp.StatusCode != 403 {
		t.Fatalf("CSRF : %d", resp.StatusCode)
	}

	// 3. Bibliothèques : chemin hors racine refusé, chemins valides acceptés.
	if code := admin.do("POST", "/api/v1/admin/libraries", map[string]string{"kind": "movies", "path": "/etc"}, nil); code != 400 {
		t.Fatalf("chemin hors racine : %d", code)
	}
	admin.do("POST", "/api/v1/admin/libraries", map[string]string{"name": "Films", "kind": "movies", "path": filepath.Join(e.media, "Films")}, nil)
	admin.do("POST", "/api/v1/admin/libraries", map[string]string{"name": "Séries", "kind": "series", "path": filepath.Join(e.media, "Series")}, nil)
	e.waitScan()

	// 4. Catalogue.
	var movies struct {
		Items []catalog.Card `json:"items"`
		Total int            `json:"total"`
	}
	admin.do("GET", "/api/v1/movies?sort=title", nil, &movies)
	if movies.Total != 2 || movies.Items[0].Title != "Amélie" || movies.Items[1].Title != "Le Parrain" {
		t.Fatalf("films inattendus : %+v", movies)
	}
	amelie := movies.Items[0]
	var series struct {
		Items []catalog.Card `json:"items"`
	}
	admin.do("GET", "/api/v1/series", nil, &series)
	if len(series.Items) != 1 || series.Items[0].Total != 2 {
		t.Fatalf("séries inattendues : %+v", series)
	}

	// 5. Recherche insensible aux accents et tolérante aux fautes.
	var sr catalog.SearchResults
	admin.do("GET", "/api/v1/search?q=amelie", nil, &sr)
	if len(sr.Movies) != 1 {
		t.Fatalf("recherche sans accent : %+v", sr)
	}
	admin.do("GET", "/api/v1/search?q=parain", nil, &sr)
	if len(sr.Movies) != 1 || !sr.Fuzzy {
		t.Fatalf("recherche approchante : %+v", sr)
	}

	// 6. Fiche et lecture directe avec requête partielle (recherche dans la vidéo).
	var detail catalog.Detail
	admin.do("GET", fmt.Sprintf("/api/v1/items/%d", amelie.ID), nil, &detail)
	if len(detail.Files) != 1 || detail.Files[0].VideoCodec != "h264" {
		t.Fatalf("fiche : %+v", detail.Files)
	}
	fileID := detail.Files[0].ID
	caps := playback.Caps{Containers: []string{"mp4"}, Video: []string{"h264"}, Audio: []string{"aac"}, HLS: "mse"}
	var play map[string]any
	admin.do("POST", "/api/v1/play", map[string]any{"fileId": fileID, "caps": caps}, &play)
	if play["mode"] != "direct" {
		t.Fatalf("lecture : %+v", play)
	}
	req, _ = http.NewRequest("GET", e.srv.URL+play["url"].(string), nil)
	req.Header.Set("Range", "bytes=0-99")
	resp, err := admin.c.Do(req)
	if err != nil || resp.StatusCode != 206 || resp.ContentLength != 100 {
		t.Fatalf("requête partielle : %v %d %d", err, resp.StatusCode, resp.ContentLength)
	}
	resp.Body.Close()

	// 7. Reprise : la position enregistrée revient dans la fiche et dans « Continuer à regarder ».
	admin.do("POST", fmt.Sprintf("/api/v1/items/%d/progress", amelie.ID), map[string]any{"fileId": fileID, "position": 12.5, "duration": 20}, nil)
	admin.do("GET", fmt.Sprintf("/api/v1/items/%d", amelie.ID), nil, &detail)
	if detail.State.Position != 12.5 || detail.State.Watched {
		t.Fatalf("reprise : %+v", detail.State)
	}
	var home catalog.Home
	admin.do("GET", "/api/v1/home", nil, &home)
	if len(home.Sections) == 0 || home.Sections[0].ID != "continue" || home.Sections[0].Items[0].ID != amelie.ID {
		t.Fatalf("accueil : %+v", home.Sections)
	}
	admin.do("POST", fmt.Sprintf("/api/v1/items/%d/progress", amelie.ID), map[string]any{"fileId": fileID, "position": 19.5, "duration": 20}, nil)
	admin.do("GET", fmt.Sprintf("/api/v1/items/%d", amelie.ID), nil, &detail)
	if !detail.State.Watched || detail.State.Position != 0 {
		t.Fatalf("marquage vu : %+v", detail.State)
	}

	// 8. Remux HLS : playlist et segment réellement produits.
	var parrain catalog.Detail
	admin.do("GET", fmt.Sprintf("/api/v1/items/%d", movies.Items[1].ID), nil, &parrain)
	admin.do("POST", "/api/v1/play", map[string]any{"fileId": parrain.Files[0].ID, "caps": caps, "start": 5}, &play)
	if play["mode"] != "remux" {
		t.Fatalf("remux attendu : %+v", play)
	}
	if off := play["offset"].(float64); off > 5 || off < 2 {
		t.Fatalf("offset calé sur l'image clé attendu (2 s, GOP de 48 images) : %v", off)
	}
	resp, err = admin.c.Get(e.srv.URL + play["url"].(string))
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("playlist : %v %d", err, resp.StatusCode)
	}
	pl, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(pl), "seg00000.m4s") {
		t.Fatalf("playlist sans segment : %s", pl)
	}
	// Régression : la playlist ne doit porter aucun validateur de cache (sinon 304 et playlist périmée).
	if resp.Header.Get("Last-Modified") != "" || resp.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("en-têtes de cache de la playlist : %v", resp.Header)
	}
	segURL := strings.Replace(play["url"].(string), "index.m3u8", "seg00000.m4s", 1)
	resp, _ = admin.c.Get(e.srv.URL + segURL)
	if resp.StatusCode != 200 || resp.Header.Get("Content-Type") != "video/iso.segment" {
		t.Fatalf("segment : %d", resp.StatusCode)
	}
	resp.Body.Close()
	// Traversée de chemin dans les noms de segment.
	bad := strings.Replace(play["url"].(string), "index.m3u8", "..%2F..%2Faurora.db", 1)
	if resp, _ := admin.c.Get(e.srv.URL + bad); resp.StatusCode == 200 {
		t.Fatal("traversée de chemin acceptée")
	}

	// 9. Profil enfant : contenu non classé masqué, administration interdite, fichiers inaccessibles.
	admin.do("POST", "/api/v1/admin/users", map[string]any{"username": "enfant", "password": "motdepasse2", "role": "child", "allLibraries": true}, nil)
	kid := e.client()
	if code := kid.do("POST", "/api/v1/auth/login", map[string]string{"username": "enfant", "password": "motdepasse2"}, nil); code != 200 {
		t.Fatalf("connexion enfant : %d", code)
	}
	kid.do("GET", "/api/v1/movies", nil, &movies)
	if movies.Total != 0 {
		t.Fatalf("un profil enfant ne doit pas voir les films non classés : %d", movies.Total)
	}
	if code := kid.do("GET", "/api/v1/admin/dashboard", nil, nil); code != 403 {
		t.Fatalf("admin pour enfant : %d", code)
	}
	if code := kid.do("POST", "/api/v1/play", map[string]any{"fileId": fileID, "caps": caps}, nil); code != 404 {
		t.Fatalf("lecture interdite attendue : %d", code)
	}
	if resp, _ := kid.c.Get(e.srv.URL + fmt.Sprintf("/api/v1/files/%d/stream", fileID)); resp.StatusCode != 404 {
		t.Fatalf("flux direct interdit attendu : %d", resp.StatusCode)
	}

	if resp, _ := kid.c.Get(e.srv.URL + "/api/v1/events"); resp.StatusCode != 403 {
		t.Fatalf("flux d'événements (noms de fichiers) interdit aux non-administrateurs : %d", resp.StatusCode)
	}

	// 9 bis. Contrôle parental hérité : série classée 18 ans, épisodes sans classification propre.
	var seriesID int64
	e.s.DB.QueryRow(`SELECT id FROM items WHERE kind='series'`).Scan(&seriesID)
	e.s.DB.Exec(`UPDATE items SET min_age=18, content_rating='-18' WHERE id=?`, seriesID)
	e.s.DB.Exec(`UPDATE items SET min_age=NULL WHERE parent_id=?`, seriesID)
	var kidID, epID int64
	e.s.DB.QueryRow(`SELECT id FROM users WHERE username='enfant'`).Scan(&kidID)
	e.s.DB.QueryRow(`SELECT id FROM items WHERE parent_id=? LIMIT 1`, seriesID).Scan(&epID)
	e.s.DB.Exec(`UPDATE users SET max_age=12, hide_unrated=0 WHERE id=?`, kidID) // non classés visibles
	var ksr catalog.SearchResults
	kid.do("GET", "/api/v1/search?q=ma+serie", nil, &ksr)
	if len(ksr.Series)+len(ksr.Episodes) != 0 {
		t.Fatalf("épisodes d'une série -18 visibles par un profil 12 ans : %+v", ksr)
	}
	if code := kid.do("GET", fmt.Sprintf("/api/v1/items/%d", epID), nil, nil); code != 404 {
		t.Fatalf("fiche d'épisode -18 accessible : %d", code)
	}

	// 10. Limitation des tentatives de connexion.
	bruteforce := e.client()
	var last int
	for i := 0; i < 6; i++ {
		last = bruteforce.do("POST", "/api/v1/auth/login", map[string]string{"username": "admin", "password": "mauvais!!"}, nil)
	}
	if last != 429 {
		t.Fatalf("limitation attendue après 5 échecs : %d", last)
	}

	// 11. Série : prochain épisode puis épisode suivant.
	var sd catalog.Detail
	admin.do("GET", fmt.Sprintf("/api/v1/items/%d", series.Items[0].ID), nil, &sd)
	if sd.NextUp == nil || *sd.NextUp.Episode != 1 {
		t.Fatalf("prochain épisode : %+v", sd.NextUp)
	}
	admin.do("POST", fmt.Sprintf("/api/v1/items/%d/state", sd.NextUp.ID), map[string]any{"watched": true}, nil)
	admin.do("GET", fmt.Sprintf("/api/v1/items/%d", series.Items[0].ID), nil, &sd)
	if sd.NextUp == nil || *sd.NextUp.Episode != 2 {
		t.Fatalf("épisode suivant : %+v", sd.NextUp)
	}

	// 11 bis. Un sous-titre externe qui est un lien symbolique (vers la base, par exemple) est ignoré.
	parrainPath := filepath.Join(e.media, "Films", "Le.Parrain.1972.720p.fr.vtt")
	if err := os.Symlink(e.s.DB.Path, parrainPath); err != nil {
		t.Fatal(err)
	}

	// 12. Rescan : un fichier déplacé garde son historique, un fichier retiré est masqué (jamais effacé du disque).
	oldPath := filepath.Join(e.media, "Films", "Amélie (2001)", "Amelie.2001.1080p.mp4")
	newPath := filepath.Join(e.media, "Films", "Amélie (2001)", "Rangement", "Amelie.2001.1080p.mp4")
	os.MkdirAll(filepath.Dir(newPath), 0o755)
	if err := os.Rename(oldPath, newPath); err != nil {
		t.Fatal(err)
	}
	admin.do("POST", "/api/v1/admin/scan", map[string]any{}, nil)
	e.waitScan()
	admin.do("GET", fmt.Sprintf("/api/v1/items/%d", amelie.ID), nil, &detail)
	if len(detail.Files) != 1 || detail.Files[0].ID != fileID || !detail.State.Watched {
		t.Fatalf("déplacement : historique perdu %+v %+v", detail.Files, detail.State)
	}
	admin.do("GET", fmt.Sprintf("/api/v1/items/%d", parrain.ID), nil, &parrain)
	for _, sb := range parrain.Files[0].Subtitles {
		if sb.Codec == "vtt" {
			t.Fatalf("sous-titre lien symbolique indexé : %+v", sb)
		}
	}

	// 13. Sauvegarde.
	var b db.BackupInfo
	if code := admin.do("POST", "/api/v1/admin/backups", nil, &b); code != 201 || b.Size == 0 {
		t.Fatalf("sauvegarde : %d %+v", code, b)
	}
}
