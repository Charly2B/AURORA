package api

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"aurora/internal/catalog"
	"aurora/internal/playback"
)

// Lien de lecture signé (lecteur externe) : valide, falsifié, expiré, profil sans droit.
func TestSignedStream(t *testing.T) {
	e := newEnv(t)
	admin := e.client()
	admin.do("POST", "/api/v1/setup/admin", map[string]string{"username": "admin", "password": "motdepasse1"}, nil)
	admin.do("POST", "/api/v1/admin/libraries", map[string]any{"kind": "movies", "path": filepath.Join(e.media, "Films")}, nil)
	e.waitScan()
	var movies struct{ Items []catalog.Card }
	admin.do("GET", "/api/v1/movies", nil, &movies)
	var d catalog.Detail
	admin.do("GET", fmt.Sprintf("/api/v1/items/%d", movies.Items[0].ID), nil, &d)
	var play map[string]any
	caps := playback.Caps{Containers: []string{"mp4"}, Video: []string{"h264"}, Audio: []string{"aac"}, HLS: "mse"}
	admin.do("POST", "/api/v1/play", map[string]any{"fileId": d.Files[0].ID, "caps": caps}, &play)
	if _, ok := play["signedUrl"]; ok {
		t.Fatal("lien signé fourni sans demande")
	}
	admin.do("POST", "/api/v1/play", map[string]any{"fileId": d.Files[0].ID, "caps": caps, "external": true}, &play)
	signed, _ := play["signedUrl"].(string)
	if !strings.Contains(signed, "/signed?") {
		t.Fatalf("lien signé absent : %v", play)
	}
	get := func(u string) int {
		req, _ := http.NewRequest("GET", e.srv.URL+u, nil) // sans cookie de session
		req.Header.Set("Range", "bytes=0-9")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp.StatusCode
	}
	if c := get(signed); c != 206 {
		t.Fatalf("lien valide : %d", c)
	}
	// Dernier caractère réellement changé (remplacer par « 0 » ne change rien une fois sur 16 : test instable).
	last := "0"
	if strings.HasSuffix(signed, "0") {
		last = "1"
	}
	if c := get(signed[:len(signed)-1] + last); c != 403 {
		t.Fatalf("signature falsifiée acceptée : %d", c)
	}
	if c := get(strings.Replace(signed, "/files/", "/files/9", 1)); c != 403 {
		t.Fatalf("autre fichier avec la même signature accepté : %d", c)
	}
	exp := e.s.signedURL(context.Background(), d.Files[0].ID, 1)
	expired := strings.Replace(exp, "e=", "e=1", 1) // date modifiée : signature invalide
	if c := get(expired); c != 403 {
		t.Fatalf("lien modifié accepté : %d", c)
	}
	// Profil enfant : le lien signé ne contourne pas le contrôle parental.
	admin.do("POST", "/api/v1/admin/users", map[string]any{"username": "enfant", "password": "motdepasse2", "role": "child", "allLibraries": true}, nil)
	var kidID int64
	e.s.DB.QueryRow(`SELECT id FROM users WHERE username='enfant'`).Scan(&kidID)
	if c := get(e.s.signedURL(context.Background(), d.Files[0].ID, kidID)); c != 404 {
		t.Fatalf("lien signé d'un profil sans droit : %d", c)
	}
	// Changement de mot de passe : les liens existants sont révoqués.
	admin.do("PUT", "/api/v1/me/password", map[string]string{"current": "motdepasse1", "new": "nouveaumotdepasse"}, nil)
	if c := get(signed); c != 403 {
		t.Fatalf("lien encore valable après changement de mot de passe : %d", c)
	}
}
