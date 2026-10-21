// AURORA — médiathèque personnelle native pour Synology DSM 7.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"aurora/internal/api"
	"aurora/internal/auth"
	"aurora/internal/catalog"
	"aurora/internal/config"
	"aurora/internal/db"
	"aurora/internal/events"
	"aurora/internal/metadata"
	"aurora/internal/playback"
	"aurora/internal/scanner"
	"aurora/internal/tools"
	"aurora/internal/watcher"
	"aurora/internal/webui"
)

func main() {
	if err := run(); err != nil {
		slog.Error("arrêt sur erreur", "err", err)
		fmt.Fprintln(os.Stderr, "AURORA :", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		return err
	}
	var logw io.Writer = os.Stderr
	var logFile *config.RotatingFile
	if cfg.LogFile != "" {
		if logFile, err = config.OpenRotating(cfg.LogFile); err != nil {
			return err
		}
		logw = logFile
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(logw, &slog.HandlerOptions{Level: slog.LevelInfo})))
	slog.Info("démarrage d'AURORA", "version", config.Version, "data", cfg.DataDir, "listen", cfg.Listen)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	for _, d := range []string{cfg.DataDir, cfg.TmpDir} {
		if err := os.MkdirAll(d, 0o750); err != nil {
			return err
		}
	}
	database, err := db.Open(ctx, filepath.Join(cfg.DataDir, "aurora.db"), filepath.Join(cfg.DataDir, "backups"))
	if err != nil {
		return fmt.Errorf("base de données : %w", err)
	}
	defer database.Close()

	tl := tools.Locate(cfg.FFmpeg, cfg.FFprobe)
	if tl.FFprobe == "" {
		slog.Warn("ffprobe introuvable : l'analyse technique des fichiers sera reportée")
	}
	if tl.FFmpeg == "" {
		slog.Warn("ffmpeg introuvable : seule la lecture directe sera possible")
	}
	slog.Info("outils", "ffmpeg", tl.FFmpeg, "ffprobe", tl.FFprobe, "version", tl.FFmpegVersion)

	setting := func(key, def string) string {
		var v string
		if database.QueryRowContext(ctx, `SELECT value FROM settings WHERE key=?`, key).Scan(&v) != nil {
			return def
		}
		return v
	}
	atoi := func(s string, def int) int {
		if v, err := strconv.Atoi(s); err == nil {
			return v
		}
		return def
	}

	hub := events.NewHub()
	imageDir := filepath.Join(cfg.DataDir, "cache", "images")
	play, err := playback.NewManager(tl.FFmpeg, cfg.TmpDir, playback.Limits{
		MaxTranscodes: atoi(setting("max_transcodes", "1"), 1), MaxRemux: 4,
	})
	if err != nil {
		return err
	}
	play.HasTonemap = hasFilters(tl.FFmpeg, "zscale", "tonemap")
	defer play.StopAll()

	enricher := &metadata.Enricher{DB: database.DB, Client: func(ctx context.Context) *metadata.Client {
		key := setting("tmdb_api_key", "")
		if key == "" {
			return nil
		}
		return metadata.NewClient(key, setting("language", "fr-FR"), imageDir)
	}}
	sc := &scanner.Scanner{DB: database.DB, FFprobe: tl.FFprobe, Hub: hub}
	sc.SetProbeJobs(atoi(setting("probe_jobs", "2"), 2))
	sc.AfterScan = func(ctx context.Context, lib scanner.Library, report func(scanner.Progress)) error {
		return enricher.EnrichLibrary(ctx, lib.ID, lib.Kind, func(p metadata.Progress) {
			report(scanner.Progress{Done: p.Done, Total: p.Total, Current: p.Current})
		})
	}
	if err := sc.RecoverInterrupted(ctx); err != nil {
		return err
	}

	srv := &api.Server{
		Cfg: cfg, DB: database, Store: &catalog.Store{DB: database.DB}, Sessions: &auth.Sessions{DB: database.DB},
		Limiter: auth.NewLimiter(), Scanner: sc, Enricher: enricher, Play: play,
		Extract: playback.NewExtractor(tl.FFmpeg, filepath.Join(cfg.DataDir, "cache")), Tools: tl, Hub: hub,
		Log: logFile, ImageDir: imageDir, StartedAt: time.Now(),
	}
	// Détection immédiate des ajouts (inotify) ; sans elle, l'analyse périodique couvre tout.
	wt := &watcher.Watcher{DB: database.DB, Enqueue: sc.Enqueue, Enabled: srv.WatchEnabled}
	if err := wt.Start(ctx); err != nil {
		slog.Warn("surveillance des dossiers indisponible : analyse périodique seulement", "err", err)
	} else {
		srv.Watcher = wt
		go wt.Sync(ctx)
	}
	if cfg.Dev {
		srv.Web = os.DirFS(cfg.WebDir)
	} else {
		srv.Web = webui.FS()
	}

	httpSrv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
		// Pas de WriteTimeout : la lecture vidéo et le flux SSE sont de longues réponses.
	}
	errc := make(chan error, 1)
	go func() {
		slog.Info("serveur HTTP prêt", "addr", cfg.Listen)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errc <- err
		}
	}()
	go maintenance(ctx, srv, sc, setting, atoi)

	select {
	case err := <-errc:
		return fmt.Errorf("serveur HTTP : %w", err)
	case <-ctx.Done():
	}
	slog.Info("arrêt demandé")
	sc.Cancel()
	play.StopAll()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	httpSrv.Shutdown(shutdownCtx)
	slog.Info("arrêt terminé")
	return nil
}

// maintenance : purge des sessions expirées et analyses périodiques (incrémentales, jamais au démarrage).
func maintenance(ctx context.Context, srv *api.Server, sc *scanner.Scanner, setting func(string, string) string, atoi func(string, int) int) {
	t := time.NewTicker(10 * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		srv.Sessions.Purge(ctx)
		// Disque rebranché, dossier recréé : la surveillance est réalignée toutes les 10 min.
		if srv.Watcher != nil {
			srv.Watcher.Sync(ctx)
		}
		hours := atoi(setting("scan_interval_hours", "6"), 6)
		if hours <= 0 {
			continue
		}
		rows, err := srv.DB.QueryContext(ctx, `SELECT id FROM libraries WHERE COALESCE(last_scan_at,0) < ?`,
			time.Now().Add(-time.Duration(hours)*time.Hour).Unix())
		if err != nil {
			continue
		}
		var ids []int64
		cur := sc.Current()
		for rows.Next() {
			var id int64
			// En cours d'analyse : last_scan_at n'est mis à jour qu'à la fin ; la remettre en file la ferait
			// réanalyser entièrement juste après (Enqueue reprogramme désormais une bibliothèque en cours).
			if rows.Scan(&id) == nil && (cur == nil || cur.LibraryID != id) {
				ids = append(ids, id)
			}
		}
		rows.Close()
		if len(ids) > 0 {
			sc.Enqueue(ids...)
		}
	}
}

// hasFilters vérifie la présence de filtres ffmpeg (ex. zscale/tonemap pour la conversion HDR -> SDR).
func hasFilters(ffmpeg string, names ...string) bool {
	if ffmpeg == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, ffmpeg, "-hide_banner", "-filters").Output()
	if err != nil {
		return false
	}
	for _, n := range names {
		if !strings.Contains(string(out), " "+n+" ") {
			return false
		}
	}
	return true
}
