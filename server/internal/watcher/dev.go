package watcher

import (
	"os"
	"syscall"
)

// deviceOf : identifiant du périphérique (montage) d'un fichier ; 0 si inconnu.
func deviceOf(st os.FileInfo) uint64 {
	if s, ok := st.Sys().(*syscall.Stat_t); ok {
		return uint64(s.Dev) //nolint:unconvert // int32 sous macOS, uint64 sous Linux
	}
	return 0
}
