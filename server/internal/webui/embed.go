// Package webui embarque l'interface compilée (web/dist copié ici par scripts/build.sh).
package webui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// FS renvoie l'interface embarquée, ou nil si le frontend n'a pas été compilé.
func FS() fs.FS {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		return nil
	}
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		return nil
	}
	return sub
}
