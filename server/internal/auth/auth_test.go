package auth

import (
	"strings"
	"testing"
	"time"
)

func TestPassword(t *testing.T) {
	h, err := HashPassword("correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(h, "$argon2id$v=19$m=65536,t=3,p=2$") {
		t.Errorf("format inattendu : %s", h)
	}
	if ok, _ := CheckPassword(h, "correct horse battery"); !ok {
		t.Error("mot de passe correct refusé")
	}
	if ok, _ := CheckPassword(h, "wrong"); ok {
		t.Error("mot de passe incorrect accepté")
	}
	h2, _ := HashPassword("correct horse battery")
	if h == h2 {
		t.Error("le sel doit rendre chaque hachage unique")
	}
	if _, err := HashPassword("court"); err == nil {
		t.Error("mot de passe trop court accepté")
	}
}

func TestLimiter(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	l := NewLimiter()
	l.now = func() time.Time { return now }
	for i := 0; i < freeAttempts-1; i++ {
		l.Fail("ip:1")
	}
	if ok, _ := l.Allowed("ip:1"); !ok {
		t.Fatal("bloqué trop tôt")
	}
	l.Fail("ip:1")
	ok, wait := l.Allowed("ip:1")
	if ok || wait != 30*time.Second {
		t.Fatalf("attendu un blocage de 30 s, obtenu ok=%v wait=%v", ok, wait)
	}
	now = now.Add(31 * time.Second)
	if ok, _ := l.Allowed("ip:1"); !ok {
		t.Fatal("le blocage doit expirer")
	}
	l.Fail("ip:1")
	if _, wait := l.Allowed("ip:1"); wait != 60*time.Second {
		t.Fatalf("attente exponentielle attendue (60 s), obtenu %v", wait)
	}
	l.Success("ip:1")
	if ok, _ := l.Allowed("ip:1"); !ok {
		t.Fatal("un succès doit réinitialiser")
	}
}
