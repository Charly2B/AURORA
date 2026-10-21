package api

import (
	"os"
	"path/filepath"
	"testing"

	"aurora/internal/probe"
)

// Régression : la clé de cache dépend de la source du sous-titre, pas de sa position dans la liste.
func TestSubtitleCacheKey(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "Film.fr.srt"), filepath.Join(dir, "Film.en.srt")
	os.WriteFile(a, []byte("1"), 0o644)
	os.WriteFile(b, []byte("2"), 0o644)
	m := &mediaRow{ID: 1, Mtime: 10, Path: filepath.Join(dir, "Film.mkv")}
	ka := subtitleCacheKey(m, probe.Subtitle{Index: 0, Stream: -1, External: a})
	kb := subtitleCacheKey(m, probe.Subtitle{Index: 0, Stream: -1, External: b}) // même position, autre fichier
	if ka == kb {
		t.Fatal("deux fichiers à la même position ne doivent pas partager le cache")
	}
	if ka != subtitleCacheKey(m, probe.Subtitle{Index: 3, Stream: -1, External: a}) {
		t.Fatal("la position ne doit pas influer sur la clé")
	}
	os.WriteFile(a, []byte("modifié"), 0o644)
	if ka == subtitleCacheKey(m, probe.Subtitle{Index: 0, Stream: -1, External: a}) {
		t.Fatal("un fichier modifié doit invalider le cache")
	}
	if subtitleCacheKey(m, probe.Subtitle{Stream: 2}) == subtitleCacheKey(m, probe.Subtitle{Stream: 3}) {
		t.Fatal("deux pistes internes différentes doivent avoir des clés différentes")
	}
}
