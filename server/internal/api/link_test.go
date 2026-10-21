package api

import (
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

// Association par code : la TV obtient une session sans mot de passe, une seule fois, et seulement
// après validation par un profil connecté.
func TestDeviceLink(t *testing.T) {
	e := newEnv(t)
	admin := e.client()
	admin.do("POST", "/api/v1/setup/admin", map[string]string{"username": "admin", "password": "motdepasse1"}, nil)

	tv := e.client()
	var start struct {
		Code      string `json:"code"`
		Poll      string `json:"poll"`
		ExpiresIn int    `json:"expiresIn"`
	}
	if c := tv.do("POST", "/api/v1/auth/link/start", map[string]string{"name": "Xiaomi Box"}, &start); c != 200 {
		t.Fatalf("start : %d", c)
	}
	if len(start.Code) != 7 || start.Code[3] != '-' || len(start.Poll) != 32 || start.ExpiresIn != 600 {
		t.Fatalf("réponse start inattendue : %+v", start)
	}
	var poll struct{ Status string }
	if c := tv.do("POST", "/api/v1/auth/link/poll", map[string]string{"poll": start.Poll}, &poll); c != 200 || poll.Status != "pending" {
		t.Fatalf("poll avant validation : %d %+v", c, poll)
	}
	if c := tv.do("GET", "/api/v1/auth/me", nil, nil); c != 401 {
		t.Fatalf("session avant validation : %d", c)
	}

	// Validation réservée à un profil connecté.
	anon := e.client()
	if c := anon.do("POST", "/api/v1/auth/link/approve", map[string]string{"code": start.Code}, nil); c != 401 {
		t.Fatalf("validation anonyme : %d", c)
	}
	// Mauvais code refusé ; casse et tiret indifférents pour le bon.
	if c := admin.do("POST", "/api/v1/auth/link/approve", map[string]string{"code": "AAA-AAA"}, nil); c != 404 {
		t.Fatalf("code faux : %d", c)
	}
	// Premier temps : description seule (nom, adresse, âge), rien n'est associé.
	var info struct {
		Name string
		IP   string
		OK   bool
	}
	if c := admin.do("POST", "/api/v1/auth/link/approve", map[string]string{"code": strings.ToLower(strings.ReplaceAll(start.Code, "-", " "))}, &info); c != 200 || info.Name != "Xiaomi Box" || info.IP != "127.0.0.1" || info.OK {
		t.Fatalf("description : %d %+v", c, info)
	}
	if c := tv.do("POST", "/api/v1/auth/link/poll", map[string]string{"poll": start.Poll}, &poll); c != 200 || poll.Status != "pending" {
		t.Fatalf("associé sans confirmation : %d %+v", c, poll)
	}
	// Second temps : confirmation explicite.
	if c := admin.do("POST", "/api/v1/auth/link/approve", map[string]any{"code": start.Code, "confirm": true}, &info); c != 200 || !info.OK {
		t.Fatalf("confirmation : %d %+v", c, info)
	}
	// Déjà validé : un second profil ne peut pas se l'approprier.
	if c := admin.do("POST", "/api/v1/auth/link/approve", map[string]any{"code": start.Code, "confirm": true}, nil); c != 404 {
		t.Fatalf("double validation : %d", c)
	}

	if c := tv.do("POST", "/api/v1/auth/link/poll", map[string]string{"poll": start.Poll}, &poll); c != 200 || poll.Status != "ok" {
		t.Fatalf("poll après validation : %d %+v", c, poll)
	}
	var me struct{ Username string }
	if c := tv.do("GET", "/api/v1/auth/me", nil, &me); c != 200 || me.Username != "admin" {
		t.Fatalf("session de la TV : %d", c)
	}
	// Usage unique : le secret ne redonne pas de session.
	other := e.client()
	if c := other.do("POST", "/api/v1/auth/link/poll", map[string]string{"poll": start.Poll}, nil); c != 410 {
		t.Fatalf("secret réutilisé : %d", c)
	}

	// Expiration après 10 min.
	tv2 := e.client()
	tv2.do("POST", "/api/v1/auth/link/start", map[string]string{"name": "TV"}, &start)
	e.s.links.mu.Lock()
	e.s.links.now = func() time.Time { return time.Now().Add(11 * time.Minute) }
	e.s.links.mu.Unlock()
	if c := admin.do("POST", "/api/v1/auth/link/approve", map[string]string{"code": start.Code}, nil); c != 404 {
		t.Fatalf("code expiré accepté : %d", c)
	}
	if c := tv2.do("POST", "/api/v1/auth/link/poll", map[string]string{"poll": start.Poll}, nil); c != 410 {
		t.Fatalf("poll expiré : %d", c)
	}
	e.s.links.mu.Lock()
	e.s.links.now = time.Now
	e.s.links.mu.Unlock()

	// Demande venue d'Internet via le proxy (dernier élément de X-Forwarded-For) : refusée. Un premier
	// élément falsifié par le client ne compte pas.
	startFrom := func(xff string) int {
		req, _ := http.NewRequest("POST", e.srv.URL+"/api/v1/auth/link/start", strings.NewReader("{}"))
		req.Header.Set("X-Aurora", "1")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", xff)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp.StatusCode
	}
	if c := startFrom("8.8.8.8"); c != 403 {
		t.Fatalf("demande depuis Internet : %d", c)
	}
	if c := startFrom("192.168.1.5, 8.8.8.8"); c != 403 {
		t.Fatalf("premier élément falsifié pris en compte : %d", c)
	}
	if c := startFrom("8.8.8.8, 192.168.1.5"); c != 200 {
		t.Fatalf("TV du réseau local derrière le proxy : %d", c)
	}

	// Plafond par adresse : 5 demandes en cours.
	spam := e.client()
	for i := 0; i < 5; i++ {
		spam.do("POST", "/api/v1/auth/link/start", map[string]string{}, nil)
	}
	if c := spam.do("POST", "/api/v1/auth/link/start", map[string]string{}, nil); c != 429 {
		t.Fatalf("plafond par adresse : %d", c)
	}
}

func TestOnLocalNetwork(t *testing.T) {
	for ip, want := range map[string]bool{
		"192.168.1.5": true, "10.0.0.2": true, "127.0.0.1": true, "::1": true, "fe80::1%en0": true,
		"::ffff:192.168.68.12": true, "fd00::5": true, "8.8.8.8": false, "2001:4860::8888": false, "pas une ip": false,
	} {
		if got := onLocalNetwork(ip); got != want {
			t.Errorf("%s → %v", ip, got)
		}
	}
	// Adresse globale d'une interface de la machine (IPv6 de la box) : acceptée si elle existe ici.
	ifaces, _ := net.InterfaceAddrs()
	for _, a := range ifaces {
		if n, ok := a.(*net.IPNet); ok && n.IP.IsGlobalUnicast() && !n.IP.IsPrivate() {
			if !onLocalNetwork(n.IP.String()) {
				t.Errorf("adresse de la machine %s refusée", n.IP)
			}
		}
	}
}

func TestNormalizeCode(t *testing.T) {
	for in, want := range map[string]string{"abc-def": "ABCDEF", " ABC DEF ": "ABCDEF", "K7M-2PQ": "K7M2PQ"} {
		if got, ok := normalizeCode(in); !ok || got != want {
			t.Errorf("%q → %q %v", in, got, ok)
		}
	}
	for _, in := range []string{"ABCDE", "ABCDEFG", "ABC-DE0", "ABCDEI", "<script>"} {
		if _, ok := normalizeCode(in); ok {
			t.Errorf("%q accepté", in)
		}
	}
}
