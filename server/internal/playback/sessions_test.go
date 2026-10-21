package playback

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// Course démarrage/arrêt : des relances rapprochées (recherches successives) ne doivent laisser
// ni ffmpeg orphelin ni dossier temporaire après l'arrêt.
func TestStartStopRaceLeavesNothing(t *testing.T) {
	ff, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg absent")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "src.mp4")
	if out, err := exec.Command(ff, "-hide_banner", "-loglevel", "error", "-y", "-f", "lavfi", "-i", "testsrc2=size=320x240:rate=24:duration=30",
		"-c:v", "libx264", "-preset", "ultrafast", "-g", "48", src).CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	m, err := NewManager(ff, filepath.Join(dir, "work"), Limits{MaxTranscodes: 1, MaxRemux: 4})
	if err != nil {
		t.Fatal(err)
	}
	req := StartRequest{UserID: 1, FileID: 1, Path: src, VideoStream: 0, BurnSubIdx: -1,
		Decision: Decision{Mode: ModeRemux, AudioStream: -1}, File: File{VideoCodec: "h264"}}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r := req
			r.Start = float64(i)
			m.Start(r)
		}(i)
	}
	wg.Wait()
	if tc, rx := m.Count(); tc+rx != 1 {
		t.Fatalf("une seule session doit subsister pour un même utilisateur et un même fichier : %d", tc+rx)
	}
	m.StopAll()
	time.Sleep(300 * time.Millisecond)
	entries, _ := os.ReadDir(m.TmpRoot)
	if len(entries) != 0 {
		t.Fatalf("dossiers temporaires restants : %d", len(entries))
	}
}

func TestPurged(t *testing.T) {
	s := &Session{purged: 5}
	if !s.Purged(4) || s.Purged(5) || s.Purged(-1) {
		t.Fatal("Purged incorrect")
	}
}
