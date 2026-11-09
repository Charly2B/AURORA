package metadata

import "testing"

func TestParseLink(t *testing.T) {
	ok := []struct {
		in, kind, imdb string
		id             int
	}{
		{"https://www.themoviedb.org/tv/219937-hunter", "tv", "", 219937},
		{"https://www.themoviedb.org/movie/603-the-matrix?language=fr-FR", "movie", "", 603},
		{"http://themoviedb.org/movie/603", "movie", "", 603},
		{"https://www.themoviedb.org/tv/1399/season/1", "tv", "", 1399},
		{"  https://www.imdb.com/title/tt0133093/  ", "", "tt0133093", 0},
		{"https://m.imdb.com/title/tt0944947/?ref_=fn", "", "tt0944947", 0},
		{"https://www.imdb.com/fr/title/tt12345678/", "", "tt12345678", 0},
		{"imdb.com/title/tt0133093", "", "tt0133093", 0},
		{"www.themoviedb.org/movie/603", "movie", "", 603},
	}
	for _, c := range ok {
		kind, id, imdb, err := ParseLink(c.in)
		if err != nil || kind != c.kind || id != c.id || imdb != c.imdb {
			t.Errorf("%q → %q %d %q %v", c.in, kind, id, imdb, err)
		}
	}
	for _, in := range []string{
		"", "603", "tt0133093", "ftp://themoviedb.org/movie/603", "https://evil.example/movie/603",
		"https://themoviedb.org.evil.com/movie/603", "https://www.themoviedb.org/person/287",
		"https://www.themoviedb.org/movie/0", "https://www.imdb.com/name/nm0000206/", "javascript:alert(1)",
		"https://www.themoviedb.org/movie/99999999999999999999999", "https://imdb.com@evil.com/title/tt0133093",
	} {
		if _, _, _, err := ParseLink(in); err == nil {
			t.Errorf("%q accepté", in)
		}
	}
}
