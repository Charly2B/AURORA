package playback

import (
	"strings"
	"testing"

	"aurora/internal/probe"
)

var chrome = Caps{Containers: []string{"mp4", "webm"}, Video: []string{"h264", "vp9", "av1"},
	Audio: []string{"aac", "mp3", "opus", "flac"}, HLS: "mse"}
var safari = Caps{Containers: []string{"mp4", "mov"}, Video: []string{"h264", "hevc"},
	Audio: []string{"aac", "mp3", "ac3", "eac3", "flac"}, HLS: "native"}

func file(container, path, vcodec, profile string, h int, audio ...string) File {
	f := File{Path: path, Container: container, VideoCodec: vcodec, VideoProfile: profile, Height: h}
	for i, a := range audio {
		f.Audio = append(f.Audio, probe.Audio{Index: i, Stream: i + 1, Codec: a, Default: i == 0})
	}
	return f
}

func TestDecide(t *testing.T) {
	mp4 := "mov,mp4,m4a,3gp,3g2,mj2"
	mkv := "matroska,webm"
	cases := []struct {
		name      string
		f         File
		caps      Caps
		opt       Options
		mode      Mode
		copyAudio bool
	}{
		{"mp4 h264 aac -> direct", file(mp4, "a.mp4", "h264", "High", 1080, "aac"), chrome, Options{AudioIndex: -1}, ModeDirect, true},
		{"mkv h264 aac -> remux copie", file(mkv, "a.mkv", "h264", "High", 1080, "aac"), chrome, Options{AudioIndex: -1}, ModeRemux, true},
		{"mkv h264 ac3 chrome -> remux audio AAC", file(mkv, "a.mkv", "h264", "High", 1080, "ac3"), chrome, Options{AudioIndex: -1}, ModeRemux, false},
		{"mkv h264 ac3 safari -> remux copie ac3", file(mkv, "a.mkv", "h264", "High", 1080, "ac3"), safari, Options{AudioIndex: -1}, ModeRemux, true},
		{"mp4 h264 dts -> remux audio", file(mp4, "a.mp4", "h264", "High", 1080, "dts"), safari, Options{AudioIndex: -1}, ModeRemux, false},
		{"mkv hevc chrome sans hevc -> transcode", file(mkv, "a.mkv", "hevc", "Main 10", 1080, "aac"), chrome, Options{AudioIndex: -1}, ModeTranscode, true},
		{"mkv hevc safari -> remux", file(mkv, "a.mkv", "hevc", "Main 10", 2160, "eac3"), safari, Options{AudioIndex: -1}, ModeRemux, true},
		{"h264 10 bits -> transcode", file(mp4, "a.mp4", "h264", "High 10", 1080, "aac"), chrome, Options{AudioIndex: -1}, ModeTranscode, true},
		{"piste audio 2 -> remux", file(mp4, "a.mp4", "h264", "High", 1080, "aac", "aac"), chrome, Options{AudioIndex: 1}, ModeRemux, true},
		{"sous-titre image -> transcode", file(mp4, "a.mp4", "h264", "High", 1080, "aac"), chrome, Options{AudioIndex: -1, BurnSubtitle: true}, ModeTranscode, false},
		{"webm vp9 opus -> direct", file(mkv, "a.webm", "vp9", "", 1080, "opus"), chrome, Options{AudioIndex: -1}, ModeDirect, true},
		{"mpeg2 -> transcode", file("mpegts", "a.ts", "mpeg2video", "Main", 576, "mp2"), chrome, Options{AudioIndex: -1}, ModeTranscode, false},
	}
	for _, c := range cases {
		d := Decide(c.f, c.caps, c.opt)
		if d.Mode != c.mode || d.CopyAudio != c.copyAudio {
			t.Errorf("%s : obtenu %s copyAudio=%v (%s), attendu %s copyAudio=%v", c.name, d.Mode, d.CopyAudio, d.Reason, c.mode, c.copyAudio)
		}
	}
}

func TestTooExpensive(t *testing.T) {
	if bad, _ := TooExpensive(File{Height: 2160, VideoCodec: "hevc"}, Options{}); !bad {
		t.Error("4K doit être refusé par défaut")
	}
	if bad, _ := TooExpensive(File{Height: 2160}, Options{Allow4KEncode: true}); bad {
		t.Error("4K autorisé explicitement")
	}
	if bad, _ := TooExpensive(File{Height: 1080}, Options{}); bad {
		t.Error("1080p doit être accepté")
	}
}

func TestShiftVTT(t *testing.T) {
	in := "WEBVTT\n\n1\n00:00:05.000 --> 00:00:08.000\nAvant\n\n2\n00:01:10.500 --> 00:01:12.000 line:90%\nAprès\n"
	out := string(ShiftVTT([]byte(in), 60))
	if strings.Contains(out, "Avant") {
		t.Errorf("la réplique antérieure au décalage doit disparaître :\n%s", out)
	}
	if !strings.Contains(out, "00:00:10.500 --> 00:00:12.000 line:90%\nAprès") {
		t.Errorf("décalage incorrect :\n%s", out)
	}
	if string(ShiftVTT([]byte(in), 0)) != in {
		t.Error("offset nul : contenu inchangé attendu")
	}
}

func TestSessionFilePath(t *testing.T) {
	s := &Session{Dir: "/tmp/x"}
	for _, bad := range []string{"../etc/passwd", "seg1.m4s", "index.m3u8/../../a", "seg00001.m4s/..", "x.mp4"} {
		if _, _, err := s.FilePath(bad); err == nil {
			t.Errorf("nom accepté à tort : %s", bad)
		}
	}
	if p, n, err := s.FilePath("seg00012.m4s"); err != nil || n != 12 || p != "/tmp/x/seg00012.m4s" {
		t.Errorf("segment valide refusé : %v %d %s", err, n, p)
	}
}
