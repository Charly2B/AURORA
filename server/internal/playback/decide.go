// Package playback choisit le mode de lecture (direct, remux, transcodage) et pilote ffmpeg.
//
// Ordre de préférence : Direct Play (le fichier tel quel, aucune charge serveur) > remux
// (conteneur réécrit, vidéo copiée, audio éventuellement converti — peu coûteux) > transcodage vidéo
// (coûteux sur un Ryzen V1500B sans accélération matérielle : limité et refusé au-delà de 1080p par défaut).
package playback

import (
	"path/filepath"
	"slices"
	"strings"

	"aurora/internal/probe"
)

// Caps : capacités déclarées par le client (détectées dans le navigateur via canPlayType/MediaSource).
type Caps struct {
	Containers []string `json:"containers"` // mp4, webm, mov
	Video      []string `json:"video"`      // h264, hevc, av1, vp9
	Audio      []string `json:"audio"`      // aac, mp3, opus, flac, ac3, eac3
	HLS        string   `json:"hls"`        // native | mse | none
	MaxHeight  int      `json:"maxHeight"`
}

// File : ce que la décision doit savoir d'un fichier.
type File struct {
	Path         string
	Container    string
	VideoCodec   string
	VideoProfile string
	PixFmt       string
	Height       int
	HDR          string
	Audio        []probe.Audio
}

// Mode de lecture.
type Mode string

const (
	ModeDirect    Mode = "direct"
	ModeRemux     Mode = "remux"     // vidéo copiée ; audio copié ou converti en AAC
	ModeTranscode Mode = "transcode" // vidéo réencodée en H.264
)

// Decision : résultat de l'arbitrage.
type Decision struct {
	Mode        Mode   `json:"mode"`
	CopyAudio   bool   `json:"copyAudio"`
	AudioStream int    `json:"-"` // index absolu du flux audio, -1 = aucun
	Reason      string `json:"reason"`
}

// Options d'arbitrage côté serveur.
type Options struct {
	AudioIndex    int  // piste audio choisie (index parmi les pistes audio), -1 = piste par défaut
	BurnSubtitle  bool // sous-titre image à incruster : impose le transcodage
	Allow4KEncode bool // autoriser le transcodage de sources > 1080p (déconseillé sur DS1825+)
}

// ContainerKind ramène le format_name de ffprobe à un conteneur simple.
func ContainerKind(formatName, path string) string {
	f := strings.ToLower(formatName)
	ext := strings.ToLower(filepath.Ext(path))
	switch {
	case strings.Contains(f, "mp4") || strings.Contains(f, "mov"):
		if ext == ".mov" {
			return "mov"
		}
		return "mp4"
	case strings.Contains(f, "matroska") || strings.Contains(f, "webm"):
		if ext == ".webm" {
			return "webm"
		}
		return "mkv"
	case strings.Contains(f, "mpegts"):
		return "ts"
	case strings.Contains(f, "avi"):
		return "avi"
	}
	return f
}

// normalizeVideo : noms ffprobe -> noms de capacités.
func normalizeVideo(c string) string {
	switch c {
	case "h265":
		return "hevc"
	}
	return c
}

// videoDecodable : les navigateurs ne décodent ni le H.264 10 bits (High 10) ni le 4:2:2/4:4:4.
func videoDecodable(f File, caps Caps) bool {
	codec := normalizeVideo(f.VideoCodec)
	if !slices.Contains(caps.Video, codec) {
		return false
	}
	if codec == "h264" {
		p := strings.ToLower(f.VideoProfile)
		if strings.Contains(p, "10") || strings.Contains(p, "4:2:2") || strings.Contains(p, "4:4:4") {
			return false
		}
	}
	if caps.MaxHeight > 0 && f.Height > caps.MaxHeight {
		return false
	}
	return true
}

// SelectAudio renvoie la piste audio choisie (piste demandée, sinon par défaut, sinon la première).
func SelectAudio(f File, idx int) *probe.Audio {
	if idx >= 0 && idx < len(f.Audio) {
		return &f.Audio[idx]
	}
	for i := range f.Audio {
		if f.Audio[i].Default {
			return &f.Audio[i]
		}
	}
	if len(f.Audio) > 0 {
		return &f.Audio[0]
	}
	return nil
}

// firstAudioPlayed : piste qu'un lecteur HTML5 jouera en Direct Play (il ne sait pas choisir).
func firstAudioPlayed(f File) *probe.Audio { return SelectAudio(f, -1) }

// Decide choisit le mode de lecture le moins coûteux compatible avec le client.
func Decide(f File, caps Caps, opt Options) Decision {
	audio := SelectAudio(f, opt.AudioIndex)
	audioStream := -1
	audioOK := true
	if audio != nil {
		audioStream = audio.Stream
		audioOK = slices.Contains(caps.Audio, audio.Codec)
	}
	videoOK := f.VideoCodec != "" && videoDecodable(f, caps)
	container := ContainerKind(f.Container, f.Path)
	containerOK := slices.Contains(caps.Containers, container)
	hlsOK := caps.HLS == "native" || caps.HLS == "mse"

	if opt.BurnSubtitle {
		return Decision{Mode: ModeTranscode, AudioStream: audioStream, CopyAudio: false,
			Reason: "Sous-titre image (PGS/VobSub) incrusté : conversion vidéo nécessaire."}
	}
	// Direct Play : seulement si la piste audio voulue est celle que le navigateur lira d'office.
	defaultAudio := firstAudioPlayed(f)
	sameTrack := audio == nil || defaultAudio == nil || audio.Stream == defaultAudio.Stream
	if videoOK && audioOK && containerOK && sameTrack {
		return Decision{Mode: ModeDirect, AudioStream: audioStream, CopyAudio: true, Reason: "Lecture directe du fichier original."}
	}
	if !hlsOK {
		return Decision{Mode: ModeDirect, AudioStream: audioStream, CopyAudio: true,
			Reason: "Ce navigateur ne prend pas en charge le streaming adaptatif : tentative de lecture directe."}
	}
	// Remux : la vidéo est copiée. L'audio n'est copié que s'il est lisible et transportable en fMP4.
	copyAudio := audioOK && audio != nil && slices.Contains([]string{"aac", "mp3", "ac3", "eac3", "flac", "opus"}, audio.Codec)
	if videoOK {
		var why []string
		if !containerOK {
			why = append(why, "conteneur "+strings.ToUpper(container)+" non pris en charge")
		}
		if !audioOK && audio != nil {
			why = append(why, "audio "+strings.ToUpper(audio.Codec)+" converti en AAC")
		}
		if !sameTrack {
			why = append(why, "piste audio choisie")
		}
		return Decision{Mode: ModeRemux, AudioStream: audioStream, CopyAudio: copyAudio,
			Reason: "Remux sans réencodage vidéo (" + strings.Join(why, ", ") + ")."}
	}
	return Decision{Mode: ModeTranscode, AudioStream: audioStream, CopyAudio: copyAudio,
		Reason: "Codec vidéo " + strings.ToUpper(f.VideoCodec) + " non décodable par ce navigateur : conversion en H.264."}
}

// TooExpensive indique si un transcodage dépasse ce que le NAS peut tenir en temps réel.
// Mesure de référence à faire sur le DS1825+ (voir docs/TESTS.md) ; seuil prudent par défaut.
func TooExpensive(f File, opt Options) (bool, string) {
	if opt.Allow4KEncode {
		return false, ""
	}
	if f.Height > 1088 {
		return true, "Ce fichier " + label4K(f) + " devrait être converti pour ce navigateur, ce qui dépasse les capacités " +
			"de calcul du NAS (pas d'accélération matérielle). Ouvrez-le avec Safari sur Mac, un lecteur compatible " +
			"HEVC, ou choisissez une version 1080p si elle existe."
	}
	return false, ""
}

func label4K(f File) string {
	s := "4K " + strings.ToUpper(f.VideoCodec)
	if f.HDR != "" {
		s += " " + f.HDR
	}
	return s
}
