package db

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

// Une base 0.1.8 doit conserver les associations et l'historique lors de l'ajout du secours IMDb.
func TestIMDbMigrationFromPublishedSchema(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "aurora.db")
	old, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { old.Close() })
	if _, err := old.Exec(`CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY, name TEXT NOT NULL, applied_at INTEGER NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range migrations {
		if migration.version > 4 {
			break
		}
		if _, err := old.Exec(migration.sql); err != nil {
			t.Fatal(err)
		}
		if _, err := old.Exec(`INSERT INTO schema_migrations VALUES(?,?,0)`, migration.version, migration.name); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := old.Exec(`INSERT INTO libraries(id,name,kind,path,created_at) VALUES (1,'Films','movies','/videos',0);
		INSERT INTO users(id,username,password_hash,role,created_at) VALUES (1,'admin','','admin',0);
		INSERT INTO items(id,library_id,kind,match_key,title,sort_title,match_state,tmdb_id,metadata_at,added_at,updated_at) VALUES
		 (1,1,'movie','a','Manuel','manuel','locked',603,123,0,0),
		 (2,1,'movie','b','Reconnu','reconnu','matched',604,123,0,0),
		 (3,1,'movie','c','Échec','echec','failed',NULL,123,0,0),
		 (4,1,'series','d','Ambigu','ambigu','ambiguous',NULL,123,0,0),
		 (5,1,'episode','e','Épisode','episode','ambiguous',NULL,123,0,0);
		INSERT INTO media_files(id,library_id,item_id,path,size,mtime,added_at) VALUES (1,1,1,'/videos/film.mkv',99,0,0);
		INSERT INTO user_state(user_id,item_id,position,watched) VALUES (1,1,45,1)`); err != nil {
		t.Fatal(err)
	}
	old.Close()
	d, err := Open(context.Background(), path, filepath.Join(dir, "backups"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	for _, tc := range []struct {
		id    int
		state string
		tmdb  int64
		at    int64
	}{
		{1, "locked", 603, 123}, {2, "matched", 604, 123},
		{3, "local", 0, 0}, {4, "local", 0, 0}, {5, "ambiguous", 0, 123},
	} {
		var state string
		var tmdb, at sql.NullInt64
		var imdb sql.NullString
		if err := d.QueryRow(`SELECT match_state,tmdb_id,metadata_at,imdb_id FROM items WHERE id=?`, tc.id).Scan(&state, &tmdb, &at, &imdb); err != nil {
			t.Fatal(err)
		}
		if state != tc.state || tmdb.Int64 != tc.tmdb || at.Int64 != tc.at || imdb.Valid {
			t.Fatalf("migration item %d : %s %v %v %v", tc.id, state, tmdb, at, imdb)
		}
	}
	var position int
	var watched bool
	if err := d.QueryRow(`SELECT position,watched FROM user_state WHERE user_id=1 AND item_id=1`).Scan(&position, &watched); err != nil || position != 45 || !watched {
		t.Fatalf("historique modifié : %d %v %v", position, watched, err)
	}
	var file string
	if err := d.QueryRow(`SELECT path FROM media_files WHERE id=1 AND item_id=1`).Scan(&file); err != nil || file != "/videos/film.mkv" {
		t.Fatalf("fichier modifié : %s %v", file, err)
	}
	backups, err := d.ListBackups()
	if err != nil || len(backups) == 0 {
		t.Fatalf("sauvegarde de migration absente : %v %v", backups, err)
	}
}
