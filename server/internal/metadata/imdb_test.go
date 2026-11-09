package metadata

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"aurora/internal/db"
)

type imdbTestTransport struct {
	target string
	base   http.RoundTripper
}

func (t imdbTestTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	cp := r.Clone(r.Context())
	u := *r.URL
	if u.Host == "v3.sg.media-imdb.com" || u.Host == "m.media-amazon.com" {
		u.Scheme = "http"
		u.Host = strings.TrimPrefix(t.target, "http://")
	}
	cp.URL = &u
	return t.base.RoundTrip(cp)
}

func imdbClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(h)
	t.Cleanup(server.Close)
	c := NewClient("SECRET-KEY", "fr-FR", t.TempDir())
	c.BaseURL = server.URL + "/3"
	c.HTTP = &http.Client{Transport: imdbTestTransport{server.URL, http.DefaultTransport}}
	return c
}

func imdbDB(t *testing.T, kind string) *db.DB {
	t.Helper()
	dir := t.TempDir()
	d, err := db.Open(context.Background(), filepath.Join(dir, "test.db"), filepath.Join(dir, "backups"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	if _, err := d.Exec(`INSERT INTO libraries(id,name,kind,path,created_at) VALUES (1,'L','movies','/videos',0)`); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Exec(`INSERT INTO items(id,library_id,kind,match_key,title,sort_title,year,added_at,updated_at) VALUES (1,1,?,'original','Mystery Film','mystery film',2020,0,0)`, kind); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestEnrichLibraryIMDbOnly(t *testing.T) {
	for _, tc := range []struct{ kind, qid string }{{"movie", "movie"}, {"series", "tvSeries"}} {
		t.Run(tc.kind, func(t *testing.T) {
			c := imdbClient(t, func(w http.ResponseWriter, r *http.Request) {
				switch {
				case strings.HasPrefix(r.URL.Path, "/3/search/"):
					fmt.Fprint(w, `{"results":[]}`)
				case strings.HasPrefix(r.URL.Path, "/3/find/"):
					fmt.Fprint(w, `{"movie_results":[],"tv_results":[]}`)
				case strings.HasPrefix(r.URL.Path, "/suggestion/x/"):
					if r.URL.Query().Has("api_key") || r.Header.Get("Authorization") != "" {
						t.Error("clé envoyée à IMDb")
					}
					fmt.Fprintf(w, `{"d":[{"id":"tt1234567","l":"Mystery Film","qid":%q,"y":2020}]}`, tc.qid)
				default:
					http.NotFound(w, r)
				}
			})
			d := imdbDB(t, tc.kind)
			e := Enricher{DB: d.DB, Client: func(context.Context) *Client { return c }}
			if err := e.EnrichLibrary(context.Background(), 1, "movies", func(Progress) {}); err != nil {
				t.Fatal(err)
			}
			var title, state string
			if err := d.QueryRow(`SELECT title,match_state FROM items WHERE id=1`).Scan(&title, &state); err != nil {
				t.Fatal(err)
			}
			if title != "Mystery Film" || state != "matched" {
				t.Fatalf("IMDb-only non appliqué : %s / %s", title, state)
			}
		})
	}
}

func TestEnrichLibraryIMDbNearTitleStaysAmbiguous(t *testing.T) {
	c := imdbClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/3/search/"):
			fmt.Fprint(w, `{"results":[]}`)
		case strings.HasPrefix(r.URL.Path, "/3/find/"):
			fmt.Fprint(w, `{}`)
		case strings.HasPrefix(r.URL.Path, "/suggestion/x/"):
			fmt.Fprint(w, `{"d":[{"id":"tt1234567","l":"Mystery Films","qid":"movie","y":2020}]}`)
		default:
			http.NotFound(w, r)
		}
	})
	d := imdbDB(t, "movie")
	e := Enricher{DB: d.DB, Client: func(context.Context) *Client { return c }}
	if err := e.EnrichLibrary(context.Background(), 1, "movies", func(Progress) {}); err != nil {
		t.Fatal(err)
	}
	var title, state string
	d.QueryRow(`SELECT title,match_state FROM items WHERE id=1`).Scan(&title, &state)
	if title != "Mystery Film" || state != "ambiguous" {
		t.Fatalf("titre proche à confirmer : %s / %s", title, state)
	}
}

func TestSearchCandidatesIMDbFiltersAndRanks(t *testing.T) {
	c := imdbClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/3/search/"):
			fmt.Fprint(w, `{"results":[]}`)
		case strings.HasPrefix(r.URL.Path, "/3/find/"):
			fmt.Fprint(w, `{}`)
		case strings.HasPrefix(r.URL.Path, "/suggestion/x/"):
			if r.Header.Get("Authorization") != "" || strings.Contains(r.URL.RawQuery, "SECRET") {
				t.Error("secret TMDB transmis")
			}
			fmt.Fprint(w, `{"d":[{"id":"tt1111111","l":"Other","qid":"movie"},{"id":"tt2222222","l":"Les Visiteurs","qid":"movie","y":1993},{"id":"tt3333333","l":"Les Visiteurs","qid":"tvEpisode"},{"id":"nm2222222","l":"Les Visiteurs"},{"id":"tt4444444","l":"Les Visiteurs","qid":"tvSeries"}]}`)
		default:
			http.NotFound(w, r)
		}
	})
	results, err := c.SearchCandidates(context.Background(), "movie", "Les Visiteur", 1993)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || results[0].IMDbID != "tt2222222" || results[0].ID != 0 {
		t.Fatalf("résultats : %+v", results)
	}
}

func TestSearchCandidatesTMDBPriorityAndYearFallback(t *testing.T) {
	c := imdbClient(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/3/search/") {
			if r.URL.Query().Get("year") != "" {
				fmt.Fprint(w, `{"results":[]}`)
				return
			}
			fmt.Fprint(w, `{"results":[{"id":3,"title":"Dunes","release_date":"1999-01-01"},{"id":2,"title":"Dune","release_date":"2021-01-01"},{"id":1,"title":"Dune","release_date":"1984-01-01"}]}`)
			return
		}
		t.Errorf("IMDb appelé alors que TMDB suffit : %s", r.URL.Path)
		http.Error(w, "unexpected", 500)
	})
	results, err := c.SearchCandidates(context.Background(), "movie", "Dune", 2021)
	if err != nil || len(results) != 3 || results[0].ID != 2 {
		t.Fatalf("priorité TMDB et classement : %+v %v", results, err)
	}
}

func TestIMDbLookupRejectsWrongTypeAndMalformedID(t *testing.T) {
	c := imdbClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"d":[{"id":"tt1234567","l":"A Series","qid":"tvSeries","y":2020}]}`)
	})
	for _, id := range []string{"tt1234567", "tt1234567/../anything", "nm1234567"} {
		if _, err := c.LookupIMDb(context.Background(), "movie", id); !errors.Is(err, ErrIMDbNotFound) {
			t.Fatalf("%s : %v", id, err)
		}
	}
}

func TestApplyIMDbPreservesHistoryAndUnknownClassification(t *testing.T) {
	c := imdbClient(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/3/find/") {
			fmt.Fprint(w, `{}`)
			return
		}
		fmt.Fprint(w, `{"d":[{"id":"tt1234567","l":"Mystery Film","qid":"movie","y":2020}]}`)
	})
	d := imdbDB(t, "movie")
	statements := []string{
		`INSERT INTO items(id,library_id,kind,match_key,title,sort_title,year,imdb_id,added_at,updated_at) VALUES(2,1,'movie','duplicate','Existing','existing',2020,'tt1234567',0,0)`,
		`INSERT INTO users(id,username,password_hash,role,created_at) VALUES(1,'test','x','admin',0)`,
		`INSERT INTO user_state(user_id,item_id,favorite,position,last_played_at) VALUES(1,1,1,42,100)`,
		`INSERT INTO user_state(user_id,item_id,watched,position,last_played_at) VALUES(1,2,1,12,50)`,
		`INSERT INTO media_files(id,library_id,item_id,path,size,mtime,added_at) VALUES(1,1,1,'/videos/file.mkv',100,0,0)`,
		`UPDATE items SET min_age=0,content_rating='Tous publics',rating=9,overview='Wrong previous data' WHERE id=2`,
	}
	for _, s := range statements {
		if _, err := d.Exec(s); err != nil {
			t.Fatal(err)
		}
	}
	e := Enricher{DB: d.DB}
	target, err := e.ApplyIMDb(context.Background(), c, 1, "tt1234567", "locked")
	if err != nil {
		t.Fatal(err)
	}
	if target != 2 {
		t.Fatalf("fusion : %d", target)
	}
	var title, state, imdb, overview string
	var minAge sql.NullInt64
	if err := d.QueryRow(`SELECT title,match_state,imdb_id,overview,min_age FROM items WHERE id=2`).Scan(&title, &state, &imdb, &overview, &minAge); err != nil {
		t.Fatal(err)
	}
	if title != "Mystery Film" || state != "locked" || imdb != "tt1234567" || overview != "" || minAge.Valid {
		t.Fatalf("fiche vérifiée : %s %s %s %s %v", title, state, imdb, overview, minAge)
	}
	var favorite, watched int
	var position float64
	if err := d.QueryRow(`SELECT favorite,watched,position FROM user_state WHERE item_id=2 AND user_id=1`).Scan(&favorite, &watched, &position); err != nil {
		t.Fatal(err)
	}
	if favorite != 1 || watched != 1 || position != 42 {
		t.Fatalf("historique : %d %d %f", favorite, watched, position)
	}
	var fileItem int
	d.QueryRow(`SELECT item_id FROM media_files WHERE id=1`).Scan(&fileItem)
	if fileItem != 2 {
		t.Fatalf("fichier non transféré : %d", fileItem)
	}
}

func TestIMDbImageCachesSafeImageAndRejectsRedirect(t *testing.T) {
	var imageRequests int
	c := imdbClient(t, func(w http.ResponseWriter, r *http.Request) {
		imageRequests++
		if strings.HasSuffix(r.URL.Path, "redirect.jpg") {
			http.Redirect(w, r, "http://127.0.0.1/private", 302)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte{137, 80, 78, 71, 13, 10, 26, 10, 0, 0, 0, 13, 73, 72, 68, 82})
	})
	raw := "https://m.media-amazon.com/images/M/safe.png"
	first, err := c.imdbImage(context.Background(), raw)
	if err != nil || first == "" {
		t.Fatalf("image : %s %v", first, err)
	}
	second, err := c.imdbImage(context.Background(), raw)
	if err != nil || second != first || imageRequests != 1 {
		t.Fatalf("cache : %s %v %d", second, err, imageRequests)
	}
	for _, raw := range []string{"https://evil.example/images/M/a.jpg", "http://m.media-amazon.com/images/M/a.jpg", "https://m.media-amazon.com.evil.example/images/M/a.jpg", "https://m.media-amazon.com/images/M/redirect.jpg"} {
		if _, err := c.imdbImage(context.Background(), raw); err == nil {
			t.Fatalf("image dangereuse acceptée : %s", raw)
		}
	}
}

func TestIMDbNetworkErrorDoesNotBecomeNotFound(t *testing.T) {
	c := imdbClient(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/3/search/") {
			fmt.Fprint(w, `{"results":[]}`)
			return
		}
		http.Error(w, "unavailable", 503)
	})
	if _, err := c.SearchCandidates(context.Background(), "movie", "Unknown", 0); err == nil || errors.Is(err, ErrIMDbNotFound) {
		t.Fatalf("panne masquée : %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.LookupIMDb(ctx, "movie", "tt1234567"); !errors.Is(err, context.Canceled) {
		t.Fatalf("annulation : %v", err)
	}
}

func TestSearchCandidatesTriesAnotherWordAfterTypo(t *testing.T) {
	c := imdbClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/3/search/"):
			if r.URL.Query().Get("query") == "citise" {
				fmt.Fprint(w, `{"results":[]}`)
				return
			}
			if r.URL.Query().Get("query") == "lost" {
				fmt.Fprint(w, `{"results":[{"id":2,"title":"Lost Cities"}]}`)
				return
			}
			fmt.Fprint(w, `{"results":[]}`)
		case strings.HasPrefix(r.URL.Path, "/3/find/"):
			fmt.Fprint(w, `{}`)
		default:
			fmt.Fprint(w, `{"d":[]}`)
		}
	})
	results, err := c.SearchCandidates(context.Background(), "movie", "Lost Citise", 0)
	if err != nil || len(results) == 0 || results[0].ID != 2 {
		t.Fatalf("variante proche écartée : %+v %v", results, err)
	}
}

func TestSearchCandidatesCancellationInterruptsTMDBBackoff(t *testing.T) {
	c := imdbClient(t, func(w http.ResponseWriter, r *http.Request) { http.Error(w, "too many", http.StatusTooManyRequests) })
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, err := c.SearchCandidates(ctx, "movie", "Dune", 0)
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(started) > time.Second {
		t.Fatalf("annulation trop tardive : %v %s", err, time.Since(started))
	}
}

func TestSearchCandidatesIMDbMappingKeepsTMDBDetails(t *testing.T) {
	c := imdbClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/3/search/"):
			fmt.Fprint(w, `{"results":[]}`)
		case strings.HasPrefix(r.URL.Path, "/3/find/"):
			fmt.Fprint(w, `{"movie_results":[{"id":42}]}`)
		case r.URL.Path == "/3/movie/42":
			fmt.Fprint(w, `{"id":42,"title":"Les Visiteurs","original_title":"Les Visiteurs","overview":"Résumé TMDB","release_date":"1993-01-01"}`)
		default:
			fmt.Fprint(w, `{"d":[{"id":"tt1234567","l":"The Visitors","qid":"movie","y":1993}]}`)
		}
	})
	results, err := c.SearchCandidates(context.Background(), "movie", "Visiteur", 0)
	if err != nil || len(results) != 1 || results[0].ID != 42 || results[0].IMDbID != "tt1234567" || results[0].Overview != "Résumé TMDB" {
		t.Fatalf("fiche TMDB liée IMDb : %+v %v", results, err)
	}
}

func TestEnrichLibraryKeepsKnownTMDBAmbiguity(t *testing.T) {
	for _, ambiguous := range []bool{true, false} {
		t.Run(fmt.Sprintf("ambiguous=%v", ambiguous), func(t *testing.T) {
			c := imdbClient(t, func(w http.ResponseWriter, r *http.Request) {
				switch {
				case strings.HasPrefix(r.URL.Path, "/3/search/"):
					if ambiguous {
						fmt.Fprint(w, `{"results":[{"id":10,"title":"Mystery Film","release_date":"2020-01-01"},{"id":20,"title":"Mystery Film","release_date":"2020-01-01"}]}`)
					} else {
						fmt.Fprint(w, `{"results":[{"id":10,"title":"Unrelated Neighbor","release_date":"2020-01-01"}]}`)
					}
				case strings.HasPrefix(r.URL.Path, "/3/find/"):
					fmt.Fprint(w, `{}`)
				default:
					fmt.Fprint(w, `{"d":[{"id":"tt1234567","l":"Mystery Film","qid":"movie","y":2020}]}`)
				}
			})
			d := imdbDB(t, "movie")
			e := Enricher{DB: d.DB, Client: func(context.Context) *Client { return c }}
			if err := e.EnrichLibrary(context.Background(), 1, "movies", func(Progress) {}); err != nil {
				t.Fatal(err)
			}
			var state string
			var imdb sql.NullString
			if err := d.QueryRow(`SELECT match_state,imdb_id FROM items WHERE id=1`).Scan(&state, &imdb); err != nil {
				t.Fatal(err)
			}
			if ambiguous && (state != "ambiguous" || imdb.Valid) {
				t.Fatalf("ambiguïté TMDB écrasée : %s %v", state, imdb)
			}
			if !ambiguous && (state != "matched" || imdb.String != "tt1234567") {
				t.Fatalf("voisin TMDB interdit secours IMDb : %s %v", state, imdb)
			}
		})
	}
}

func TestEnrichLibraryIMDbAliasSurvivesTMDBMapping(t *testing.T) {
	c := imdbClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/3/search/"):
			fmt.Fprint(w, `{"results":[]}`)
		case strings.HasPrefix(r.URL.Path, "/3/find/"):
			fmt.Fprint(w, `{"movie_results":[{"id":42}]}`)
		case r.URL.Path == "/3/movie/42":
			fmt.Fprint(w, `{"id":42,"title":"Les Visiteurs","original_title":"Les Visiteurs","overview":"Résumé","release_date":"1993-01-01"}`)
		default:
			fmt.Fprint(w, `{"d":[{"id":"tt1234567","l":"The Visitors","qid":"movie","y":1993}]}`)
		}
	})
	d := imdbDB(t, "movie")
	if _, err := d.Exec(`UPDATE items SET title='The Visitors',sort_title='the visitors',year=1993 WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	e := Enricher{DB: d.DB, Client: func(context.Context) *Client { return c }}
	if err := e.EnrichLibrary(context.Background(), 1, "movies", func(Progress) {}); err != nil {
		t.Fatal(err)
	}
	var state string
	var tmdb sql.NullInt64
	if err := d.QueryRow(`SELECT match_state,tmdb_id FROM items WHERE id=1`).Scan(&state, &tmdb); err != nil {
		t.Fatal(err)
	}
	if state != "matched" || tmdb.Int64 != 42 {
		t.Fatalf("alias IMDb perdu : %s %v", state, tmdb)
	}
}

func TestApplyIMDbClearsPreviousTMDBDerivedMetadata(t *testing.T) {
	c := imdbClient(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/3/find/") {
			fmt.Fprint(w, `{}`)
			return
		}
		fmt.Fprint(w, `{"d":[{"id":"tt1234567","l":"New Series","qid":"tvSeries","y":2020}]}`)
	})
	d := imdbDB(t, "series")
	statements := []string{
		`UPDATE items SET tmdb_id=99 WHERE id=1`,
		`INSERT INTO seasons(series_id,season,title,overview,poster,air_date) VALUES(1,1,'Old Season','Old Summary','old.jpg','2001-01-01')`,
		`INSERT INTO items(id,library_id,kind,parent_id,match_key,title,sort_title,season,episode,overview,still,air_date,runtime,match_state,added_at,updated_at) VALUES(2,1,'episode',1,'ep2','Old Episode','old episode',1,4,'Old Plot','old.jpg','2001-01-01',45,'matched',0,0)`,
		`INSERT INTO items(id,library_id,kind,parent_id,match_key,title,sort_title,season,episode,overview,match_state,added_at,updated_at) VALUES(3,1,'episode',1,'ep3','Manual Episode','manual episode',1,5,'Manual Plot','locked',0,0)`,
		`UPDATE items SET min_age=0,content_rating='Tous publics',year=2011,still='old.jpg',poster='oldposter.jpg',backdrop='oldback.jpg',runtime=45,air_date='2001-01-01',genres='["Old"]',rating=9,status='Old' WHERE id=3`,
		`INSERT INTO credits(item_id,name,role) VALUES(3,'Old Actor','actor')`,
		`INSERT INTO users(id,username,password_hash,role,created_at) VALUES(1,'test','x','admin',0)`,
		`INSERT INTO user_state(user_id,item_id,position,last_played_at) VALUES(1,2,123,100)`,
		`INSERT INTO media_files(id,library_id,item_id,path,size,mtime,added_at) VALUES(1,1,2,'/videos/episode.mkv',100,0,0)`,
		`INSERT INTO collections(id,name,kind,created_at) VALUES(1,'Old Saga','tmdb',0),(2,'Manual Collection','manual',0)`,
		`INSERT INTO collection_items(collection_id,item_id) VALUES(1,1),(2,1)`,
	}
	for _, s := range statements {
		if _, err := d.Exec(s); err != nil {
			t.Fatal(err)
		}
	}
	e := Enricher{DB: d.DB}
	if _, err := e.ApplyIMDb(context.Background(), c, 1, "tt1234567", "locked"); err != nil {
		t.Fatal(err)
	}
	var title, overview, poster, date string
	if err := d.QueryRow(`SELECT title,overview,poster,air_date FROM seasons WHERE series_id=1 AND season=1`).Scan(&title, &overview, &poster, &date); err != nil {
		t.Fatal(err)
	}
	if title != "" || overview != "" || poster != "" || date != "" {
		t.Fatalf("ancienne saison restante : %q %q %q %q", title, overview, poster, date)
	}
	var runtime sql.NullInt64
	if err := d.QueryRow(`SELECT title,overview,still,air_date,runtime FROM items WHERE id=2`).Scan(&title, &overview, &poster, &date, &runtime); err != nil {
		t.Fatal(err)
	}
	if title != "Épisode 4" || overview != "" || poster != "" || date != "" || runtime.Valid {
		t.Fatalf("ancien épisode restant : %q %q %q %q %v", title, overview, poster, date, runtime)
	}
	if err := d.QueryRow(`SELECT title,overview FROM items WHERE id=3`).Scan(&title, &overview); err != nil {
		t.Fatal(err)
	}
	if title != "Manual Episode" || overview != "Manual Plot" {
		t.Fatalf("correction manuelle perdue : %q %q", title, overview)
	}
	var manualYear int
	var manualAge sql.NullInt64
	var manualClassification string
	if err := d.QueryRow(`SELECT year,min_age,content_rating FROM items WHERE id=3`).Scan(&manualYear, &manualAge, &manualClassification); err != nil {
		t.Fatal(err)
	}
	if manualYear != 2011 || manualAge.Valid || manualClassification != "" {
		t.Fatalf("classification héritée verrouillée : année=%d âge=%v classification=%q", manualYear, manualAge, manualClassification)
	}
	var manualStill, manualPoster, manualBackdrop, manualAir, manualGenres, manualStatus string
	var manualRuntime sql.NullInt64
	var manualRating sql.NullFloat64
	if err := d.QueryRow(`SELECT still,poster,backdrop,air_date,runtime,genres,rating,status FROM items WHERE id=3`).Scan(&manualStill, &manualPoster, &manualBackdrop, &manualAir, &manualRuntime, &manualGenres, &manualRating, &manualStatus); err != nil {
		t.Fatal(err)
	}
	if manualStill != "" || manualPoster != "" || manualBackdrop != "" || manualAir != "" || manualRuntime.Valid || manualGenres != "[]" || manualRating.Valid || manualStatus != "" {
		t.Fatalf("dérivés épisode verrouillé conservés : %q %q %q %q %v %q %v %q", manualStill, manualPoster, manualBackdrop, manualAir, manualRuntime, manualGenres, manualRating, manualStatus)
	}
	var manualCredits int
	if err := d.QueryRow(`SELECT COUNT(*) FROM credits WHERE item_id=3`).Scan(&manualCredits); err != nil || manualCredits != 0 {
		t.Fatalf("ancien générique épisode verrouillé : %d %v", manualCredits, err)
	}
	var position float64
	if err := d.QueryRow(`SELECT position FROM user_state WHERE user_id=1 AND item_id=2`).Scan(&position); err != nil || position != 123 {
		t.Fatalf("historique perdu : %f %v", position, err)
	}
	var itemID int
	if err := d.QueryRow(`SELECT item_id FROM media_files WHERE id=1`).Scan(&itemID); err != nil || itemID != 2 {
		t.Fatalf("fichier perdu : %d %v", itemID, err)
	}
	var count int
	if err := d.QueryRow(`SELECT COUNT(*) FROM collection_items WHERE item_id=1 AND collection_id=1`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("ancienne saga restante : %d %v", count, err)
	}
	if err := d.QueryRow(`SELECT COUNT(*) FROM collection_items WHERE item_id=1 AND collection_id=2`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("collection manuelle perdue : %d %v", count, err)
	}
	if err := d.QueryRow(`SELECT COUNT(*) FROM items_fts WHERE rowid=2 AND title='Épisode 4'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("index épisode périmé : %d %v", count, err)
	}
}

func TestApplyIMDbAutomaticMergePreservesLockedTarget(t *testing.T) {
	for _, mapped := range []bool{false, true} {
		t.Run(fmt.Sprintf("mapped=%v", mapped), func(t *testing.T) {
			c := imdbClient(t, func(w http.ResponseWriter, r *http.Request) {
				switch {
				case strings.HasPrefix(r.URL.Path, "/3/find/"):
					if mapped {
						fmt.Fprint(w, `{"movie_results":[{"id":42}]}`)
					} else {
						fmt.Fprint(w, `{}`)
					}
				case r.URL.Path == "/3/movie/42":
					fmt.Fprint(w, `{"id":42,"title":"Mystery Film","original_title":"Mystery Film","overview":"Metadata","release_date":"2020-01-01"}`)
				default:
					fmt.Fprint(w, `{"d":[{"id":"tt1234567","l":"Mystery Film","qid":"movie","y":2020}]}`)
				}
			})
			d := imdbDB(t, "movie")
			var tmdb any
			if mapped {
				tmdb = 42
			}
			if _, err := d.Exec(`INSERT INTO items(id,library_id,kind,match_key,title,sort_title,year,imdb_id,tmdb_id,match_state,added_at,updated_at) VALUES(2,1,'movie','duplicate','My Manual Title','my manual title',2020,'tt1234567',?,'locked',0,0)`, tmdb); err != nil {
				t.Fatal(err)
			}
			if _, err := d.Exec(`INSERT INTO media_files(id,library_id,item_id,path,size,mtime,added_at) VALUES(1,1,1,'/videos/file.mkv',100,0,0)`); err != nil {
				t.Fatal(err)
			}
			e := Enricher{DB: d.DB}
			target, err := e.ApplyIMDb(context.Background(), c, 1, "tt1234567", "matched")
			if err != nil {
				t.Fatal(err)
			}
			var title, state string
			if err := d.QueryRow(`SELECT title,match_state FROM items WHERE id=?`, target).Scan(&title, &state); err != nil {
				t.Fatal(err)
			}
			if target != 2 || title != "My Manual Title" || state != "locked" {
				t.Fatalf("cible verrouillée remplacée : %d %s %s", target, title, state)
			}
			var fileItem int
			if err := d.QueryRow(`SELECT item_id FROM media_files WHERE id=1`).Scan(&fileItem); err != nil || fileItem != 2 {
				t.Fatalf("fusion fichier : %d %v", fileItem, err)
			}
		})
	}
}
