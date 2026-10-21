package metadata

import (
	"context"
	"strings"
	"testing"
)

func TestChoosePrudent(t *testing.T) {
	dune := []SearchResult{
		{ID: 438631, Title: "Dune", OriginalTitle: "Dune", Year: 2021},
		{ID: 841, Title: "Dune", OriginalTitle: "Dune", Year: 1984},
		{ID: 693134, Title: "Dune : Deuxième partie", OriginalTitle: "Dune: Part Two", Year: 2024},
	}
	cases := []struct {
		name  string
		res   []SearchResult
		title string
		year  int
		id    int
		state string
	}{
		{"titre + année exacts", dune, "Dune", 2021, 438631, "matched"},
		{"titre ambigu sans année : pas de choix arbitraire", dune, "Dune", 0, 0, "ambiguous"},
		{"titre original", dune, "Dune Part Two", 2024, 693134, "matched"},
		{"sortie décalée d'un an", dune, "Dune", 1985, 841, "matched"},
		{"année incohérente", dune, "Dune", 2010, 0, "ambiguous"},
		{"aucun résultat", nil, "Inconnu", 2000, 0, "failed"},
		{"accents et ponctuation ignorés", []SearchResult{{ID: 194, Title: "Le Fabuleux Destin d'Amélie Poulain", Year: 2001}},
			"Le fabuleux destin d amelie poulain", 2001, 194, "matched"},
		{"titre approchant refusé", []SearchResult{{ID: 1, Title: "Alien", Year: 1979}}, "Aliens", 1986, 0, "ambiguous"},
	}
	for _, c := range cases {
		id, state := Choose(c.res, c.title, c.year)
		if id != c.id || state != c.state {
			t.Errorf("%s : obtenu %d/%s, attendu %d/%s", c.name, id, state, c.id, c.state)
		}
	}
}

func TestCertification(t *testing.T) {
	cases := []struct {
		in   map[string]string
		want int
		ok   bool
	}{
		{map[string]string{"FR": "12"}, 12, true},
		{map[string]string{"FR": "U"}, 0, true},
		{map[string]string{"US": "PG-13"}, 13, true},
		{map[string]string{"FR": "16", "US": "PG"}, 16, true},
		{map[string]string{"DE": "12"}, 0, false},
		{map[string]string{}, 0, false},
	}
	for _, c := range cases {
		_, age := certification(c.in)
		if (age != nil) != c.ok || (age != nil && *age != c.want) {
			t.Errorf("%v : âge %v", c.in, age)
		}
	}
}

func TestErrorNeverLeaksKey(t *testing.T) {
	c := NewClient("SECRET-KEY-123", "fr-FR", t.TempDir())
	c.BaseURL = "http://127.0.0.1:1" // port fermé : erreur réseau
	_, err := c.Search(context.Background(), "movie", "Dune", 0)
	if err == nil {
		t.Fatal("erreur réseau attendue")
	}
	if strings.Contains(err.Error(), "SECRET") || strings.Contains(err.Error(), "api_key") {
		t.Fatalf("la clé apparaît dans l'erreur : %v", err)
	}
}
