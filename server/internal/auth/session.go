package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

// SessionTTL : durée de vie d'une session (glissante : prolongée à l'usage).
const SessionTTL = 30 * 24 * time.Hour

// User : utilisateur authentifié.
type User struct {
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	Role         string `json:"role"` // admin, user, child
	MaxAge       *int   `json:"maxAge"`
	HideUnrated  bool   `json:"hideUnrated"`
	AllLibraries bool   `json:"allLibraries"`
	Prefs        string `json:"-"`
}

// IsAdmin indique un administrateur.
func (u *User) IsAdmin() bool { return u.Role == "admin" }

// Sessions gère les jetons de session stockés (hachés) en base.
type Sessions struct{ DB *sql.DB }

func hashToken(tok string) string {
	h := sha256.Sum256([]byte(tok))
	return hex.EncodeToString(h[:])
}

// Create crée une session et renvoie le jeton brut à placer dans le cookie.
func (s *Sessions) Create(ctx context.Context, userID int64, userAgent string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	tok := base64.RawURLEncoding.EncodeToString(b)
	now := time.Now()
	if len(userAgent) > 200 {
		userAgent = userAgent[:200]
	}
	_, err := s.DB.ExecContext(ctx, `INSERT INTO sessions(token_hash, user_id, created_at, expires_at, last_seen, user_agent)
		VALUES (?,?,?,?,?,?)`, hashToken(tok), userID, now.Unix(), now.Add(SessionTTL).Unix(), now.Unix(), userAgent)
	return tok, err
}

// ErrNoSession : jeton absent, inconnu ou expiré.
var ErrNoSession = errors.New("session invalide")

// Lookup valide un jeton et renvoie l'utilisateur. La session est prolongée au plus une fois par heure
// (évite une écriture SQLite à chaque requête, notamment pendant la lecture vidéo).
func (s *Sessions) Lookup(ctx context.Context, tok string) (*User, error) {
	if tok == "" {
		return nil, ErrNoSession
	}
	th := hashToken(tok)
	var u User
	var maxAge sql.NullInt64
	var expires, lastSeen int64
	err := s.DB.QueryRowContext(ctx, `SELECT u.id, u.username, u.role, u.max_age, u.hide_unrated, u.all_libraries, u.prefs,
		s.expires_at, s.last_seen
		FROM sessions s JOIN users u ON u.id = s.user_id WHERE s.token_hash = ?`, th).
		Scan(&u.ID, &u.Username, &u.Role, &maxAge, &u.HideUnrated, &u.AllLibraries, &u.Prefs, &expires, &lastSeen)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNoSession
	}
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if now.Unix() > expires {
		s.DB.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, th)
		return nil, ErrNoSession
	}
	if maxAge.Valid {
		v := int(maxAge.Int64)
		u.MaxAge = &v
	}
	if now.Unix()-lastSeen > 3600 {
		s.DB.ExecContext(ctx, `UPDATE sessions SET last_seen = ?, expires_at = ? WHERE token_hash = ?`,
			now.Unix(), now.Add(SessionTTL).Unix(), th)
	}
	return &u, nil
}

// Delete révoque une session.
func (s *Sessions) Delete(ctx context.Context, tok string) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, hashToken(tok))
	return err
}

// DeleteForUser révoque toutes les sessions d'un utilisateur (changement de mot de passe, suppression).
func (s *Sessions) DeleteForUser(ctx context.Context, userID int64) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ?`, userID)
	return err
}

// Purge supprime les sessions expirées.
func (s *Sessions) Purge(ctx context.Context) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at < ?`, time.Now().Unix())
	return err
}

// Limiter : limitation des tentatives de connexion par clé (IP et nom d'utilisateur).
// Après 5 échecs, attente exponentielle (30 s, 60 s… plafonnée à 15 min).
type Limiter struct {
	mu    sync.Mutex
	state map[string]*attempts
	now   func() time.Time
}

type attempts struct {
	failures int
	until    time.Time
	last     time.Time
}

// NewLimiter crée un limiteur.
func NewLimiter() *Limiter { return &Limiter{state: map[string]*attempts{}, now: time.Now} }

const freeAttempts = 5

// Allowed renvoie (true, 0) si une tentative est permise, sinon le délai restant.
func (l *Limiter) Allowed(keys ...string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	var wait time.Duration
	for _, k := range keys {
		if a := l.state[k]; a != nil && now.Before(a.until) {
			if d := a.until.Sub(now); d > wait {
				wait = d
			}
		}
	}
	return wait == 0, wait
}

// Fail enregistre un échec.
func (l *Limiter) Fail(keys ...string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	for _, k := range keys {
		a := l.state[k]
		if a == nil || now.Sub(a.last) > time.Hour {
			a = &attempts{}
			l.state[k] = a
		}
		a.failures++
		a.last = now
		if a.failures >= freeAttempts {
			d := 30 * time.Second << min(a.failures-freeAttempts, 5)
			a.until = now.Add(min(d, 15*time.Minute))
		}
	}
	if len(l.state) > 10000 { // borne mémoire : on oublie les entrées anciennes
		for k, a := range l.state {
			if now.Sub(a.last) > time.Hour {
				delete(l.state, k)
			}
		}
	}
}

// Success réinitialise les compteurs.
func (l *Limiter) Success(keys ...string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, k := range keys {
		delete(l.state, k)
	}
}
