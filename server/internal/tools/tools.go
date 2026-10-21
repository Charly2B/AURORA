// Package tools localise les exécutables externes (ffmpeg, ffprobe) et en lit la version.
package tools

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Tools : chemins résolus des outils multimédias. Une chaîne vide = outil absent.
type Tools struct {
	FFmpeg         string `json:"ffmpeg"`
	FFprobe        string `json:"ffprobe"`
	FFmpegVersion  string `json:"ffmpegVersion"`
	FFprobeVersion string `json:"ffprobeVersion"`
}

// Locate cherche dans l'ordre : variable d'environnement explicite, dossier bin/ livré à côté du binaire
// AURORA (paquet SPK), puis le PATH. Les ffmpeg système de DSM ne sont utilisés qu'en dernier recours :
// leur liste de codecs varie selon les versions de DSM.
func Locate(ffmpegEnv, ffprobeEnv string) Tools {
	exeDir := ""
	if exe, err := os.Executable(); err == nil {
		exeDir = filepath.Dir(exe)
	}
	t := Tools{
		FFmpeg:  find("ffmpeg", ffmpegEnv, exeDir),
		FFprobe: find("ffprobe", ffprobeEnv, exeDir),
	}
	t.FFmpegVersion = version(t.FFmpeg)
	t.FFprobeVersion = version(t.FFprobe)
	return t
}

func find(name, explicit, exeDir string) string {
	if explicit != "" {
		if isExec(explicit) {
			return explicit
		}
		return ""
	}
	if exeDir != "" {
		if p := filepath.Join(exeDir, name); isExec(p) {
			return p
		}
	}
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	return ""
}

func isExec(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir() && st.Mode()&0o111 != 0
}

func version(bin string) string {
	if bin == "" {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, "-version").Output()
	if err != nil {
		return ""
	}
	line, _, _ := strings.Cut(string(out), "\n")
	return strings.TrimSpace(line)
}
