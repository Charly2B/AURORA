// Package api : serveur HTTP (API REST /api/v1 et interface web).
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"path"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
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
	"aurora/internal/watcher"
)

// Server regroupe les dépendances des routes.
type Server struct {
	Cfg       *config.Config
	DB        *db.DB
	Store     *catalog.Store
	Sessions  *auth.Sessions
	Limiter   *auth.Limiter
	Scanner   *scanner.Scanner
	Enricher  *metadata.Enricher
	Play      *playback.Manager
	Extract   *playback.Extractor
	Tools     tools.Tools
	Hub       *events.Hub
	Log       *config.RotatingFile
	Web       fs.FS
	ImageDir  string
	StartedAt time.Time
	// TMDBBase / TMDBImages : surchargés par les tests (faux serveur TMDB) ; vides = TMDB réel.
	TMDBBase, TMDBImages string

	streamKeyOnce sync.Once
	streamKey     []byte // clé des liens de lecture signés (régénérée à chaque démarrage)

	links linkStore // associations d'appareils par code en attente (link.go)

	// Watcher : détection immédiate des ajouts (nil si inotify indisponible ; l'analyse périodique reste).
	Watcher *watcher.Watcher
}

const cookieName = "aurora_session"

// Handler construit le routeur complet.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	// Public
	mux.HandleFunc("GET /api/v1/status", s.handleStatus)
	mux.HandleFunc("POST /api/v1/setup/admin", s.handleSetupAdmin)
	mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/logout", s.handleLogout)
	mux.HandleFunc("POST /api/v1/auth/link/start", s.handleLinkStart)
	mux.HandleFunc("POST /api/v1/auth/link/poll", s.handleLinkPoll)

	// Profil connecté
	u := s.requireUser
	mux.Handle("GET /api/v1/auth/me", u(s.handleMe))
	mux.Handle("POST /api/v1/auth/link/approve", u(s.handleLinkApprove))
	mux.Handle("PUT /api/v1/me/prefs", u(s.handlePutPrefs))
	mux.Handle("PUT /api/v1/me/password", u(s.handleChangePassword))
	mux.Handle("GET /api/v1/events", s.requireAdmin(s.handleEvents)) // noms de fichiers : administrateurs seulement
	mux.Handle("GET /api/v1/home", u(s.handleHome))
	mux.Handle("GET /api/v1/libraries", u(s.handleLibraries))
	mux.Handle("GET /api/v1/movies", u(s.handleList("movie")))
	mux.Handle("GET /api/v1/series", u(s.handleList("series")))
	mux.Handle("GET /api/v1/facets/{kind}", u(s.handleFacets))
	mux.Handle("GET /api/v1/items/{id}", u(s.handleItem))
	mux.Handle("GET /api/v1/items/{id}/seasons/{season}", u(s.handleEpisodes))
	mux.Handle("GET /api/v1/items/{id}/next", u(s.handleNextEpisode))
	mux.Handle("GET /api/v1/items/{id}/thumb", u(s.handleThumb))
	mux.Handle("POST /api/v1/items/{id}/state", u(s.handleSetState))
	mux.Handle("POST /api/v1/items/{id}/progress", u(s.handleProgress))
	mux.Handle("POST /api/v1/items/{id}/continue/hide", u(s.handleHideContinue))
	mux.Handle("GET /api/v1/collections", u(s.handleCollections))
	mux.Handle("GET /api/v1/collections/{id}", u(s.handleCollection))
	mux.Handle("GET /api/v1/search", u(s.handleSearch))
	mux.Handle("GET /api/v1/img/{name}", u(s.handleImage))
	mux.Handle("POST /api/v1/play", u(s.handlePlay))
	mux.Handle("GET /api/v1/files/{id}/stream", u(s.handleDirectStream))
	// Lien signé et temporaire : lecteurs externes (VLC sur la box) qui ne transmettent pas la session.
	mux.HandleFunc("GET /api/v1/files/{id}/signed", s.handleSignedStream)
	mux.Handle("GET /api/v1/files/{id}/subtitles/{index}", u(s.handleSubtitle))
	mux.Handle("GET /api/v1/hls/{session}/{name}", u(s.handleHLS))
	mux.Handle("DELETE /api/v1/hls/{session}", u(s.handleStopSession))

	// Administration
	a := s.requireAdmin
	mux.Handle("GET /api/v1/admin/dashboard", a(s.handleDashboard))
	mux.Handle("GET /api/v1/admin/setup", a(s.handleGetSetupStep))
	mux.Handle("PUT /api/v1/admin/setup", a(s.handlePutSetupStep))
	mux.Handle("GET /api/v1/admin/browse", a(s.handleBrowse))
	mux.Handle("GET /api/v1/admin/libraries", a(s.handleAdminLibraries))
	mux.Handle("POST /api/v1/admin/libraries", a(s.handleCreateLibrary))
	mux.Handle("PUT /api/v1/admin/libraries/{id}", a(s.handleUpdateLibrary))
	mux.Handle("DELETE /api/v1/admin/libraries/{id}", a(s.handleDeleteLibrary))
	mux.Handle("POST /api/v1/admin/scan", a(s.handleScan))
	mux.Handle("POST /api/v1/admin/scan/cancel", a(s.handleScanCancel))
	mux.Handle("GET /api/v1/admin/tasks", a(s.handleTasks))
	mux.Handle("GET /api/v1/admin/tasks/{id}/errors", a(s.handleTaskErrors))
	mux.Handle("GET /api/v1/admin/unidentified", a(s.handleUnidentified))
	mux.Handle("POST /api/v1/admin/files/{id}/assign", a(s.handleAssignFile))
	mux.Handle("GET /api/v1/admin/items/{id}/candidates", a(s.handleCandidates))
	mux.Handle("POST /api/v1/admin/items/{id}/match", a(s.handleMatch))
	mux.Handle("PUT /api/v1/admin/items/{id}", a(s.handleEditItem))
	mux.Handle("GET /api/v1/admin/items/{id}/artwork", a(s.handleArtworkList))
	mux.Handle("POST /api/v1/admin/items/{id}/artwork", a(s.handleArtworkSet))
	mux.Handle("GET /api/v1/admin/users", a(s.handleUsers))
	mux.Handle("POST /api/v1/admin/users", a(s.handleCreateUser))
	mux.Handle("PUT /api/v1/admin/users/{id}", a(s.handleUpdateUser))
	mux.Handle("DELETE /api/v1/admin/users/{id}", a(s.handleDeleteUser))
	mux.Handle("GET /api/v1/admin/settings", a(s.handleGetSettings))
	mux.Handle("PUT /api/v1/admin/settings", a(s.handlePutSettings))
	mux.Handle("GET /api/v1/admin/backups", a(s.handleBackups))
	mux.Handle("POST /api/v1/admin/backups", a(s.handleCreateBackup))
	mux.Handle("GET /api/v1/admin/backups/{name}", a(s.handleDownloadBackup))
	mux.Handle("POST /api/v1/admin/backups/{name}/restore", a(s.handleRestoreBackup))
	mux.Handle("GET /api/v1/admin/logs", a(s.handleLogs))
	mux.Handle("GET /api/v1/admin/diagnostics", a(s.handleDiagnostics))

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "route inconnue")
	})
	mux.HandleFunc("/", s.handleWeb)
	return s.recoverer(s.securityHeaders(s.csrf(mux)))
}

// --- Middlewares ---------------------------------------------------------------------------------

func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				if v == http.ErrAbortHandler {
					panic(v)
				}
				slog.Error("panique", "path", r.URL.Path, "err", v, "stack", string(debug.Stack()))
				writeError(w, http.StatusInternalServerError, "erreur interne")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "same-origin")
		h.Set("X-Frame-Options", "SAMEORIGIN")
		h.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data: blob: https://image.tmdb.org; media-src 'self' blob:; "+
			"style-src 'self' 'unsafe-inline'; script-src 'self'; worker-src 'self' blob:; connect-src 'self'; "+
			"font-src 'self' data:; frame-ancestors 'self'; base-uri 'self'; form-action 'self'")
		next.ServeHTTP(w, r)
	})
}

// csrf : toute requête modifiant l'état doit porter l'en-tête X-Aurora. Un site tiers ne peut pas
// l'ajouter sans requête CORS préalable, que ce serveur n'autorise jamais.
func (s *Server) csrf(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
		default:
			if strings.HasPrefix(r.URL.Path, "/api/") && r.Header.Get("X-Aurora") != "1" {
				writeError(w, http.StatusForbidden, "requête refusée (en-tête de sécurité manquant)")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

type userHandler func(w http.ResponseWriter, r *http.Request, u *auth.User)

func (s *Server) currentUser(r *http.Request) (*auth.User, error) {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return nil, auth.ErrNoSession
	}
	return s.Sessions.Lookup(r.Context(), c.Value)
}

func (s *Server) requireUser(h userHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, err := s.currentUser(r)
		if err != nil {
			if !errors.Is(err, auth.ErrNoSession) {
				slog.Error("session", "err", err)
			}
			writeError(w, http.StatusUnauthorized, "connexion requise")
			return
		}
		h(w, r, u)
	})
}

func (s *Server) requireAdmin(h userHandler) http.Handler {
	return s.requireUser(func(w http.ResponseWriter, r *http.Request, u *auth.User) {
		if !u.IsAdmin() {
			writeError(w, http.StatusForbidden, "réservé aux administrateurs")
			return
		}
		h(w, r, u)
	})
}

func viewer(u *auth.User) catalog.Viewer {
	return catalog.Viewer{UserID: u.ID, AllLibraries: u.AllLibraries || u.IsAdmin(), MaxAge: u.MaxAge, HideUnrated: u.HideUnrated}
}

// --- Réponses ------------------------------------------------------------------------------------

type apiError struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, apiError{msg})
}

// httpError : erreur portant son code HTTP et un message destiné à l'utilisateur.
type httpError struct {
	status int
	msg    string
}

func (e *httpError) Error() string { return e.msg }

func badRequest(format string, a ...any) error {
	return &httpError{http.StatusBadRequest, fmt.Sprintf(format, a...)}
}

// fail centralise la conversion erreur -> réponse. Les erreurs internes sont journalisées
// et un message générique est renvoyé (pas de fuite de chemins ou de SQL).
func fail(w http.ResponseWriter, r *http.Request, err error) {
	var he *httpError
	switch {
	case errors.As(err, &he):
		writeError(w, he.status, he.msg)
	case errors.Is(err, catalog.ErrNotFound):
		writeError(w, http.StatusNotFound, "introuvable")
	case errors.Is(err, context.Canceled):
		// client parti
	default:
		slog.Error("erreur", "method", r.Method, "path", r.URL.Path, "err", err)
		writeError(w, http.StatusInternalServerError, "erreur interne (détails dans le journal)")
	}
}

func readJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	if err := dec.Decode(v); err != nil {
		return badRequest("requête JSON invalide")
	}
	return nil
}

func pathID(r *http.Request, name string) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, badRequest("identifiant invalide")
	}
	return id, nil
}

// clientIP : adresse du client. X-Forwarded-For n'est cru que si la requête vient de la machine locale
// (reverse proxy de DSM), sinon un client pourrait contourner la limitation des tentatives.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		// Dernier élément : celui qu'ajoute le proxy de confiance (nginx de DSM, $proxy_add_x_forwarded_for).
		// Les précédents viennent du client et sont falsifiables (contournement des limites par adresse).
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			if last := strings.TrimSpace(parts[len(parts)-1]); last != "" {
				return last
			}
		}
	}
	return host
}

func isHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
	}
	return false
}

// --- Interface web -------------------------------------------------------------------------------

// handleWeb sert l'application monopage ; toute route inconnue renvoie index.html.
func (s *Server) handleWeb(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeError(w, http.StatusMethodNotAllowed, "méthode non autorisée")
		return
	}
	if s.Web == nil {
		http.Error(w, "interface web absente de ce binaire", http.StatusNotFound)
		return
	}
	p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if p == "" {
		p = "index.html"
	}
	f, err := s.Web.Open(p)
	if err == nil {
		st, _ := f.Stat()
		f.Close()
		if st != nil && !st.IsDir() {
			if strings.HasPrefix(p, "assets/") {
				// Fichiers à empreinte (Vite) : cache long.
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			} else {
				w.Header().Set("Cache-Control", "no-cache")
			}
			http.ServeFileFS(w, r, s.Web, p)
			return
		}
	}
	if strings.HasPrefix(p, "assets/") {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFileFS(w, r, s.Web, "index.html")
}
