package db

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenMigrateBackupRestore(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	path := filepath.Join(dir, "aurora.db")
	d, err := Open(ctx, path, filepath.Join(dir, "backups"))
	if err != nil {
		t.Fatal(err)
	}
	v, _ := d.SchemaVersion(ctx)
	if v != LatestSchemaVersion() || v == 0 {
		t.Fatalf("version de schéma %d, attendu %d", v, LatestSchemaVersion())
	}
	if res, err := d.IntegrityCheck(ctx); err != nil || res != "ok" {
		t.Fatalf("intégrité : %s %v", res, err)
	}
	// FTS5 avec suppression des accents disponible.
	if _, err := d.ExecContext(ctx, `INSERT INTO items_fts(rowid, title, original_title, people, extra) VALUES (1, 'Amélie', '', '', '')`); err != nil {
		t.Fatal(err)
	}
	var n int
	d.QueryRowContext(ctx, `SELECT COUNT(*) FROM items_fts WHERE items_fts MATCH '"amel"*'`).Scan(&n)
	if n != 1 {
		t.Fatal("la recherche plein texte doit ignorer les accents")
	}
	d.ExecContext(ctx, `INSERT INTO settings(key, value) VALUES ('marker', 'avant')`)
	if _, err := d.BackupTo(ctx, "test.db"); err != nil {
		t.Fatal(err)
	}
	if _, err := d.BackupTo(ctx, "../evil.db"); err == nil {
		t.Fatal("nom de sauvegarde avec chemin accepté")
	}
	d.ExecContext(ctx, `UPDATE settings SET value='après' WHERE key='marker'`)
	if err := d.ScheduleRestore("test.db"); err != nil {
		t.Fatal(err)
	}
	d.Close()

	// Redémarrage : la sauvegarde est restaurée, l'ancienne base est conservée.
	d, err = Open(ctx, path, filepath.Join(dir, "backups"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	var val string
	d.QueryRowContext(ctx, `SELECT value FROM settings WHERE key='marker'`).Scan(&val)
	if val != "avant" {
		t.Fatalf("restauration non appliquée : %q", val)
	}
	kept, _ := filepath.Glob(path + ".before-restore-*")
	if len(kept) == 0 {
		t.Fatal("l'ancienne base doit être conservée")
	}
	if _, err := os.Stat(path + ".restore"); err == nil {
		t.Fatal("le marqueur de restauration doit être consommé")
	}
}

func TestRefuseNewerSchema(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	path := filepath.Join(dir, "aurora.db")
	d, err := Open(ctx, path, filepath.Join(dir, "b"))
	if err != nil {
		t.Fatal(err)
	}
	d.ExecContext(ctx, `INSERT INTO schema_migrations(version, name, applied_at) VALUES (9999, 'futur', 0)`)
	d.Close()
	if _, err := Open(ctx, path, filepath.Join(dir, "b")); err == nil {
		t.Fatal("une base plus récente que le binaire doit être refusée")
	}
}
