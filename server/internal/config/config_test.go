package config

import (
	"os"
	"path/filepath"
	"testing"
)

// Disques externes Synology (USB, eSATA) visibles au même titre que les volumes internes.
func TestVolumesIncludeExternalDisks(t *testing.T) {
	base := t.TempDir()
	for _, d := range []string{"volume1", "volume2", "volumeUSB1", "volumeSATA1", "volumes", "volumeX", "tmp"} {
		os.Mkdir(filepath.Join(base, d), 0o755)
	}
	got := map[string]bool{}
	for _, p := range volumesIn(base) {
		got[filepath.Base(p)] = true
	}
	for _, want := range []string{"volume1", "volume2", "volumeUSB1", "volumeSATA1"} {
		if !got[want] {
			t.Errorf("%s absent : %v", want, got)
		}
	}
	for _, no := range []string{"volumes", "volumeX", "tmp"} {
		if got[no] {
			t.Errorf("%s ne doit pas être proposé", no)
		}
	}
}
