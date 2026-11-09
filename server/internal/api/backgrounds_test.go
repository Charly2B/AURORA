package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aurora/internal/tools"
)

func uploadBackground(t *testing.T, c *client, name string, data []byte, csrf bool) (int, []byte) {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("image", name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = part.Write(data); err != nil {
		t.Fatal(err)
	}
	if err = mw.Close(); err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest("POST", c.e.srv.URL+"/api/v1/admin/backgrounds", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if csrf {
		req.Header.Set("X-Aurora", "1")
	}
	resp, err := c.c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, b
}

func TestBackgrounds(t *testing.T) {
	e := newEnv(t)
	admin := e.client()
	admin.do("POST", "/api/v1/setup/admin", map[string]string{"username": "admin", "password": "motdepasse1"}, nil)
	var gallery struct {
		Images []struct {
			ID, Name, URL string
			Width, Height int
			Size          int64
		}
	}
	if code := admin.do("GET", "/api/v1/backgrounds", nil, &gallery); code != 200 || gallery.Images == nil || len(gallery.Images) != 0 {
		t.Fatalf("galerie vide : %d %+v", code, gallery)
	}
	var img bytes.Buffer
	if err := png.Encode(&img, image.NewRGBA(image.Rect(0, 0, 4, 2))); err != nil {
		t.Fatal(err)
	}
	code, body := uploadBackground(t, admin, "Nuit.png", img.Bytes(), true)
	var imported struct {
		ID, Name, URL string
		Width, Height int
		Size          int64
	}
	if err := json.Unmarshal(body, &imported); code != 201 || err != nil || imported.Name != "Nuit.png" || imported.Width != 4 || imported.Height != 2 || imported.Size != int64(img.Len()) {
		t.Fatalf("import : %d %s (%v)", code, body, err)
	}
	code, duplicate := uploadBackground(t, admin, "autre nom.png", img.Bytes(), true)
	var same struct{ ID string }
	json.Unmarshal(duplicate, &same)
	if code != 200 || same.ID != imported.ID {
		t.Fatalf("déduplication : %d %s", code, duplicate)
	}
	admin.do("GET", "/api/v1/backgrounds", nil, &gallery)
	if len(gallery.Images) != 1 {
		t.Fatalf("galerie : %+v", gallery)
	}
	resp, err := admin.c.Get(e.srv.URL + imported.URL)
	if err != nil {
		t.Fatal(err)
	}
	served, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 || !bytes.Equal(served, img.Bytes()) || resp.Header.Get("Content-Type") != "image/png" || !strings.Contains(resp.Header.Get("Cache-Control"), "private") {
		t.Fatalf("image : %d %v", resp.StatusCode, resp.Header)
	}
	// Une sauvegarde SQLite contient aussi les octets de l’image.
	backup, err := e.s.DB.BackupTo(t.Context(), "with-background.db")
	if err != nil {
		t.Fatal(err)
	}
	saved, err := sql.Open("sqlite", backup)
	if err != nil {
		t.Fatal(err)
	}
	defer saved.Close()
	var backedUp []byte
	if err := saved.QueryRow("SELECT data FROM backgrounds WHERE id=?", imported.ID).Scan(&backedUp); err != nil || !bytes.Equal(backedUp, img.Bytes()) {
		t.Fatalf("image absente de la sauvegarde : %v", err)
	}
	if code := e.client().do("GET", imported.URL, nil, nil); code != 401 {
		t.Fatalf("image anonyme : %d", code)
	}
	if code := e.client().do("GET", "/api/v1/backgrounds", nil, nil); code != 401 {
		t.Fatalf("galerie anonyme : %d", code)
	}
	admin.do("POST", "/api/v1/admin/users", map[string]any{"username": "user", "password": "motdepasse2", "role": "user", "allLibraries": true}, nil)
	user := e.client()
	user.do("POST", "/api/v1/auth/login", map[string]string{"username": "user", "password": "motdepasse2"}, nil)
	if code := user.do("GET", imported.URL, nil, nil); code != 200 {
		t.Fatalf("image pour client connecté : %d", code)
	}
	if code, _ := uploadBackground(t, user, "test.png", img.Bytes(), true); code != 403 {
		t.Fatalf("import non-admin : %d", code)
	}
	if code, _ := uploadBackground(t, admin, "test.png", img.Bytes(), false); code != 403 {
		t.Fatalf("import sans CSRF : %d", code)
	}
	for _, invalid := range [][]byte{[]byte("<svg></svg>"), img.Bytes()[:img.Len()/2]} {
		if code, body := uploadBackground(t, admin, "test.png", invalid, true); code != 400 {
			t.Fatalf("fausse image : %d %s", code, body)
		}
	}
	var wide bytes.Buffer
	png.Encode(&wide, image.NewGray(image.Rect(0, 0, 8193, 1)))
	if code, body := uploadBackground(t, admin, "wide.png", wide.Bytes(), true); code != 201 {
		t.Fatalf("panorama redimensionné attendu : %d %s", code, body)
	}
	if code, _ := uploadBackground(t, admin, "large.png", bytes.Repeat([]byte("x"), 51<<20), true); code != 413 {
		t.Fatalf("taille excessive : %d", code)
	}
	var jpg bytes.Buffer
	jpeg.Encode(&jpg, image.NewRGBA(image.Rect(0, 0, 4, 2)), nil)
	if code, body := uploadBackground(t, admin, "photo.jpg", jpg.Bytes(), true); code != 201 {
		t.Fatalf("JPEG : %d %s", code, body)
	}
}

// Les images autrefois refusées (côté >8192 ou >16 MP) doivent devenir des fonds exploitables.
func TestBackgroundResize(t *testing.T) {
	e := newEnv(t)
	admin := e.client()
	admin.do("POST", "/api/v1/setup/admin", map[string]string{"username": "admin", "password": "motdepasse1"}, nil)
	for _, tc := range []struct {
		format                               string
		width, height, wantWidth, wantHeight int
		oriented                             bool
	}{
		{"png", 10000, 2000, 3840, 768, false},
		{"jpeg", 6000, 4000, 3240, 2160, false},
		{"png", 8193, 1, 3840, 1, false},
		{"jpeg", 6000, 4000, 1440, 2160, true},
	} {
		t.Run(tc.format+"-"+fmt.Sprint(tc.width), func(t *testing.T) {
			var source bytes.Buffer
			var img image.Image = image.NewGray(image.Rect(0, 0, tc.width, tc.height))
			if tc.height == 1 {
				transparent := image.NewNRGBA(image.Rect(0, 0, tc.width, tc.height))
				for x := 0; x < tc.width; x++ {
					alpha := uint8(128)
					if x < 32 {
						alpha = 0
					}
					if x >= tc.width-32 {
						alpha = 255
					}
					transparent.SetNRGBA(x, 0, color.NRGBA{R: 200, G: 100, B: 50, A: alpha})
				}
				img = transparent
			}
			var err error
			if tc.format == "png" {
				err = png.Encode(&source, img)
			} else {
				err = jpeg.Encode(&source, img, nil)
			}
			if err != nil {
				t.Fatal(err)
			}
			if tc.oriented {
				// APP1 Exif, TIFF little-endian, Orientation=6 (rotation horaire).
				exif := []byte{0xff, 0xe1, 0, 34, 'E', 'x', 'i', 'f', 0, 0, 'I', 'I', 42, 0, 8, 0, 0, 0, 1, 0, 0x12, 1, 3, 0, 1, 0, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0}
				oriented := append([]byte{}, source.Bytes()[:2]...)
				oriented = append(oriented, exif...)
				oriented = append(oriented, source.Bytes()[2:]...)
				source.Reset()
				source.Write(oriented)
			}
			code, body := uploadBackground(t, admin, "large."+tc.format, source.Bytes(), true)
			var imported struct {
				ID, URL       string
				Width, Height int
				Size          int64
			}
			if err := json.Unmarshal(body, &imported); code != 201 || err != nil || imported.Width != tc.wantWidth || imported.Height != tc.wantHeight {
				t.Fatalf("import redimensionné : %d %s (%v)", code, body, err)
			}
			resp, err := admin.c.Get(e.srv.URL + imported.URL)
			if err != nil {
				t.Fatal(err)
			}
			data, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil || resp.StatusCode != 200 || resp.Header.Get("Content-Type") != "image/"+tc.format || int64(len(data)) != imported.Size {
				t.Fatalf("image adaptée illisible : %d %v", resp.StatusCode, err)
			}
			decoded, format, err := image.Decode(bytes.NewReader(data))
			if err != nil || format != tc.format || decoded.Bounds().Dx() != tc.wantWidth || decoded.Bounds().Dy() != tc.wantHeight {
				t.Fatalf("dimensions réelles incorrectes : %s %v", format, err)
			}
			if tc.height == 1 {
				// Un niveau d’arrondi swscale est admis ; aucun aplatissement de la transparence.
				if _, _, _, alpha := decoded.At(100, 0).RGBA(); alpha < 127*257 || alpha > 129*257 {
					t.Fatalf("transparence PNG perdue : %d", alpha)
				}
			}
			if tc.height == 1 {
				if _, _, _, alpha := decoded.At(0, 0).RGBA(); alpha != 0 {
					t.Fatalf("zone transparente perdue : %d", alpha)
				}
				if _, _, _, alpha := decoded.At(tc.wantWidth-1, 0).RGBA(); alpha != 65535 {
					t.Fatalf("zone opaque modifiée : %d", alpha)
				}
			}
			if tc.format == "png" && tc.height > 1 {
				if code, _ := uploadBackground(t, admin, "tronquée.png", source.Bytes()[:source.Len()/2], true); code != 400 {
					t.Fatalf("grande image tronquée acceptée : %d", code)
				}
			}
			code, body = uploadBackground(t, admin, "doublon."+tc.format, source.Bytes(), true)
			var duplicate struct{ ID string }
			if err := json.Unmarshal(body, &duplicate); code != 200 || err != nil || duplicate.ID != imported.ID {
				t.Fatalf("déduplication après adaptation : %d %s", code, body)
			}
		})
	}
}

func TestBackgroundCanceledUploadDoesNotWaitForConversion(t *testing.T) {
	s := &Server{Tools: tools.Tools{FFmpeg: "ffmpeg"}}
	release, err := s.acquireBackground(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("image", "large.png")
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(part, image.NewGray(image.Rect(0, 0, 8193, 1))); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	req := httptest.NewRequest("POST", "/api/v1/admin/backgrounds", &body).WithContext(ctx)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	done := make(chan struct{})
	go func() { defer close(done); s.handleUploadBackground(httptest.NewRecorder(), req, nil) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("l’import annulé attend encore la conversion active")
	}
}
