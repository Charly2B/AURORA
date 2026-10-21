package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"aurora/internal/auth"
	"aurora/internal/catalog"
)

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request, u *auth.User) {
	h, err := s.Store.Home(r.Context(), viewer(u), parsePrefs(u.Prefs).Home)
	if err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, h)
}

// handleLibraries : bibliothèques visibles par le profil, avec le nombre de films et de séries.
func (s *Server) handleLibraries(w http.ResponseWriter, r *http.Request, u *auth.User) {
	libs, err := s.Store.Libraries(r.Context(), viewer(u))
	if err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, libs)
}

func qInt(r *http.Request, k string) int {
	v, _ := strconv.Atoi(r.URL.Query().Get(k))
	return v
}

func (s *Server) handleList(kind string) userHandler {
	return func(w http.ResponseWriter, r *http.Request, u *auth.User) {
		q := r.URL.Query()
		lq := catalog.ListQuery{
			Kind: kind, Sort: q.Get("sort"), Desc: q.Get("order") == "desc", Genre: q.Get("genre"),
			Year: qInt(r, "year"), Decade: qInt(r, "decade"), Resolution: q.Get("resolution"), HDR: q.Get("hdr") == "1",
			Watched: q.Get("watched"), Favorite: q.Get("favorite") == "1", Watchlist: q.Get("watchlist") == "1",
			Person: q.Get("person"), Language: q.Get("language"), Offset: qInt(r, "offset"), Limit: qInt(r, "limit"),
			LibraryID: int64(qInt(r, "library")), Collection: int64(qInt(r, "collection")),
		}
		items, total, err := s.Store.List(r.Context(), viewer(u), lq)
		if err != nil {
			fail(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": total})
	}
}

func (s *Server) handleFacets(w http.ResponseWriter, r *http.Request, u *auth.User) {
	kind := r.PathValue("kind")
	if kind != "movies" && kind != "series" {
		fail(w, r, badRequest("type inconnu"))
		return
	}
	k := "movie"
	if kind == "series" {
		k = "series"
	}
	f, err := s.Store.Facets(r.Context(), viewer(u), k, int64(qInt(r, "library")))
	if err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, f)
}

func (s *Server) handleItem(w http.ResponseWriter, r *http.Request, u *auth.User) {
	id, err := pathID(r, "id")
	if err != nil {
		fail(w, r, err)
		return
	}
	d, err := s.Store.Item(r.Context(), viewer(u), id)
	if err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (s *Server) handleEpisodes(w http.ResponseWriter, r *http.Request, u *auth.User) {
	id, err := pathID(r, "id")
	if err != nil {
		fail(w, r, err)
		return
	}
	season, err := strconv.Atoi(r.PathValue("season"))
	if err != nil {
		fail(w, r, badRequest("saison invalide"))
		return
	}
	v := viewer(u)
	if !s.Store.ItemVisible(r.Context(), v, id) {
		fail(w, r, catalog.ErrNotFound)
		return
	}
	eps, err := s.Store.Episodes(r.Context(), v, id, season)
	if err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, eps)
}

func (s *Server) handleNextEpisode(w http.ResponseWriter, r *http.Request, u *auth.User) {
	id, err := pathID(r, "id")
	if err != nil {
		fail(w, r, err)
		return
	}
	c, err := s.Store.NextEpisode(r.Context(), viewer(u), id)
	if err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"next": c})
}

func (s *Server) handleSetState(w http.ResponseWriter, r *http.Request, u *auth.User) {
	id, err := pathID(r, "id")
	if err != nil {
		fail(w, r, err)
		return
	}
	var req struct {
		Favorite  *bool `json:"favorite"`
		Watchlist *bool `json:"watchlist"`
		Watched   *bool `json:"watched"`
		Season    *int  `json:"season"`
	}
	if err := readJSON(r, &req); err != nil {
		fail(w, r, err)
		return
	}
	ctx := r.Context()
	if !s.Store.ItemVisible(ctx, viewer(u), id) {
		fail(w, r, catalog.ErrNotFound)
		return
	}
	if req.Favorite != nil {
		err = s.Store.SetFlag(ctx, u.ID, id, "favorite", *req.Favorite)
	}
	if err == nil && req.Watchlist != nil {
		err = s.Store.SetFlag(ctx, u.ID, id, "watchlist", *req.Watchlist)
	}
	if err == nil && req.Watched != nil {
		err = s.Store.SetWatched(ctx, u.ID, id, *req.Watched, req.Season)
	}
	if err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// POST /api/v1/items/{id}/continue/hide : « Retirer de Continuer à regarder » (fichier et progression intacts).
func (s *Server) handleHideContinue(w http.ResponseWriter, r *http.Request, u *auth.User) {
	id, err := pathID(r, "id")
	if err != nil {
		fail(w, r, err)
		return
	}
	ctx := r.Context()
	if !s.Store.ItemVisible(ctx, viewer(u), id) {
		fail(w, r, catalog.ErrNotFound)
		return
	}
	if err := s.Store.HideFromContinue(ctx, u.ID, id); err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleProgress(w http.ResponseWriter, r *http.Request, u *auth.User) {
	id, err := pathID(r, "id")
	if err != nil {
		fail(w, r, err)
		return
	}
	var req struct {
		FileID   int64   `json:"fileId"`
		Position float64 `json:"position"`
		Duration float64 `json:"duration"`
	}
	if err := readJSON(r, &req); err != nil {
		fail(w, r, err)
		return
	}
	if req.Position < 0 || req.Duration < 0 || req.Duration > 48*3600 {
		fail(w, r, badRequest("position invalide"))
		return
	}
	ctx := r.Context()
	var owner int64
	if err := s.DB.QueryRowContext(ctx, `SELECT item_id FROM media_files WHERE id=?`, req.FileID).Scan(&owner); err != nil || owner != id {
		fail(w, r, badRequest("fichier invalide"))
		return
	}
	if !s.Store.ItemVisible(ctx, viewer(u), id) {
		fail(w, r, catalog.ErrNotFound)
		return
	}
	watched, err := s.Store.SaveProgress(ctx, u.ID, id, req.FileID, req.Position, req.Duration)
	if err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"watched": watched})
}

func (s *Server) handleCollections(w http.ResponseWriter, r *http.Request, u *auth.User) {
	cs, err := s.Store.Collections(r.Context(), viewer(u))
	if err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, cs)
}

func (s *Server) handleCollection(w http.ResponseWriter, r *http.Request, u *auth.User) {
	id, err := pathID(r, "id")
	if err != nil {
		fail(w, r, err)
		return
	}
	name, items, err := s.Store.CollectionItems(r.Context(), viewer(u), id)
	if err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"name": name, "items": items})
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request, u *auth.User) {
	q := r.URL.Query().Get("q")
	if len(q) > 200 {
		q = q[:200]
	}
	res, err := s.Store.Search(r.Context(), viewer(u), q, qInt(r, "limit"))
	if err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

var imageName = regexp.MustCompile(`^[0-9a-f]{40}\.(jpg|jpeg|png|webp)$`)

// handleImage sert une image du cache local (affiches, fonds, portraits). Le nom est contraint par motif.
func (s *Server) handleImage(w http.ResponseWriter, r *http.Request, u *auth.User) {
	name := r.PathValue("name")
	if !imageName.MatchString(name) {
		http.NotFound(w, r)
		return
	}
	p := filepath.Join(s.ImageDir, name)
	if _, err := os.Stat(p); err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "private, max-age=2592000, immutable")
	http.ServeFile(w, r, p)
}

// handleThumb : image extraite de la vidéo (œuvres sans affiche, épisodes sans vignette).
func (s *Server) handleThumb(w http.ResponseWriter, r *http.Request, u *auth.User) {
	id, err := pathID(r, "id")
	if err != nil {
		fail(w, r, err)
		return
	}
	ctx := r.Context()
	if !s.Store.ItemVisible(ctx, viewer(u), id) {
		http.NotFound(w, r)
		return
	}
	var fileID, mtime int64
	var path string
	var duration float64
	err = s.DB.QueryRowContext(ctx, `SELECT id, path, mtime, COALESCE(duration,0) FROM media_files WHERE item_id=? AND present=1
		ORDER BY COALESCE(height,0) ASC LIMIT 1`, id).Scan(&fileID, &path, &mtime, &duration)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	p, err := s.Extract.Thumbnail(ctx, fmt.Sprintf("%d-%d", fileID, mtime), path, duration)
	if err != nil {
		w.Header().Set("Cache-Control", "no-store")
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "private, max-age=604800")
	http.ServeFile(w, r, p)
}
