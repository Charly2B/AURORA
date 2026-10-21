package playback

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Extractor produit sous-titres WebVTT et miniatures, avec cache disque et concurrence limitée.
type Extractor struct {
	FFmpeg   string
	CacheDir string
	sem      chan struct{}
}

// NewExtractor crée un extracteur (2 opérations ffmpeg simultanées au plus).
func NewExtractor(ffmpeg, cacheDir string) *Extractor {
	os.MkdirAll(filepath.Join(cacheDir, "subs"), 0o750)
	os.MkdirAll(filepath.Join(cacheDir, "thumbs"), 0o750)
	return &Extractor{FFmpeg: ffmpeg, CacheDir: cacheDir, sem: make(chan struct{}, 2)}
}

func (e *Extractor) acquire(ctx context.Context) error {
	select {
	case e.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (e *Extractor) release() { <-e.sem }

// SubtitleVTT renvoie un sous-titre converti en WebVTT. stream >= 0 : piste interne ; sinon externalPath.
// cacheKey doit changer quand le fichier source change (id + mtime).
func (e *Extractor) SubtitleVTT(ctx context.Context, cacheKey, videoPath string, stream int, externalPath string) ([]byte, error) {
	cached := filepath.Join(e.CacheDir, "subs", cacheKey+".vtt")
	if b, err := os.ReadFile(cached); err == nil {
		return b, nil
	}
	if strings.EqualFold(filepath.Ext(externalPath), ".vtt") {
		return os.ReadFile(externalPath)
	}
	if e.FFmpeg == "" {
		return nil, errors.New("ffmpeg introuvable : conversion des sous-titres impossible")
	}
	if err := e.acquire(ctx); err != nil {
		return nil, err
	}
	defer e.release()
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	args := []string{"-hide_banner", "-loglevel", "error", "-nostdin"}
	if stream >= 0 {
		args = append(args, "-i", videoPath, "-map", fmt.Sprintf("0:%d", stream))
	} else {
		args = append(args, "-sub_charenc", "UTF-8", "-i", externalPath)
	}
	args = append(args, "-f", "webvtt", "-")
	var out, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, e.FFmpeg, args...)
	cmd.Stdout, cmd.Stderr = &out, &stderr
	if err := cmd.Run(); err != nil {
		if stream < 0 {
			// Fichiers .srt hérités en Latin-1/Windows-1252 : on réessaie avec ce jeu de caractères.
			out.Reset()
			stderr.Reset()
			args[5] = "CP1252"
			cmd = exec.CommandContext(ctx, e.FFmpeg, args...)
			cmd.Stdout, cmd.Stderr = &out, &stderr
			err = cmd.Run()
		}
		if err != nil {
			return nil, fmt.Errorf("conversion des sous-titres : %s", strings.TrimSpace(stderr.String()))
		}
	}
	tmp := cached + ".tmp"
	if os.WriteFile(tmp, out.Bytes(), 0o640) == nil {
		os.Rename(tmp, cached)
	}
	return out.Bytes(), nil
}

var vttTime = regexp.MustCompile(`^((?:\d+:)?\d{2}:\d{2}\.\d{3}) --> ((?:\d+:)?\d{2}:\d{2}\.\d{3})(.*)$`)

// ShiftVTT décale les horodatages de -offset secondes (lecture HLS démarrée en cours de film).
// Les répliques qui finissent avant 0 sont retirées.
func ShiftVTT(vtt []byte, offset float64) []byte {
	if offset <= 0 {
		return vtt
	}
	var out bytes.Buffer
	sc := bufio.NewScanner(bytes.NewReader(vtt))
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	skipping := false
	for sc.Scan() {
		line := sc.Text()
		if m := vttTime.FindStringSubmatch(line); m != nil {
			start := parseVTTTime(m[1]) - offset
			end := parseVTTTime(m[2]) - offset
			if end <= 0 {
				skipping = true
				continue
			}
			skipping = false
			out.WriteString(formatVTTTime(max(start, 0)) + " --> " + formatVTTTime(end) + m[3] + "\n")
			continue
		}
		if skipping {
			if strings.TrimSpace(line) == "" {
				skipping = false
			}
			continue
		}
		out.WriteString(line + "\n")
	}
	return out.Bytes()
}

func parseVTTTime(s string) float64 {
	parts := strings.Split(s, ":")
	var t float64
	for _, p := range parts {
		v, _ := strconv.ParseFloat(p, 64)
		t = t*60 + v
	}
	return t
}

func formatVTTTime(t float64) string {
	ms := int64(t*1000 + 0.5)
	h := ms / 3600000
	m := (ms / 60000) % 60
	s := (ms / 1000) % 60
	return fmt.Sprintf("%02d:%02d:%02d.%03d", h, m, s, ms%1000)
}

// Thumbnail renvoie (et met en cache) une image extraite de la vidéo, à 15 % de sa durée.
func (e *Extractor) Thumbnail(ctx context.Context, cacheKey, videoPath string, duration float64) (string, error) {
	// « t2- » : nouvelle méthode de choix d'image ; les anciennes vignettes (souvent noires) sont ignorées.
	cached := filepath.Join(e.CacheDir, "thumbs", "t2-"+cacheKey+".jpg")
	if st, err := os.Stat(cached); err == nil && st.Size() > 0 {
		return cached, nil
	}
	if e.FFmpeg == "" {
		return "", errors.New("ffmpeg introuvable")
	}
	if err := e.acquire(ctx); err != nil {
		return "", err
	}
	defer e.release()
	if st, err := os.Stat(cached); err == nil && st.Size() > 0 { // produite pendant l'attente
		return cached, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	at := 30.0
	if duration > 0 {
		at = duration * 0.15
	}
	tmp := cached + ".tmp.jpg"
	cmd := exec.CommandContext(ctx, e.FFmpeg, "-hide_banner", "-loglevel", "error", "-nostdin",
		"-ss", strconv.FormatFloat(at, 'f', 1, 64), "-i", videoPath, "-frames:v", "1",
		// thumbnail : retient l'image la plus représentative parmi 30, ce qui évite les images noires
		// (fondus, scènes de nuit) d'une capture prise au hasard.
		// La réduction se fait avant : thumbnail garde 30 images en mémoire (≈ 10 Mo en 640 px contre
		// ≈ 750 Mo en 4K 10 bits).
		"-vf", "scale=w='min(640,iw)':h=-2,thumbnail=30,format=yuvj420p", "-q:v", "4", "-y", tmp)
	if out, err := cmd.CombinedOutput(); err != nil {
		os.Remove(tmp)
		return "", fmt.Errorf("miniature : %s", strings.TrimSpace(string(out)))
	}
	if err := os.Rename(tmp, cached); err != nil {
		return "", err
	}
	return cached, nil
}

// KeyframeBefore renvoie l'instant de la dernière image clé <= start. En remux (copie vidéo), ffmpeg ne peut
// démarrer que sur une image clé : sans ce calage, l'horloge affichée et les sous-titres seraient décalés
// de plusieurs secondes. Seuls les paquets sont lus (aucun décodage).
func KeyframeBefore(ctx context.Context, ffprobe, path string, start float64) float64 {
	if ffprobe == "" || start <= 0 {
		return start
	}
	for _, window := range []float64{20, 90} {
		from := max(start-window, 0)
		c, cancel := context.WithTimeout(ctx, 15*time.Second)
		out, err := exec.CommandContext(c, ffprobe, "-v", "error", "-select_streams", "v:0",
			"-read_intervals", fmt.Sprintf("%.3f%%%.3f", from, start+0.05),
			"-show_entries", "packet=pts_time,flags", "-of", "csv=p=0", "--", path).Output()
		cancel()
		if err != nil {
			return start
		}
		best := -1.0
		for _, line := range strings.Split(string(out), "\n") {
			ts, flags, ok := strings.Cut(strings.TrimSpace(line), ",")
			if !ok || !strings.Contains(flags, "K") {
				continue
			}
			t, err := strconv.ParseFloat(ts, 64)
			if err == nil && t <= start+0.05 && t > best {
				best = t
			}
		}
		if best >= 0 {
			return best
		}
	}
	return start
}
