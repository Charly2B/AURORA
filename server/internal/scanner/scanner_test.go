package scanner

import (
	"encoding/json"
	"testing"

	"aurora/internal/probe"
)

func TestExternalSubs(t *testing.T) {
	video := "/m/Film (2020)/Film.2020.1080p.mkv"
	cands := []string{
		"/m/Film (2020)/Film.2020.1080p.fr.srt",
		"/m/Film (2020)/Film.2020.1080p.en.forced.srt",
		"/m/Film (2020)/Film.2020.1080p.srt",
		"/m/Film (2020)/Autre.Film.srt",
		"/m/Film (2020)/Film.2020.1080p.eng.sdh.vtt",
	}
	var subs []probe.Subtitle
	if err := json.Unmarshal([]byte(externalSubsJSON(video, cands)), &subs); err != nil {
		t.Fatal(err)
	}
	if len(subs) != 4 {
		t.Fatalf("4 sous-titres attendus (le fichier d'un autre film est exclu) : %+v", subs)
	}
	byPath := map[string]probe.Subtitle{}
	for _, s := range subs {
		byPath[s.External] = s
		if s.Stream != -1 || !s.Text {
			t.Errorf("externe mal décrit : %+v", s)
		}
	}
	if s := byPath[cands[1]]; s.Language != "en" || !s.Forced {
		t.Errorf("forcé anglais : %+v", s)
	}
	if s := byPath[cands[4]]; s.Language != "eng" || s.Title != "SDH" || s.Codec != "vtt" {
		t.Errorf("SDH : %+v", s)
	}
	if externalSubsJSON(video, nil) != "[]" {
		t.Error("aucun sous-titre : tableau vide attendu")
	}
}

func TestSkipDir(t *testing.T) {
	for _, d := range []string{"@eaDir", "#recycle", ".AppleDouble", "Extras", "$RECYCLE.BIN"} {
		if !skipDir(d) {
			t.Errorf("%s doit être ignoré", d)
		}
	}
	for _, d := range []string{"Saison 1", "Films", "Specials"} {
		if skipDir(d) {
			t.Errorf("%s ne doit pas être ignoré", d)
		}
	}
}

// Exclusive réserve la bibliothèque sans tenir le verrou du planificateur pendant fn.
func TestExclusiveReservation(t *testing.T) {
	s := &Scanner{}
	err := s.Exclusive(1, func() error {
		s.Enqueue(1) // ignorée pendant la modification
		if q := s.Queued(); len(q) != 0 {
			t.Errorf("bibliothèque réservée mise en file : %v", q)
		}
		if err := s.Exclusive(1, func() error { return nil }); err != ErrBusy {
			t.Errorf("double réservation acceptée : %v", err)
		}
		_ = s.Current() // ne doit pas bloquer : le verrou n'est pas tenu pendant fn
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if s.reserved[1] {
		t.Fatal("réservation non libérée")
	}
}
