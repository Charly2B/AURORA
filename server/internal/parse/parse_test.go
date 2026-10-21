package parse

import "testing"

func TestParseMovie(t *testing.T) {
	root := "/volume1/Films"
	cases := []struct {
		path, title string
		year        int
		edition     string
		res, hdr    string
	}{
		{"/volume1/Films/Film.2025.2160p.mkv", "Film", 2025, "", "2160p", ""},
		{"/volume1/Films/Dune.Part.Two.2024.2160p.UHD.BluRay.REMUX.HDR10.HEVC.mkv", "Dune Part Two", 2024, "", "2160p", "HDR10"},
		{"/volume1/Films/Blade Runner 2049 (2017).mkv", "Blade Runner 2049", 2017, "", "", ""},
		{"/volume1/Films/Blade Runner 2049.mkv", "Blade Runner 2049", 0, "", "", ""},
		{"/volume1/Films/1917 (2019)/1917.2019.1080p.WEB-DL.mkv", "1917", 2019, "", "1080p", ""},
		{"/volume1/Films/2012.2009.MULTI.1080p.mkv", "2012", 2009, "", "1080p", ""},
		{"/volume1/Films/The.French.Connection.1971.1080p.mkv", "The French Connection", 1971, "", "1080p", ""},
		{"/volume1/Films/Amélie (2001)/movie.mkv", "Amélie", 2001, "", "", ""},
		{"/volume1/Films/Le Seigneur des anneaux (2001)/Le.Seigneur.des.Anneaux.2001.EXTENDED.MULTI.2160p.DV.mkv", "Le Seigneur des anneaux", 2001, "Version longue", "2160p", "Dolby Vision"},
		{"/volume1/Films/Alien.1979.Directors.Cut.1080p.mkv", "Alien", 1979, "Director's Cut", "1080p", ""},
		{"/volume1/Films/S.W.A.T.2003.720p.mkv", "S.W.A.T", 2003, "", "720p", ""},
		{"/volume1/Films/Inception.FRENCH.mkv", "Inception", 0, "", "", ""},
		{"/volume1/Films/[Group] Akira (1988) [1080p].mkv", "Akira", 1988, "", "1080p", ""},
		{"/volume1/Films/Hevc.Sample.2021.720p.x265.mkv", "Hevc Sample", 2021, "", "720p", ""},
	}
	for _, c := range cases {
		m := ParseMovie(root, c.path)
		if m.Title != c.title || m.Year != c.year || m.Edition != c.edition || m.Tags.Resolution != c.res || m.Tags.HDR != c.hdr {
			t.Errorf("%s\n got  %q %d %q %q %q\n want %q %d %q %q %q", c.path,
				m.Title, m.Year, m.Edition, m.Tags.Resolution, m.Tags.HDR, c.title, c.year, c.edition, c.res, c.hdr)
		}
	}
}

func TestParseEpisode(t *testing.T) {
	root := "/volume1/Series"
	cases := []struct {
		path              string
		series            string
		season, ep, epEnd int
		title             string
		ok                bool
	}{
		{"/volume1/Series/Serie.S01E03.1080p.mkv", "Serie", 1, 3, 0, "", true},
		{"/volume1/Series/Serie.1x03.mkv", "Serie", 1, 3, 0, "", true},
		{"/volume1/Series/The Office (US)/Season 02/The.Office.US.S02E01.The.Dundies.720p.mkv", "The Office (US)", 2, 1, 0, "The Dundies", true},
		{"/volume1/Series/Lost (2004)/Saison 1/Lost - S01E01E02 - Pilot.mkv", "Lost", 1, 1, 2, "Pilot", true},
		{"/volume1/Series/Lost (2004)/Saison 1/Lost.S01E01-E02.mkv", "Lost", 1, 1, 2, "", true},
		{"/volume1/Series/Doctor Who/Specials/Doctor.Who.S00E05.mkv", "Doctor Who", 0, 5, 0, "", true},
		{"/volume1/Series/Kaamelott/Livre I/Episode 12.mkv", "", 0, 0, 0, "", false},
		{"/volume1/Series/Kaamelott/Saison 1/Episode 12 - Le Code.mkv", "Kaamelott", 1, 12, 0, "Le Code", true},
		{"/volume1/Series/Breaking Bad/S05/breaking.bad.s05e16.felina.mkv", "Breaking Bad", 5, 16, 0, "felina", true},
		{"/volume1/Series/random_video_1920x1080.mkv", "", 0, 0, 0, "", false},
	}
	for _, c := range cases {
		e, ok := ParseEpisode(root, c.path)
		if ok != c.ok {
			t.Errorf("%s: ok=%v want %v (%+v)", c.path, ok, c.ok, e)
			continue
		}
		if !ok {
			continue
		}
		if e.Series != c.series || e.Season != c.season || e.Episode != c.ep || e.EpisodeEnd != c.epEnd || e.Title != c.title {
			t.Errorf("%s\n got  %q S%dE%d-%d %q\n want %q S%dE%d-%d %q", c.path,
				e.Series, e.Season, e.Episode, e.EpisodeEnd, e.Title, c.series, c.season, c.ep, c.epEnd, c.title)
		}
	}
}

func TestNormalizeAndSort(t *testing.T) {
	if got := Normalize("Amélie : Le Fabuleux Destin!"); got != "amelie le fabuleux destin" {
		t.Errorf("Normalize = %q", got)
	}
	if got := SortTitle("Les Misérables"); got != "miserables" {
		t.Errorf("SortTitle = %q", got)
	}
	if got := SortTitle("The Matrix"); got != "matrix" {
		t.Errorf("SortTitle = %q", got)
	}
	if got := SortTitle("L'Odyssée"); got != "odyssee" {
		t.Errorf("SortTitle = %q", got)
	}
}

func TestIsVideo(t *testing.T) {
	for p, want := range map[string]bool{
		"/a/Film.mkv": true, "/a/Film.MP4": true, "/a/sample.mkv": false, "/a/Film-sample.mkv": false,
		"/a/._Film.mkv": false, "/a/Film.srt": false, "/a/Film.nfo": false,
	} {
		if IsVideo(p) != want {
			t.Errorf("IsVideo(%s) != %v", p, want)
		}
	}
}

func TestVersionLabel(t *testing.T) {
	m := ParseMovie("/r", "/r/Film.2020.2160p.HDR.BluRay.MULTI.mkv")
	if got := VersionLabel(m.Tags, m.Edition); got != "2160p · HDR · Blu-ray · MULTI" {
		t.Errorf("VersionLabel = %q", got)
	}
}

func TestParseEpisodeInMovies(t *testing.T) {
	root := "/volume1/Films"
	cases := []struct {
		path, series string
		season, ep   int
		ok           bool
	}{
		{"/volume1/Films/Agen kim S01E06.mkv", "Agen kim", 1, 6, true},
		{"/volume1/Films/Kill Jackie - S01E04 - Cocaine Blues.mkv", "Kill Jackie", 1, 4, true},
		{"/volume1/Films/Action/Eureka S04E21 Christmas Special.mkv", "Eureka", 4, 21, true},
		{"/volume1/Films/Sons of Anarchy/Saison 1/S01E04 - FR.mkv", "Sons of Anarchy", 1, 4, true},
		{"/volume1/Films/The.Walking.Dead.Dead.City.S03E07.mkv", "The Walking Dead Dead City", 3, 7, true},
		{"/volume1/Films/Wonder Woman 1984 (2020).mkv", "", 0, 0, false},
		{"/volume1/Films/Blade Runner 2049.mkv", "", 0, 0, false},
		{"/volume1/Films/Saison 1/Episode 12.mkv", "", 0, 0, false}, // numéro trop vague parmi des films
	}
	for _, c := range cases {
		e, ok := ParseEpisodeInMovies(root, c.path)
		if ok != c.ok || (ok && (e.Series != c.series || e.Season != c.season || e.Episode != c.ep)) {
			t.Errorf("%s : %v %q S%dE%d", c.path, ok, e.Series, e.Season, e.Episode)
		}
	}
}
