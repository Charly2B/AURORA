package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os/exec"
	"path"
	"strings"
	"time"

	"aurora/internal/auth"
)

const maxBackgroundBytes = 50 << 20

type background struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	URL       string `json:"url"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Size      int64  `json:"size"`
	CreatedAt int64  `json:"createdAt"`
}

func backgroundURL(id string) string { return "/api/v1/backgrounds/" + id + "/image" }

func (s *Server) handleBackgrounds(w http.ResponseWriter, r *http.Request, u *auth.User) {
	rows, err := s.DB.QueryContext(r.Context(), `SELECT id,name,width,height,size,created_at FROM backgrounds ORDER BY created_at DESC,id`)
	if err != nil {
		fail(w, r, err)
		return
	}
	defer rows.Close()
	images := []background{}
	for rows.Next() {
		var b background
		if err := rows.Scan(&b.ID, &b.Name, &b.Width, &b.Height, &b.Size, &b.CreatedAt); err != nil {
			fail(w, r, err)
			return
		}
		b.URL = backgroundURL(b.ID)
		images = append(images, b)
	}
	if err := rows.Err(); err != nil {
		fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"images": images})
}

func (s *Server) handleBackgroundImage(w http.ResponseWriter, r *http.Request, u *auth.User) {
	var data []byte
	var mime string
	var created int64
	err := s.DB.QueryRowContext(r.Context(), `SELECT data,mime,created_at FROM backgrounds WHERE id=?`, r.PathValue("id")).Scan(&data, &mime, &created)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		fail(w, r, err)
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "private, max-age=2592000, immutable")
	w.Header().Set("ETag", `"`+r.PathValue("id")+`"`)
	http.ServeContent(w, r, "background", time.Unix(created, 0), bytes.NewReader(data))
}

// Imports atomiques en base : aucun chemin fourni par le client n’est utilisé sur disque.
func (s *Server) handleUploadBackground(w http.ResponseWriter, r *http.Request, u *auth.User) {
	release, err := s.acquireBackground(r.Context())
	if err != nil {
		fail(w, r, err)
		return
	}
	defer release()
	r.Body = http.MaxBytesReader(w, r.Body, maxBackgroundBytes+(64<<10))
	mr, err := r.MultipartReader()
	if err != nil {
		fail(w, r, badRequest("envoi multipart attendu (champ image)"))
		return
	}
	var data []byte
	var name string
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			backgroundUploadError(w, r, err)
			return
		}
		if part.FormName() != "image" || part.FileName() == "" || data != nil {
			fail(w, r, badRequest("un seul fichier attendu, dans le champ image"))
			return
		}
		name = path.Base(strings.ReplaceAll(part.FileName(), `\`, "/"))
		if len(name) > 240 {
			fail(w, r, badRequest("nom de fichier trop long"))
			return
		}
		data, err = io.ReadAll(io.LimitReader(part, maxBackgroundBytes+1))
		if err != nil {
			backgroundUploadError(w, r, err)
			return
		}
		if len(data) > maxBackgroundBytes {
			writeError(w, http.StatusRequestEntityTooLarge, "image limitée à 50 Mio")
			return
		}
	}
	if len(data) == 0 {
		fail(w, r, badRequest("image manquante"))
		return
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "jpeg" && format != "png") {
		fail(w, r, badRequest("image JPEG ou PNG valide requise"))
		return
	}
	// Les gros fichiers sont traités hors du processus Go, une image à la fois.
	// Ce plafond technique protège le NAS contre les en-têtes démesurés.
	if cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 32768 || cfg.Height > 32768 || int64(cfg.Width)*int64(cfg.Height) > 128000000 {
		fail(w, r, badRequest("image trop grande pour être traitée par le NAS"))
		return
	}
	if cfg.Width > 3840 || cfg.Height > 2160 {
		data, err = s.resizeBackground(r.Context(), data, format)
		if err != nil {
			fail(w, r, err)
			return
		}
		cfg, _, err = image.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			fail(w, r, err)
			return
		}
	}
	// Décoder après les dimensions : un en-tête valide seul ne suffit pas (image tronquée).
	if _, _, err := image.Decode(bytes.NewReader(data)); err != nil {
		fail(w, r, badRequest("image endommagée ou incomplète"))
		return
	}
	id := fmt.Sprintf("%x", sha256.Sum256(data))
	b := background{ID: id, Name: name, URL: backgroundURL(id), Width: cfg.Width, Height: cfg.Height, Size: int64(len(data)), CreatedAt: time.Now().Unix()}
	mime := "image/" + format
	result, err := s.DB.ExecContext(r.Context(), `INSERT INTO backgrounds(id,name,mime,width,height,size,created_at,data) VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(id) DO NOTHING`, id, name, mime, b.Width, b.Height, b.Size, b.CreatedAt, data)
	if err != nil {
		fail(w, r, err)
		return
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		fail(w, r, err)
		return
	}
	status := http.StatusCreated
	if inserted == 0 {
		status = http.StatusOK
		if err := s.DB.QueryRowContext(r.Context(), `SELECT name,created_at FROM backgrounds WHERE id=?`, id).Scan(&b.Name, &b.CreatedAt); err != nil {
			fail(w, r, err)
			return
		}
	}
	writeJSON(w, status, b)
}

func backgroundUploadError(w http.ResponseWriter, r *http.Request, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		writeError(w, http.StatusRequestEntityTooLarge, "image limitée à 50 Mio")
		return
	}
	fail(w, r, badRequest("envoi d’image incomplet ou invalide"))
}

// resizeBackground conserve proportions, format et transparence PNG, sans agrandissement.
func (s *Server) resizeBackground(ctx context.Context, data []byte, format string) ([]byte, error) {
	if s.Tools.FFmpeg == "" {
		return nil, &httpError{http.StatusServiceUnavailable, "adaptation des grandes images indisponible : FFmpeg manquant sur le serveur"}
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	codec, pixelFormat := "png", "rgba"
	if format == "jpeg" {
		codec, pixelFormat = "mjpeg", "yuvj444p"
	}
	// iw/ih sont évalués après autorotation EXIF ; le clamp à 1 garde les panoramas très fins.
	filter := "scale=w='max(1,trunc(iw*min(1,min(3840/iw,2160/ih))))':h='max(1,trunc(ih*min(1,min(3840/iw,2160/ih))))':flags=lanczos"
	cmd := exec.CommandContext(ctx, s.Tools.FFmpeg,
		"-hide_banner", "-loglevel", "error", "-nostdin", "-xerror", "-max_alloc", "536870912",
		"-threads", "1", "-err_detect", "explode", "-f", "image2pipe", "-c:v", codec, "-i", "pipe:0",
		"-map", "0:v:0", "-frames:v", "1", "-filter_threads", "1",
		"-vf", filter,
		"-threads", "1", "-c:v", codec, "-pix_fmt", pixelFormat, "-q:v", "2", "-f", "image2pipe", "pipe:1")
	cmd.Stdin = bytes.NewReader(data)
	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, badRequest("cette image n’a pas pu être adaptée ; vérifiez qu’elle est complète et lisible")
	}
	return out, nil
}

// Acquérir avant de lire le corps évite d’accumuler les fichiers en mémoire dans la file.
func (s *Server) acquireBackground(ctx context.Context) (func(), error) {
	s.backgroundOnce.Do(func() { s.backgroundGate = make(chan struct{}, 1) })
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	select {
	case s.backgroundGate <- struct{}{}:
		return func() { <-s.backgroundGate }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
