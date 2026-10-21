// Package probe analyse un fichier vidéo avec ffprobe (pistes, codecs, HDR, chapitres).
package probe

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Audio décrit une piste audio.
type Audio struct {
	Index    int    `json:"index"`    // index de la piste parmi les pistes audio (0, 1…)
	Stream   int    `json:"stream"`   // index absolu du flux dans le conteneur
	Codec    string `json:"codec"`    // aac, ac3, eac3, dts, truehd, flac, opus, mp3…
	Profile  string `json:"profile"`  // ex. « DTS-HD MA »
	Channels int    `json:"channels"` // 2, 6, 8…
	Layout   string `json:"layout"`
	Language string `json:"language"`
	Title    string `json:"title"`
	Default  bool   `json:"default"`
	Atmos    bool   `json:"atmos"` // indiqué par le profil ou le titre ; non vérifiable autrement
}

// Subtitle décrit une piste de sous-titres interne ou un fichier externe.
type Subtitle struct {
	Index    int    `json:"index"`
	Stream   int    `json:"stream"` // -1 pour un fichier externe
	Codec    string `json:"codec"`  // subrip, ass, webvtt, mov_text, hdmv_pgs_subtitle, dvd_subtitle
	Language string `json:"language"`
	Title    string `json:"title"`
	Default  bool   `json:"default"`
	Forced   bool   `json:"forced"`
	External string `json:"external,omitempty"` // chemin du fichier externe (jamais exposé au client)
	Text     bool   `json:"text"`               // convertible en WebVTT ; false = image (PGS, VobSub)
}

// Chapter décrit un chapitre.
type Chapter struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Title string  `json:"title"`
}

// Result : résultat complet d'une analyse.
type Result struct {
	Container    string     `json:"container"` // format_name ffprobe, ex. « matroska,webm », « mov,mp4,m4a… »
	Duration     float64    `json:"duration"`
	Bitrate      int64      `json:"bitrate"`
	VideoStream  int        `json:"videoStream"`
	VideoCodec   string     `json:"videoCodec"`
	VideoProfile string     `json:"videoProfile"`
	VideoLevel   int        `json:"videoLevel"`
	PixFmt       string     `json:"pixFmt"`
	Width        int        `json:"width"`
	Height       int        `json:"height"`
	FrameRate    float64    `json:"frameRate"`
	HDR          string     `json:"hdr"`
	Audio        []Audio    `json:"audio"`
	Subtitles    []Subtitle `json:"subtitles"`
	Chapters     []Chapter  `json:"chapters"`
}

type ffStream struct {
	Index         int               `json:"index"`
	CodecType     string            `json:"codec_type"`
	CodecName     string            `json:"codec_name"`
	Profile       string            `json:"profile"`
	Level         int               `json:"level"`
	PixFmt        string            `json:"pix_fmt"`
	Width         int               `json:"width"`
	Height        int               `json:"height"`
	AvgFrameRate  string            `json:"avg_frame_rate"`
	Channels      int               `json:"channels"`
	ChannelLayout string            `json:"channel_layout"`
	ColorTransfer string            `json:"color_transfer"`
	ColorPrim     string            `json:"color_primaries"`
	Tags          map[string]string `json:"tags"`
	Disposition   map[string]int    `json:"disposition"`
	SideData      []map[string]any  `json:"side_data_list"`
}

type ffOutput struct {
	Streams  []ffStream `json:"streams"`
	Chapters []struct {
		StartTime string            `json:"start_time"`
		EndTime   string            `json:"end_time"`
		Tags      map[string]string `json:"tags"`
	} `json:"chapters"`
	Format struct {
		FormatName string `json:"format_name"`
		Duration   string `json:"duration"`
		BitRate    string `json:"bit_rate"`
	} `json:"format"`
}

// Run exécute ffprobe sur path. Le délai protège le NAS d'un fichier pathologique.
func Run(ctx context.Context, ffprobe, path string) (*Result, error) {
	if ffprobe == "" {
		return nil, errors.New("ffprobe introuvable")
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, ffprobe, "-v", "error", "-print_format", "json",
		"-show_format", "-show_streams", "-show_chapters", "--", path)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if len(msg) > 300 {
			msg = msg[:300]
		}
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("ffprobe : %s", msg)
	}
	return Parse(stdout.Bytes())
}

// Parse convertit la sortie JSON de ffprobe.
func Parse(data []byte) (*Result, error) {
	var out ffOutput
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("sortie ffprobe illisible : %w", err)
	}
	r := &Result{VideoStream: -1, Audio: []Audio{}, Subtitles: []Subtitle{}, Chapters: []Chapter{}}
	r.Container = out.Format.FormatName
	r.Duration, _ = strconv.ParseFloat(out.Format.Duration, 64)
	r.Bitrate, _ = strconv.ParseInt(out.Format.BitRate, 10, 64)
	for _, s := range out.Streams {
		switch s.CodecType {
		case "video":
			// On ignore les pochettes intégrées (attached_pic) et on garde le premier vrai flux vidéo.
			if s.Disposition["attached_pic"] == 1 || r.VideoStream >= 0 {
				continue
			}
			r.VideoStream = s.Index
			r.VideoCodec = s.CodecName
			r.VideoProfile = s.Profile
			r.VideoLevel = s.Level
			r.PixFmt = s.PixFmt
			r.Width, r.Height = s.Width, s.Height
			r.FrameRate = parseRate(s.AvgFrameRate)
			r.HDR = detectHDR(s)
		case "audio":
			title := s.Tags["title"]
			a := Audio{
				Index: len(r.Audio), Stream: s.Index, Codec: s.CodecName, Profile: s.Profile,
				Channels: s.Channels, Layout: s.ChannelLayout, Language: lang(s.Tags),
				Title: title, Default: s.Disposition["default"] == 1,
			}
			lp, lt := strings.ToLower(s.Profile), strings.ToLower(title)
			a.Atmos = strings.Contains(lp, "atmos") || strings.Contains(lt, "atmos")
			r.Audio = append(r.Audio, a)
		case "subtitle":
			r.Subtitles = append(r.Subtitles, Subtitle{
				Index: len(r.Subtitles), Stream: s.Index, Codec: s.CodecName, Language: lang(s.Tags),
				Title: s.Tags["title"], Default: s.Disposition["default"] == 1,
				Forced: s.Disposition["forced"] == 1, Text: IsTextSubtitle(s.CodecName),
			})
		}
	}
	if r.Duration == 0 && r.VideoStream < 0 {
		return nil, errors.New("aucun flux vidéo ni durée détectés")
	}
	for _, c := range out.Chapters {
		start, _ := strconv.ParseFloat(c.StartTime, 64)
		end, _ := strconv.ParseFloat(c.EndTime, 64)
		r.Chapters = append(r.Chapters, Chapter{Start: start, End: end, Title: c.Tags["title"]})
	}
	return r, nil
}

// IsTextSubtitle : sous-titres textuels convertibles en WebVTT par ffmpeg.
func IsTextSubtitle(codec string) bool {
	switch codec {
	case "subrip", "srt", "ass", "ssa", "webvtt", "mov_text", "text", "microdvd", "subviewer":
		return true
	}
	return false
}

func lang(tags map[string]string) string {
	l := strings.ToLower(tags["language"])
	if l == "und" {
		return ""
	}
	return l
}

func parseRate(s string) float64 {
	num, den, ok := strings.Cut(s, "/")
	if !ok {
		f, _ := strconv.ParseFloat(s, 64)
		return f
	}
	n, _ := strconv.ParseFloat(num, 64)
	d, _ := strconv.ParseFloat(den, 64)
	if d == 0 {
		return 0
	}
	return n / d
}

// detectHDR ne déclare que ce que les métadonnées du flux attestent. HDR10+ (métadonnées dynamiques
// par image) n'est pas visible au niveau du flux : il n'est donc jamais affirmé ici.
func detectHDR(s ffStream) string {
	dv := false
	for _, sd := range s.SideData {
		if t, _ := sd["side_data_type"].(string); strings.Contains(strings.ToLower(t), "dovi") ||
			strings.Contains(strings.ToLower(t), "dolby vision") {
			dv = true
		}
	}
	switch {
	case dv && s.ColorTransfer == "smpte2084":
		return "Dolby Vision + HDR10"
	case dv:
		return "Dolby Vision"
	case s.ColorTransfer == "smpte2084":
		return "HDR10"
	case s.ColorTransfer == "arib-std-b67":
		return "HLG"
	}
	return ""
}
