package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"aurora/internal/auth"
	"aurora/internal/catalog"
	"aurora/internal/config"
)

// --- Réglages (table settings) -------------------------------------------------------------------

func (s *Server) setting(ctx context.Context, key, def string) string {
	var v string
	if err := s.DB.QueryRowContext(ctx, `SELECT value FROM settings WHERE key=?`, key).Scan(&v); err != nil {
		return def
	}
	return v
}

func (s *Server) setSetting(ctx context.Context, key, value string) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO settings(key, value) VALUES (?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, value)
	return err
}

func (s *Server) userCount(ctx context.Context) int {
	var n int
	s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n
}

// --- Statut et premier démarrage -----------------------------------------------------------------

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	resp := map[string]any{
		"version":    config.Version,
		"hasAdmin":   s.userCount(ctx) > 0,
		"setupDone":  s.setting(ctx, "setup_step", "") == "done",
		"serverTime": time.Now().Unix(),
	}
	if u, err := s.currentUser(r); err == nil {
		resp["user"] = s.userJSON(u)
	}
	writeJSON(w, http.StatusOK, resp)
}

var usernameRe = regexp.MustCompile(`^[\p{L}\p{N}._-]{2,32}$`)

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// handleSetupAdmin crée le premier compte administrateur. Possible uniquement tant qu'aucun compte n'existe.
func (s *Server) handleSetupAdmin(w http.ResponseWriter, r *http.Request) {
	var c credentials
	if err := readJSON(r, &c); err != nil {
		fail(w, r, err)
		return
	}
	c.Username = strings.TrimSpace(c.Username)
	if !usernameRe.MatchString(c.Username) {
		fail(w, r, badRequest("nom d'utilisateur invalide (2 à 32 lettres, chiffres, . _ -)"))
		return
	}
	hash, err := auth.HashPassword(c.Password)
	if err != nil {
		fail(w, r, badRequest("%s", err.Error()))
		return
	}
	ctx := r.Context()
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		fail(w, r, err)
		return
	}
	defer tx.Rollback()
	var n int
	tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n) // dans la transaction : pas de course
	if n > 0 {
		writeError(w, http.StatusConflict, "un administrateur existe déjà")
		return
	}
	res, err := tx.ExecContext(ctx, `INSERT INTO users(username, password_hash, role, created_at) VALUES (?,?,?,?)`,
		c.Username, hash, "admin", time.Now().Unix())
	if err != nil {
		fail(w, r, err)
		return
	}
	id, _ := res.LastInsertId()
	if _, err := tx.ExecContext(ctx, `INSERT INTO settings(key, value) VALUES ('setup_step','language')
		ON CONFLICT(key) DO UPDATE SET value=excluded.value`); err != nil {
		fail(w, r, err)
		return
	}
	if err := tx.Commit(); err != nil {
		fail(w, r, err)
		return
	}
	if !s.startSession(w, r, id) {
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true})
}

// startSession crée la session et pose le cookie ; false si l'erreur a déjà été répondue.
func (s *Server) startSession(w http.ResponseWriter, r *http.Request, userID int64) bool {
	tok, err := s.Sessions.Create(r.Context(), userID, r.UserAgent())
	if err != nil {
		fail(w, r, err)
		return false
	}
	http.SetCookie(w, &http.Cookie{
		Name: cookieName, Value: tok, Path: "/", HttpOnly: true, Secure: isHTTPS(r),
		SameSite: http.SameSiteLaxMode, MaxAge: int(auth.SessionTTL.Seconds()),
	})
	return true
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var c credentials
	if err := readJSON(r, &c); err != nil {
		fail(w, r, err)
		return
	}
	c.Username = strings.TrimSpace(c.Username)
	ipKey, userKey := "ip:"+clientIP(r), "user:"+strings.ToLower(c.Username)
	if ok, wait := s.Limiter.Allowed(ipKey, userKey); !ok {
		w.Header().Set("Retry-After", fmt.Sprint(int(wait.Seconds())+1))
		writeError(w, http.StatusTooManyRequests, fmt.Sprintf("trop de tentatives : réessayez dans %d s", int(wait.Seconds())+1))
		return
	}
	var id int64
	var hash string
	err := s.DB.QueryRowContext(r.Context(), `SELECT id, password_hash FROM users WHERE username=?`, c.Username).Scan(&id, &hash)
	ok := false
	if err == nil {
		ok, _ = auth.CheckPassword(hash, c.Password)
	} else {
		auth.BurnTime(c.Password)
	}
	if !ok {
		s.Limiter.Fail(ipKey, userKey)
		writeError(w, http.StatusUnauthorized, "identifiant ou mot de passe incorrect")
		return
	}
	s.Limiter.Success(ipKey, userKey)
	if !s.startSession(w, r, id) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(cookieName); err == nil {
		s.Sessions.Delete(r.Context(), c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: "/", HttpOnly: true, MaxAge: -1, SameSite: http.SameSiteLaxMode})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// Prefs : préférences d'un profil.
type Prefs struct {
	Home         []catalog.HomePref `json:"home"`
	AutoplayNext bool               `json:"autoplayNext"`
	AudioLang    string             `json:"audioLang"`
	SubLang      string             `json:"subLang"`
	SubMode      string             `json:"subMode"` // auto | always | off
	SubSize      string             `json:"subSize"` // s | m | l
}

func parsePrefs(raw string) Prefs {
	p := Prefs{AutoplayNext: true, AudioLang: "fre", SubMode: "auto", SubSize: "m"}
	json.Unmarshal([]byte(raw), &p)
	p.Home = catalog.ResolveHomePrefs(p.Home)
	return p
}

func (s *Server) userJSON(u *auth.User) map[string]any {
	return map[string]any{"id": u.ID, "username": u.Username, "role": u.Role, "maxAge": u.MaxAge,
		"hideUnrated": u.HideUnrated, "prefs": parsePrefs(u.Prefs)}
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request, u *auth.User) {
	writeJSON(w, http.StatusOK, s.userJSON(u))
}

func (s *Server) handlePutPrefs(w http.ResponseWriter, r *http.Request, u *auth.User) {
	var p Prefs
	if err := readJSON(r, &p); err != nil {
		fail(w, r, err)
		return
	}
	if p.SubMode != "auto" && p.SubMode != "always" && p.SubMode != "off" {
		p.SubMode = "auto"
	}
	if p.SubSize != "s" && p.SubSize != "m" && p.SubSize != "l" {
		p.SubSize = "m"
	}
	p.Home = catalog.ResolveHomePrefs(p.Home)
	b, _ := json.Marshal(p)
	if _, err := s.DB.ExecContext(r.Context(), `UPDATE users SET prefs=? WHERE id=?`, string(b), u.ID); err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request, u *auth.User) {
	var req struct {
		Current string `json:"current"`
		New     string `json:"new"`
	}
	if err := readJSON(r, &req); err != nil {
		fail(w, r, err)
		return
	}
	var hash string
	if err := s.DB.QueryRowContext(r.Context(), `SELECT password_hash FROM users WHERE id=?`, u.ID).Scan(&hash); err != nil {
		fail(w, r, err)
		return
	}
	if ok, _ := auth.CheckPassword(hash, req.Current); !ok {
		writeError(w, http.StatusForbidden, "mot de passe actuel incorrect")
		return
	}
	nh, err := auth.HashPassword(req.New)
	if err != nil {
		fail(w, r, badRequest("%s", err.Error()))
		return
	}
	if _, err := s.DB.ExecContext(r.Context(), `UPDATE users SET password_hash=? WHERE id=?`, nh, u.ID); err != nil {
		fail(w, r, err)
		return
	}
	s.Sessions.DeleteForUser(r.Context(), u.ID) // déconnecte les autres appareils
	if !s.startSession(w, r, u.ID) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// --- Utilisateurs (admin) ------------------------------------------------------------------------

type userIn struct {
	Username     string  `json:"username"`
	Password     string  `json:"password"`
	Role         string  `json:"role"`
	MaxAge       *int    `json:"maxAge"`
	HideUnrated  bool    `json:"hideUnrated"`
	AllLibraries bool    `json:"allLibraries"`
	Libraries    []int64 `json:"libraries"`
}

func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request, u *auth.User) {
	rows, err := s.DB.QueryContext(r.Context(), `SELECT id, username, role, max_age, hide_unrated, all_libraries, created_at,
		(SELECT json_group_array(library_id) FROM user_libraries WHERE user_id=users.id) FROM users ORDER BY id`)
	if err != nil {
		fail(w, r, err)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, created int64
		var name, role, libs string
		var maxAge sql.NullInt64
		var hide, all bool
		if rows.Scan(&id, &name, &role, &maxAge, &hide, &all, &created, &libs) != nil {
			continue
		}
		var l []int64
		json.Unmarshal([]byte(libs), &l)
		var ma any
		if maxAge.Valid {
			ma = maxAge.Int64
		}
		out = append(out, map[string]any{"id": id, "username": name, "role": role, "maxAge": ma, "hideUnrated": hide,
			"allLibraries": all, "libraries": l, "createdAt": created})
	}
	writeJSON(w, http.StatusOK, out)
}

func validRole(r string) bool { return r == "admin" || r == "user" || r == "child" }

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request, u *auth.User) {
	var in userIn
	if err := readJSON(r, &in); err != nil {
		fail(w, r, err)
		return
	}
	in.Username = strings.TrimSpace(in.Username)
	if !usernameRe.MatchString(in.Username) || !validRole(in.Role) {
		fail(w, r, badRequest("nom d'utilisateur ou rôle invalide"))
		return
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		fail(w, r, badRequest("%s", err.Error()))
		return
	}
	if in.Role == "child" && in.MaxAge == nil {
		v := 10
		in.MaxAge, in.HideUnrated = &v, true // valeurs prudentes par défaut pour un profil enfant
	}
	ctx := r.Context()
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		fail(w, r, err)
		return
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `INSERT INTO users(username, password_hash, role, max_age, hide_unrated, all_libraries, created_at)
		VALUES (?,?,?,?,?,?,?)`, in.Username, hash, in.Role, in.MaxAge, in.HideUnrated, in.AllLibraries, time.Now().Unix())
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			fail(w, r, &httpError{http.StatusConflict, "ce nom d'utilisateur existe déjà"})
			return
		}
		fail(w, r, err)
		return
	}
	id, _ := res.LastInsertId()
	if err := setUserLibraries(ctx, tx, id, in.Libraries); err != nil {
		fail(w, r, err)
		return
	}
	if err := tx.Commit(); err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": id})
}

func setUserLibraries(ctx context.Context, tx *sql.Tx, userID int64, libs []int64) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM user_libraries WHERE user_id=?`, userID); err != nil {
		return err
	}
	for _, l := range libs {
		if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO user_libraries(user_id, library_id) SELECT ?, id FROM libraries WHERE id=?`, userID, l); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request, u *auth.User) {
	id, err := pathID(r, "id")
	if err != nil {
		fail(w, r, err)
		return
	}
	var in userIn
	if err := readJSON(r, &in); err != nil {
		fail(w, r, err)
		return
	}
	if !validRole(in.Role) {
		fail(w, r, badRequest("rôle invalide"))
		return
	}
	ctx := r.Context()
	if id == u.ID && in.Role != "admin" {
		fail(w, r, badRequest("vous ne pouvez pas retirer vos propres droits d'administrateur"))
		return
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		fail(w, r, err)
		return
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE users SET role=?, max_age=?, hide_unrated=?, all_libraries=? WHERE id=?`,
		in.Role, in.MaxAge, in.HideUnrated, in.AllLibraries, id); err != nil {
		fail(w, r, err)
		return
	}
	if in.Password != "" {
		hash, err := auth.HashPassword(in.Password)
		if err != nil {
			fail(w, r, badRequest("%s", err.Error()))
			return
		}
		if _, err := tx.ExecContext(ctx, `UPDATE users SET password_hash=? WHERE id=?`, hash, id); err != nil {
			fail(w, r, err)
			return
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE user_id=?`, id); err != nil {
			fail(w, r, err)
			return
		}
	}
	if err := setUserLibraries(ctx, tx, id, in.Libraries); err != nil {
		fail(w, r, err)
		return
	}
	if err := tx.Commit(); err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request, u *auth.User) {
	id, err := pathID(r, "id")
	if err != nil {
		fail(w, r, err)
		return
	}
	if id == u.ID {
		fail(w, r, badRequest("vous ne pouvez pas supprimer votre propre compte"))
		return
	}
	var admins int
	s.DB.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM users WHERE role='admin' AND id<>?`, id).Scan(&admins)
	if admins == 0 {
		fail(w, r, badRequest("il doit rester au moins un administrateur"))
		return
	}
	res, err := s.DB.ExecContext(r.Context(), `DELETE FROM users WHERE id=?`, id)
	if err != nil {
		fail(w, r, err)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		fail(w, r, catalog.ErrNotFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
