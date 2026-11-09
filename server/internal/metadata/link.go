package metadata

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// Lien collé par l'administrateur quand la recherche TMDB ne trouve pas l'œuvre. Seul l'identifiant est
// extrait : le serveur n'ouvre jamais l'adresse (pas de requête vers un hôte choisi par l'utilisateur).
var (
	tmdbPath = regexp.MustCompile(`^/(movie|tv)/(\d+)(?:[-/?].*)?$`)
	imdbPath = regexp.MustCompile(`^/(?:[a-z]{2}(?:-[a-z]{2})?/)?title/(tt\d{7,10})(?:/.*)?$`)
)

// ErrBadLink : lien non reconnu (message affichable).
var ErrBadLink = errors.New("lien non reconnu : collez l'adresse d'une fiche TMDB (themoviedb.org/movie/… ou /tv/…) ou IMDb (imdb.com/title/tt…)")

// ParseLink lit un lien TMDB ou IMDb. Renvoie soit (kind « movie »/« tv », id TMDB), soit l'identifiant IMDb.
func ParseLink(raw string) (kind string, tmdbID int, imdbID string, err error) {
	raw = strings.TrimSpace(raw)
	if raw != "" && !strings.Contains(raw, "://") {
		raw = "https://" + raw // « imdb.com/title/tt… » collé sans le schéma
	}
	u, perr := url.Parse(raw)
	if perr != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return "", 0, "", ErrBadLink
	}
	host := strings.TrimPrefix(strings.TrimPrefix(strings.ToLower(u.Hostname()), "www."), "m.")
	path := strings.ToLower(u.EscapedPath())
	switch host {
	case "themoviedb.org":
		if m := tmdbPath.FindStringSubmatch(path); m != nil {
			// Erreur testée : un nombre démesuré donnerait sinon la valeur maximale.
			if id, err := strconv.Atoi(m[2]); err == nil && id > 0 {
				return m[1], id, "", nil
			}
		}
	case "imdb.com":
		if m := imdbPath.FindStringSubmatch(path); m != nil {
			return "", 0, m[1], nil
		}
	}
	return "", 0, "", ErrBadLink
}

// FindIMDb convertit un identifiant IMDb en fiche TMDB (film ou série).
func (c *Client) FindIMDb(ctx context.Context, imdbID string) (kind string, id int, err error) {
	if !imdbIDPattern.MatchString(imdbID) {
		return "", 0, ErrIMDbNotFound
	}
	var r struct {
		Movies []struct {
			ID int `json:"id"`
		} `json:"movie_results"`
		TV []struct {
			ID int `json:"id"`
		} `json:"tv_results"`
		Episodes []struct {
			ShowID int `json:"show_id"`
		} `json:"tv_episode_results"`
	}
	if err := c.get(ctx, "/find/"+imdbID, url.Values{"external_source": {"imdb_id"}}, &r); err != nil {
		return "", 0, err
	}
	switch {
	case len(r.Movies) > 0:
		return "movie", r.Movies[0].ID, nil
	case len(r.TV) > 0:
		return "tv", r.TV[0].ID, nil
	case len(r.Episodes) > 0 && r.Episodes[0].ShowID > 0:
		return "tv", r.Episodes[0].ShowID, nil // lien d'un épisode : on propose sa série
	}
	return "", 0, fmt.Errorf("%w : aucune fiche TMDB ne correspond à %s", ErrIMDbNotFound, imdbID)
}

// Lookup renvoie l'aperçu d'une fiche TMDB (mêmes champs que la recherche) pour confirmation.
func (c *Client) Lookup(ctx context.Context, kind string, id int) (*SearchResult, error) {
	var s SearchResult
	var poster string
	switch kind {
	case "movie":
		m, err := c.Movie(ctx, id)
		if err != nil {
			return nil, err
		}
		s = SearchResult{ID: m.ID, Title: m.Title, OriginalTitle: m.OriginalTitle, Year: yearOf(m.ReleaseDate), Overview: m.Overview}
		poster = m.PosterPath
	case "tv":
		t, err := c.TV(ctx, id)
		if err != nil {
			return nil, err
		}
		s = SearchResult{ID: t.ID, Title: t.Name, OriginalTitle: t.OriginalName, Year: yearOf(t.FirstAirDate), Overview: t.Overview}
		poster = t.PosterPath
	default:
		return nil, ErrBadLink
	}
	if poster != "" {
		s.Poster = c.ImageURL + "/w185" + poster
	}
	return &s, nil
}
