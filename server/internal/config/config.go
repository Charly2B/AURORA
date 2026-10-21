// Package config : configuration de démarrage (variables d'environnement / options) et journal tournant.
package config

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// Version est injectée à la compilation (-ldflags "-X aurora/internal/config.Version=…").
var Version = "dev"

// Config : paramètres de démarrage. Les réglages modifiables depuis l'interface sont en base (table settings).
type Config struct {
	Listen      string // adresse d'écoute, ex. 0.0.0.0:8890
	DataDir     string // base, sauvegardes, cache d'images (persistant)
	TmpDir      string // segments HLS (temporaire)
	LogFile     string // vide = sortie standard
	FFmpeg      string // chemin explicite (sinon recherche automatique)
	FFprobe     string
	BrowseRoots []string // dossiers explorables imposés par -browse-roots (vide = détection automatique)
	Dev         bool     // sert le frontend depuis le disque (WebDir) au lieu du binaire
	WebDir      string
}

// Load lit options et variables AURORA_*. Les options l'emportent.
func Load(args []string) (*Config, error) {
	fs := flag.NewFlagSet("aurora", flag.ContinueOnError)
	c := &Config{}
	fs.StringVar(&c.Listen, "listen", env("AURORA_LISTEN", "0.0.0.0:8890"), "adresse d'écoute")
	fs.StringVar(&c.DataDir, "data", env("AURORA_DATA_DIR", "./aurora-data"), "dossier de données persistantes")
	fs.StringVar(&c.TmpDir, "tmp", env("AURORA_TMP_DIR", ""), "dossier temporaire (défaut : <data>/tmp)")
	fs.StringVar(&c.LogFile, "log", env("AURORA_LOG_FILE", ""), "fichier journal (défaut : sortie standard)")
	fs.StringVar(&c.FFmpeg, "ffmpeg", env("AURORA_FFMPEG", ""), "chemin de ffmpeg")
	fs.StringVar(&c.FFprobe, "ffprobe", env("AURORA_FFPROBE", ""), "chemin de ffprobe")
	roots := fs.String("browse-roots", env("AURORA_BROWSE_ROOTS", ""), "dossiers explorables, séparés par « : »")
	fs.BoolVar(&c.Dev, "dev", env("AURORA_DEV", "") == "1", "mode développement")
	fs.StringVar(&c.WebDir, "web", env("AURORA_WEB_DIR", "web/dist"), "frontend (mode dev)")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	var err error
	if c.DataDir, err = filepath.Abs(c.DataDir); err != nil {
		return nil, err
	}
	if c.TmpDir == "" {
		c.TmpDir = filepath.Join(c.DataDir, "tmp")
	}
	if *roots != "" {
		for _, r := range strings.Split(*roots, ":") {
			if r = strings.TrimSpace(r); r != "" {
				c.BrowseRoots = append(c.BrowseRoots, filepath.Clean(r))
			}
		}
	}
	return c, nil
}

// Roots renvoie les dossiers explorables. Sans -browse-roots, ils sont recalculés à chaque appel :
// un disque USB branché après le démarrage d'AURORA apparaît sans redémarrage.
func (c *Config) Roots() []string {
	if len(c.BrowseRoots) > 0 {
		return c.BrowseRoots
	}
	return defaultRoots()
}

// defaultRoots : sur DSM, les volumes internes (/volume1…) ET les disques externes : USB monté sur
// /volumeUSB1/usbshare, eSATA sur /volumeSATA1/… Ailleurs (développement), le dossier personnel.
func defaultRoots() []string {
	out := volumesIn("/")
	if len(out) == 0 {
		if h, err := os.UserHomeDir(); err == nil {
			out = append(out, h)
		}
	}
	return out
}

var volumeName = regexp.MustCompile(`^volume(\d+|USB\d+|SATA\d+)$`)

// volumesIn liste les volumes Synology présents sous base (paramétrable pour les tests).
func volumesIn(base string) []string {
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if !volumeName.MatchString(e.Name()) {
			continue
		}
		p := filepath.Join(base, e.Name())
		if st, err := os.Stat(p); err == nil && st.IsDir() { // suit les liens éventuels
			out = append(out, p)
		}
	}
	return out
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// RotatingFile : journal limité en taille (5 Mio, 3 archives) pour ne jamais remplir le volume.
type RotatingFile struct {
	Path     string
	MaxBytes int64
	Keep     int
	mu       sync.Mutex
	f        *os.File
	size     int64
}

// OpenRotating ouvre (ou crée) le journal.
func OpenRotating(path string) (*RotatingFile, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, err
	}
	r := &RotatingFile{Path: path, MaxBytes: 5 << 20, Keep: 3}
	return r, r.open()
}

func (r *RotatingFile) open() error {
	f, err := os.OpenFile(r.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640)
	if err != nil {
		return err
	}
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}
	r.f, r.size = f, st.Size()
	return nil
}

func (r *RotatingFile) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.size+int64(len(p)) > r.MaxBytes {
		r.f.Close()
		for i := r.Keep - 1; i >= 1; i-- {
			os.Rename(fmt.Sprintf("%s.%d", r.Path, i), fmt.Sprintf("%s.%d", r.Path, i+1))
		}
		os.Rename(r.Path, r.Path+".1")
		if err := r.open(); err != nil {
			return 0, err
		}
	}
	n, err := r.f.Write(p)
	r.size += int64(n)
	return n, err
}

// Tail renvoie les dernières lignes du journal courant.
func (r *RotatingFile) Tail(maxBytes int64) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	f, err := os.Open(r.Path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return "", err
	}
	start := max(st.Size()-maxBytes, 0)
	if _, err := f.Seek(start, io.SeekStart); err != nil {
		return "", err
	}
	b, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}
	s := string(b)
	if start > 0 {
		if i := strings.IndexByte(s, '\n'); i >= 0 {
			s = s[i+1:]
		}
	}
	return s, nil
}
