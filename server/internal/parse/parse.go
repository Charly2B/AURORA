// Package parse extrait titre, année, saison/épisode et étiquettes de version depuis un chemin de fichier.
// Aucun renommage n'est imposé : on reconnaît les conventions courantes et on laisse le reste « non identifié ».
package parse

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// VideoExtensions : extensions reconnues comme vidéo.
var VideoExtensions = map[string]bool{
	".mkv": true, ".mp4": true, ".m4v": true, ".mov": true, ".avi": true, ".webm": true,
	".ts": true, ".m2ts": true, ".mts": true, ".mpg": true, ".mpeg": true, ".wmv": true, ".flv": true,
	".ogv": true, ".vob": true,
}

// SubtitleExtensions : sous-titres externes (fichiers voisins).
var SubtitleExtensions = map[string]bool{".srt": true, ".vtt": true, ".ass": true, ".ssa": true}

// IsVideo indique si le chemin désigne une vidéo à indexer (exclut les échantillons « sample »).
func IsVideo(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if !VideoExtensions[ext] {
		return false
	}
	base := strings.ToLower(filepath.Base(path))
	if strings.HasPrefix(base, "._") { // métadonnées macOS (AppleDouble)
		return false
	}
	stem := strings.TrimSuffix(base, ext)
	return stem != "sample" && !strings.HasSuffix(stem, "-sample") && !strings.HasSuffix(stem, ".sample")
}

// Movie : résultat d'analyse d'un fichier de film.
type Movie struct {
	Title   string
	Year    int
	Edition string // « Version longue », « Director's Cut »…
	Tags    Tags
}

// Episode : résultat d'analyse d'un fichier d'épisode.
type Episode struct {
	Series     string
	Year       int // année éventuelle dans le nom de la série
	Season     int
	Episode    int
	EpisodeEnd int // >0 pour un épisode double
	Title      string
	Tags       Tags
}

// Tags : informations techniques lisibles dans le nom (servent au libellé de version, pas à la lecture).
type Tags struct {
	Resolution string // 2160p, 1080p, 720p, 480p
	HDR        string // HDR, HDR10, HDR10+, DV
	Source     string // BluRay, WEB-DL, Remux…
	Languages  []string
}

var (
	reYear       = regexp.MustCompile(`(?:19|20)\d{2}`)
	reSxxEyy     = regexp.MustCompile(`(?i)(?:^|[\s.\-_\[(])s(\d{1,2})[\s.\-_]?e(\d{1,4})(?:(?:[\s.\-_]?e|-e?)(\d{1,4}))?(?:$|[\s.\-_\])])`)
	reNxNN       = regexp.MustCompile(`(?i)(?:^|[\s.\-_\[(])(\d{1,2})x(\d{2,3})(?:-(?:\d{1,2}x)?(\d{2,3}))?(?:$|[\s.\-_\])])`)
	reSeasonDir  = regexp.MustCompile(`(?i)^(?:season|saison|staffel|temporada|s)[\s.\-_]*(\d{1,3})$`)
	reSpecials   = regexp.MustCompile(`(?i)^(?:specials?|extras?|sp[ée]ciaux|bonus)$`)
	reEpOnly     = regexp.MustCompile(`(?i)(?:^|[\s.\-_])(?:e|ep|episode|[ée]pisode)[\s.\-_]?(\d{1,4})(?:$|[\s.\-_])`)
	reResolution = regexp.MustCompile(`(?i)\b(2160p|4k|uhd|1080p|1080i|720p|576p|480p)\b`)
	reHDR        = regexp.MustCompile(`(?i)\b(hdr10\+|hdr10plus|hdr10|hdr|dv|dovi|dolby[\s.\-_]?vision)\b`)
	reSource     = regexp.MustCompile(`(?i)\b(remux|blu[\s.\-]?ray|bdrip|brrip|web[\s.\-]?dl|webrip|web|hdtv|dvdrip|dvd)\b`)
	reLang       = regexp.MustCompile(`(?i)\b(multi|vff|vfq|vf2|vfi|vf|vostfr|vo|french|truefrench|english|vost)\b`)
	reBrackets   = regexp.MustCompile(`\[[^\]]*\]|\{[^}]*\}`)
	reSpaces     = regexp.MustCompile(`\s+`)

	editions = []struct {
		re    *regexp.Regexp
		label string
	}{
		{regexp.MustCompile(`(?i)\b(director'?s?[\s.\-_]?cut|dc)\b`), "Director's Cut"},
		{regexp.MustCompile(`(?i)\b(extended|version[\s.\-_]longue|longue)\b`), "Version longue"},
		{regexp.MustCompile(`(?i)\b(theatrical|version[\s.\-_]cin[ée]ma)\b`), "Version cinéma"},
		{regexp.MustCompile(`(?i)\b(unrated|uncut)\b`), "Non censurée"},
		{regexp.MustCompile(`(?i)\b(remastered|remasteris[ée]e?)\b`), "Remasterisée"},
		{regexp.MustCompile(`(?i)\b(imax)\b`), "IMAX"},
		{regexp.MustCompile(`(?i)\b(final[\s.\-_]cut)\b`), "Final Cut"},
	}

	// Marqueurs techniques forts : ils ne figurent quasiment jamais dans un titre et marquent sa fin.
	reJunkStart = regexp.MustCompile(`(?i)(?:^|[\s.\-_\[(])(2160p|4k|uhd|1080p|1080i|720p|576p|480p|remux|blu[\s.\-]?ray|bdrip|brrip|web[\s.\-]?dl|webrip|hdtv|dvdrip|x264|x265|h\.?264|h\.?265|hevc|avc|av1|10bit|hdr10\+?|hdr|dovi)(?:$|[\s.\-_\])])`)
	// Marqueurs faibles (langue, édition…) : retirés seulement en fin de titre, car « The French Connection »
	// ou « Extended Family » sont des titres légitimes.
	reWeakTail = regexp.MustCompile(`(?i)[\s.\-_]+(multi|vff|vfq|vf2|vfi|vf|vostfr|vost|vo|truefrench|french|english|proper|repack|internal|extended|unrated|uncut|remastered|imax|theatrical|director'?s[\s.\-_]?cut|dc|version[\s.\-_]longue)$`)
)

// cleanSeparators remplace les séparateurs de release (. _) par des espaces, sauf dans les sigles (S.W.A.T.).
func cleanSeparators(s string) string {
	s = reBrackets.ReplaceAllString(s, " ")
	// Conserver les sigles du type « S.W.A.T » : lettres isolées séparées par des points.
	var b strings.Builder
	rs := []rune(s)
	for i, r := range rs {
		if r == '_' {
			b.WriteRune(' ')
			continue
		}
		if r == '.' {
			prevLetter := i > 0 && unicode.IsLetter(rs[i-1]) && (i == 1 || !unicode.IsLetter(rs[i-2]))
			nextLetter := i+1 < len(rs) && unicode.IsLetter(rs[i+1]) && (i+2 >= len(rs) || !unicode.IsLetter(rs[i+2]))
			if prevLetter && nextLetter {
				b.WriteRune('.')
				continue
			}
			b.WriteRune(' ')
			continue
		}
		b.WriteRune(r)
	}
	out := reSpaces.ReplaceAllString(b.String(), " ")
	return strings.Trim(out, " -–—.([")
}

func stem(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// ParseTags lit les étiquettes techniques d'un nom.
func ParseTags(name string) Tags {
	var t Tags
	n := strings.NewReplacer(".", " ", "_", " ").Replace(name)
	if m := reResolution.FindStringSubmatch(n); m != nil {
		switch strings.ToLower(m[1]) {
		case "2160p", "4k", "uhd":
			t.Resolution = "2160p"
		case "1080i":
			t.Resolution = "1080p"
		default:
			t.Resolution = strings.ToLower(m[1])
		}
	}
	if m := reHDR.FindStringSubmatch(n); m != nil {
		switch v := strings.ToLower(strings.NewReplacer(" ", "", "-", "", ".", "").Replace(m[1])); v {
		case "hdr10+", "hdr10plus":
			t.HDR = "HDR10+"
		case "dv", "dovi", "dolbyvision":
			t.HDR = "Dolby Vision"
		case "hdr10":
			t.HDR = "HDR10"
		default:
			t.HDR = "HDR"
		}
	}
	if m := reSource.FindStringSubmatch(n); m != nil {
		v := strings.ToLower(strings.NewReplacer(" ", "", "-", "", ".", "").Replace(m[1]))
		switch {
		case v == "remux":
			t.Source = "Remux"
		case strings.Contains(v, "bluray") || v == "bdrip" || v == "brrip":
			t.Source = "Blu-ray"
		case strings.HasPrefix(v, "web"):
			t.Source = "WEB"
		case v == "hdtv":
			t.Source = "HDTV"
		default:
			t.Source = "DVD"
		}
	}
	seen := map[string]bool{}
	for _, m := range reLang.FindAllStringSubmatch(n, -1) {
		l := strings.ToUpper(m[1])
		switch l {
		case "TRUEFRENCH", "FRENCH", "VF", "VF2", "VFI":
			l = "VFF"
		case "VOST":
			l = "VOSTFR"
		case "ENGLISH":
			l = "VO"
		}
		if !seen[l] {
			seen[l] = true
			t.Languages = append(t.Languages, l)
		}
	}
	return t
}

// Edition reconnaît une édition particulière (version longue…).
func Edition(name string) string {
	n := strings.NewReplacer(".", " ", "_", " ").Replace(name)
	for _, e := range editions {
		if e.re.MatchString(n) {
			return e.label
		}
	}
	return ""
}

// ParseMovie analyse un fichier de film situé sous root. Un dossier parent « Titre (Année) » est préféré
// au nom du fichier (convention répandue, plus fiable qu'un nom de release), sauf si les années divergent.
func ParseMovie(root, path string) Movie {
	name := stem(path)
	m := parseTitleYear(name)
	dir := filepath.Dir(path)
	if filepath.Clean(dir) != filepath.Clean(root) {
		pm := parseTitleYear(filepath.Base(dir))
		switch {
		case pm.Year != 0 && pm.Title != "" && (m.Year == 0 || m.Year == pm.Year):
			m = pm
		case m.Title == "" && pm.Title != "":
			m = pm
		}
	}
	return Movie{Title: m.Title, Year: m.Year, Edition: Edition(name), Tags: ParseTags(name)}
}

type titleYear struct {
	Title string
	Year  int
}

// parseTitleYear coupe le nom au premier marqueur technique fort, puis prend la dernière année plausible.
func parseTitleYear(name string) titleYear {
	head := name
	// Premier marqueur technique qui n'est pas en tête : un titre ne peut pas être vide (« HEVC Sample »).
	for _, loc := range reJunkStart.FindAllStringIndex(head, -1) {
		if loc[0] > 0 {
			head = head[:loc[0]]
			break
		}
	}
	year := 0
	maxYear := time.Now().Year() + 1
	// Dernière année plausible : « Blade Runner 2049 (2017) » garde 2017 ; « Blade Runner 2049 » n'a pas d'année.
	locs := reYear.FindAllStringIndex(head, -1)
	for i := len(locs) - 1; i >= 0; i-- {
		start, end := locs[i][0], locs[i][1]
		if !isSep(head, start-1) || !isSep(head, end) {
			continue // fait partie d'un mot ou d'un nombre plus long
		}
		if start == 0 {
			continue // une année en tout début est le titre (« 1917 », « 2012 »)
		}
		y, _ := strconv.Atoi(head[start:end])
		if y >= 1900 && y <= maxYear {
			year = y
			head = head[:start]
			break
		}
	}
	for {
		head = strings.TrimRight(head, " ._-([")
		loc := reWeakTail.FindStringIndex(head)
		if loc == nil || loc[0] == 0 {
			break
		}
		head = head[:loc[0]]
	}
	return titleYear{Title: cleanSeparators(head), Year: year}
}

// ParseEpisode analyse un fichier d'épisode. root est la racine de la bibliothèque : le premier
// dossier sous la racine est le nom de la série le plus fiable. ok=false si aucun numéro n'est reconnu.
func ParseEpisode(root, path string) (Episode, bool) {
	return parseEpisode(root, path, false)
}

// ParseEpisodeInMovies reconnaît un épisode rangé parmi des films : seulement si le nom du fichier porte
// un numéro explicite (S01E06, 1x06). Le nom de la série vient du nom du fichier (ou de son dossier,
// en ignorant « Saison N »), jamais du premier dossier de la bibliothèque, qui peut être une collection.
func ParseEpisodeInMovies(root, path string) (Episode, bool) {
	return parseEpisode(root, path, true)
}

func parseEpisode(root, path string, inMovies bool) (Episode, bool) {
	name := stem(path)
	ep := Episode{Tags: ParseTags(name)}
	found := false
	var titleStart int
	var seriesPart string

	if m := reSxxEyy.FindStringSubmatchIndex(name); m != nil {
		ep.Season, _ = strconv.Atoi(name[m[2]:m[3]])
		ep.Episode, _ = strconv.Atoi(name[m[4]:m[5]])
		if m[6] >= 0 {
			ep.EpisodeEnd, _ = strconv.Atoi(name[m[6]:m[7]])
		}
		seriesPart, titleStart, found = name[:m[0]], m[1], true
	} else if m := reNxNN.FindStringSubmatchIndex(name); m != nil && !isResolutionLike(name[m[0]:m[1]]) {
		ep.Season, _ = strconv.Atoi(name[m[2]:m[3]])
		ep.Episode, _ = strconv.Atoi(name[m[4]:m[5]])
		if m[6] >= 0 {
			ep.EpisodeEnd, _ = strconv.Atoi(name[m[6]:m[7]])
		}
		seriesPart, titleStart, found = name[:m[0]], m[1], true
	}

	// Dossier de saison (« Season 2 », « Saison 02 », « Specials »).
	dirSeason := -1
	dir := filepath.Base(filepath.Dir(path))
	if m := reSeasonDir.FindStringSubmatch(dir); m != nil {
		dirSeason, _ = strconv.Atoi(m[1])
	} else if reSpecials.MatchString(dir) {
		dirSeason = 0
	}
	if !found && dirSeason >= 0 && !inMovies {
		if m := reEpOnly.FindStringSubmatchIndex(name); m != nil {
			ep.Season = dirSeason
			ep.Episode, _ = strconv.Atoi(name[m[2]:m[3]])
			seriesPart, titleStart, found = name[:m[0]], m[1], true
		}
	}
	if !found {
		return ep, false
	}
	if ep.EpisodeEnd != 0 && ep.EpisodeEnd <= ep.Episode {
		ep.EpisodeEnd = 0
	}

	// Titre de l'épisode : ce qui suit le numéro, avant les marqueurs techniques.
	if titleStart < len(name) {
		rest := strings.TrimLeft(name[titleStart:], " .-_")
		// Après le numéro, un marqueur technique immédiat signifie « pas de titre » (S01E03.1080p).
		if loc := reJunkStart.FindStringIndex(rest); loc == nil || loc[0] > 0 {
			ep.Title = parseTitleYear(rest).Title
		}
	}

	if inMovies {
		ty := parseTitleYear(seriesPart)
		ep.Series, ep.Year = ty.Title, ty.Year
		if ep.Series == "" { // « S01E04 - Titre.mkv » : nom de la série dans le dossier
			dir := filepath.Dir(path)
			if reSeasonDir.MatchString(filepath.Base(dir)) || reSpecials.MatchString(filepath.Base(dir)) {
				dir = filepath.Dir(dir)
			}
			if filepath.Clean(dir) != filepath.Clean(root) {
				ty = parseTitleYear(filepath.Base(dir))
				ep.Series, ep.Year = ty.Title, ty.Year
			}
		}
		return ep, ep.Series != ""
	}

	// Nom de la série : premier dossier sous la racine, sinon la partie avant le numéro.
	if top := topFolder(root, path); top != "" && !reSeasonDir.MatchString(top) {
		ty := parseTitleYear(top)
		ep.Series, ep.Year = ty.Title, ty.Year
	}
	if ep.Series == "" {
		ty := parseTitleYear(seriesPart)
		ep.Series, ep.Year = ty.Title, ty.Year
	}
	return ep, ep.Series != ""
}

// isSep : vrai si la position i est hors de la chaîne ou sur un séparateur.
func isSep(s string, i int) bool {
	if i < 0 || i >= len(s) {
		return true
	}
	return strings.IndexByte(" ._-()[]", s[i]) >= 0
}

func isResolutionLike(s string) bool {
	s = strings.ToLower(s)
	return strings.Contains(s, "1920x") || strings.Contains(s, "1280x") || strings.Contains(s, "3840x")
}

// topFolder renvoie le premier composant de path relatif à root, si path est dans un sous-dossier.
func topFolder(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return ""
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) < 2 {
		return ""
	}
	return parts[0]
}

// Normalize produit une clé de comparaison : minuscules, sans accents ni ponctuation, articles retirés.
func Normalize(s string) string {
	s = RemoveAccents(strings.ToLower(s))
	var b strings.Builder
	space := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			space = false
		} else if r == '&' {
			b.WriteString(" and ")
			space = false
		} else if !space {
			b.WriteRune(' ')
			space = true
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// RemoveAccents retire les diacritiques (é -> e).
func RemoveAccents(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	out, _, err := transform.String(t, s)
	if err != nil {
		return s
	}
	return out
}

// SortTitle : titre de tri, sans article initial (français/anglais) et sans accents.
func SortTitle(title string) string {
	t := strings.TrimSpace(title)
	lower := strings.ToLower(t)
	for _, art := range []string{"the ", "a ", "an ", "le ", "la ", "les ", "l'", "l’", "un ", "une ", "des "} {
		if strings.HasPrefix(lower, art) && len(t) > len(art) {
			t = t[len(art):]
			break
		}
	}
	return Normalize(t)
}

// VersionLabel construit un libellé lisible pour distinguer plusieurs fichiers d'une même œuvre.
func VersionLabel(t Tags, edition string) string {
	var parts []string
	if t.Resolution != "" {
		parts = append(parts, t.Resolution)
	}
	if t.HDR != "" {
		parts = append(parts, t.HDR)
	}
	if t.Source != "" {
		parts = append(parts, t.Source)
	}
	if edition != "" {
		parts = append(parts, edition)
	}
	parts = append(parts, t.Languages...)
	return strings.Join(parts, " · ")
}
