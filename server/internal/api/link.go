package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"

	"aurora/internal/auth"
)

// Association d'un appareil par code (comme plex.tv/link) : la TV affiche un code, un profil déjà connecté
// le valide sur /link, la TV reçoit alors sa propre session — sans saisir de mot de passe à la télécommande.
//
// Les demandes vivent en mémoire (perdues au redémarrage : la TV en redemande une). Le secret de suivi
// (`poll`, 128 bits) n'est connu que de la TV ; le code court ne sert qu'à la validation, réservée à un profil
// connecté et limitée par le limiteur d'échecs.

const (
	linkTTL        = 10 * time.Minute
	linkMaxPending = 100
	linkPerIP      = 5
	// Sans 0/O ni 1/I/L : lisible sur une TV et dictable.
	linkAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	linkCodeLen  = 6
)

type deviceLink struct {
	code    string
	name    string
	ip      string
	created time.Time
	userID  int64 // 0 tant que non validée
}

type linkStore struct {
	mu      sync.Mutex
	pending map[string]*deviceLink // clé : secret de suivi
	now     func() time.Time
}

func (l *linkStore) init() {
	if l.pending == nil {
		l.pending = map[string]*deviceLink{}
	}
	if l.now == nil {
		l.now = time.Now
	}
}

// purge retire les demandes expirées (appelé sous verrou).
func (l *linkStore) purge() {
	now := l.now()
	for k, d := range l.pending {
		if now.Sub(d.created) > linkTTL {
			delete(l.pending, k)
		}
	}
}

func randomCode() (string, error) {
	b := make([]byte, linkCodeLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	out := make([]byte, linkCodeLen)
	for i, v := range b {
		// 256 n'est pas multiple de 31 : biais négligeable pour un code à usage unique de 10 min.
		out[i] = linkAlphabet[int(v)%len(linkAlphabet)]
	}
	return string(out), nil
}

// onLocalNetwork : adresse privée, loopback, lien-local, ou dans un sous-réseau d'une interface du NAS.
// Ce dernier cas couvre l'IPv6 globale des box (2a01:… chez Orange), ni privée ni lien-local.
func onLocalNetwork(ip string) bool {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return false
	}
	addr = addr.Unmap().WithZone("") // « ::ffff:192.168.1.5 » = adresse IPv4 privée
	if addr.IsPrivate() || addr.IsLoopback() || addr.IsLinkLocalUnicast() {
		return true
	}
	ifaces, _ := net.InterfaceAddrs()
	for _, a := range ifaces {
		if n, ok := a.(*net.IPNet); ok {
			if p, err := netip.ParsePrefix(n.String()); err == nil && p.Masked().Contains(addr) {
				return true
			}
		}
	}
	return false
}

// uniqueCode tire un code absent des demandes en cours (appelé sous verrou).
func (l *linkStore) uniqueCode() (string, error) {
	for {
		code, err := randomCode()
		if err != nil {
			return "", err
		}
		taken := false
		for _, d := range l.pending {
			taken = taken || d.code == code
		}
		if !taken {
			return code, nil
		}
	}
}

// normalizeCode accepte « abc-def », « ABC DEF »… et rejette tout caractère hors alphabet.
func normalizeCode(s string) (string, bool) {
	var b strings.Builder
	for _, r := range strings.ToUpper(s) {
		if r == '-' || r == ' ' {
			continue
		}
		if !strings.ContainsRune(linkAlphabet, r) {
			return "", false
		}
		b.WriteRune(r)
	}
	return b.String(), b.Len() == linkCodeLen
}

// POST /api/v1/auth/link/start (public) : {name} → {code, poll, expiresIn}.
func (s *Server) handleLinkStart(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := readJSON(r, &body); err != nil {
		fail(w, r, err)
		return
	}
	name := strings.TrimSpace(body.Name)
	if len([]rune(name)) > 60 {
		name = string([]rune(name)[:60])
	}
	if name == "" {
		name = "Appareil"
	}
	ip := clientIP(r)
	// Une TV est sur le réseau local (ou derrière un VPN, en adresse privée) : refuser les demandes venues
	// d'Internet (proxy inverse exposé) coupe court à l'hameçonnage par code lancé de l'extérieur.
	if !onLocalNetwork(ip) {
		writeError(w, http.StatusForbidden, "association possible seulement depuis le réseau local")
		return
	}

	s.links.mu.Lock()
	defer s.links.mu.Unlock()
	s.links.init()
	s.links.purge()
	perIP := 0
	for _, d := range s.links.pending {
		if d.ip == ip {
			perIP++
		}
	}
	if perIP >= linkPerIP || len(s.links.pending) >= linkMaxPending {
		writeError(w, http.StatusTooManyRequests, "trop de demandes d'association en cours : réessayez dans quelques minutes")
		return
	}
	code, err := s.links.uniqueCode()
	if err != nil {
		fail(w, r, err)
		return
	}
	secret := make([]byte, 16)
	if _, err := rand.Read(secret); err != nil {
		fail(w, r, err)
		return
	}
	poll := hex.EncodeToString(secret)
	s.links.pending[poll] = &deviceLink{code: code, name: name, ip: ip, created: s.links.now()}
	writeJSON(w, http.StatusOK, map[string]any{
		"code": code[:3] + "-" + code[3:], "poll": poll, "expiresIn": int(linkTTL.Seconds()),
	})
}

// POST /api/v1/auth/link/poll (public) : {poll} → {status: "pending"} ou, une fois validée, session + {status: "ok"}.
func (s *Server) handleLinkPoll(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Poll string `json:"poll"`
	}
	if err := readJSON(r, &body); err != nil {
		fail(w, r, err)
		return
	}
	s.links.mu.Lock()
	s.links.init()
	s.links.purge()
	d := s.links.pending[body.Poll]
	var userID int64
	if d != nil && d.userID != 0 {
		userID = d.userID
		delete(s.links.pending, body.Poll) // usage unique
	}
	s.links.mu.Unlock()

	switch {
	case d == nil:
		writeError(w, http.StatusGone, "code expiré : un nouveau code va s'afficher")
	case userID == 0:
		writeJSON(w, http.StatusOK, map[string]any{"status": "pending"})
	default:
		if !s.startSession(w, r, userID) {
			// Échec de création : la demande validée est rendue, la TV réessaiera au suivi suivant.
			s.links.mu.Lock()
			s.links.pending[body.Poll] = d
			s.links.mu.Unlock()
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	}
}

// POST /api/v1/auth/link/approve (profil connecté), en deux temps contre l'hameçonnage par lien :
//   - {code} : décrit la demande (nom, adresse, âge) sans rien associer ;
//   - {code, confirm: true} : associe l'appareil à ce profil.
func (s *Server) handleLinkApprove(w http.ResponseWriter, r *http.Request, u *auth.User) {
	var body struct {
		Code    string `json:"code"`
		Confirm bool   `json:"confirm"`
	}
	if err := readJSON(r, &body); err != nil {
		fail(w, r, err)
		return
	}
	// Échecs comptés par profil et par adresse ; jamais remis à zéro par une réussite (sinon un profil
	// pourrait enchaîner les essais en validant ses propres codes).
	keys := []string{fmt.Sprintf("link:%d", u.ID), "linkip:" + clientIP(r)}
	if ok, wait := s.Limiter.Allowed(keys...); !ok {
		w.Header().Set("Retry-After", fmt.Sprint(int(wait.Seconds())+1))
		writeError(w, http.StatusTooManyRequests, fmt.Sprintf("trop d'essais : réessayez dans %d s", int(wait.Seconds())+1))
		return
	}
	code, valid := normalizeCode(body.Code)
	var found *deviceLink
	var info map[string]any
	if valid {
		s.links.mu.Lock()
		s.links.init()
		s.links.purge()
		now := s.links.now()
		for _, d := range s.links.pending {
			if d.code == code && d.userID == 0 {
				found = d
				info = map[string]any{"name": d.name, "ip": d.ip, "age": int(now.Sub(d.created).Seconds())}
				if body.Confirm {
					d.userID = u.ID
					// Validée juste avant l'expiration : on laisse à la TV le temps de son prochain suivi.
					if min := now.Add(-linkTTL + time.Minute); d.created.Before(min) {
						d.created = min
					}
				}
				break
			}
		}
		s.links.mu.Unlock()
	}
	if found == nil {
		s.Limiter.Fail(keys...)
		writeError(w, http.StatusNotFound, "code inconnu ou expiré : vérifiez le code affiché sur la TV")
		return
	}
	info["ok"] = body.Confirm
	writeJSON(w, http.StatusOK, info)
}
