// Package db ouvre la base SQLite, applique les migrations versionnées et gère sauvegardes/restaurations.
package db

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite" // pilote SQLite pur Go : binaire statique, aucune dépendance système
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// DB enveloppe *sql.DB et connaît son emplacement disque (pour les sauvegardes).
type DB struct {
	*sql.DB
	Path       string
	BackupsDir string
}

// Open ouvre (ou crée) la base, applique une éventuelle restauration en attente puis les migrations.
// Avant toute migration d'une base existante, une copie de sécurité est créée.
func Open(ctx context.Context, path, backupsDir string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(backupsDir, 0o750); err != nil {
		return nil, err
	}
	if err := applyPendingRestore(path); err != nil {
		return nil, fmt.Errorf("restauration en attente : %w", err)
	}
	// _txlock=immediate : les transactions d'écriture prennent le verrou d'emblée, ce qui évite
	// les « database is locked » en cours de transaction avec plusieurs écrivains.
	dsn := "file:" + path + "?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)" +
		"&_pragma=foreign_keys(1)&_pragma=synchronous(NORMAL)&_txlock=immediate"
	sqldb, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	sqldb.SetMaxOpenConns(8)
	sqldb.SetConnMaxIdleTime(5 * time.Minute)
	d := &DB{DB: sqldb, Path: path, BackupsDir: backupsDir}
	if err := d.PingContext(ctx); err != nil {
		sqldb.Close()
		return nil, err
	}
	if err := d.migrate(ctx); err != nil {
		sqldb.Close()
		return nil, err
	}
	return d, nil
}

type migration struct {
	version int
	name    string
	sql     string
}

func loadMigrations() ([]migration, error) {
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return nil, err
	}
	var out []migration
	for _, e := range entries {
		name := e.Name()
		num, _, ok := strings.Cut(name, "_")
		if !ok {
			return nil, fmt.Errorf("migration mal nommée : %s", name)
		}
		v, err := strconv.Atoi(num)
		if err != nil {
			return nil, fmt.Errorf("migration mal nommée : %s", name)
		}
		b, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return nil, err
		}
		out = append(out, migration{v, name, string(b)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].version < out[j].version })
	return out, nil
}

// SchemaVersion renvoie la dernière migration appliquée.
func (d *DB) SchemaVersion(ctx context.Context) (int, error) {
	var v sql.NullInt64
	err := d.QueryRowContext(ctx, `SELECT MAX(version) FROM schema_migrations`).Scan(&v)
	return int(v.Int64), err
}

// LatestSchemaVersion renvoie la version cible embarquée dans le binaire.
func LatestSchemaVersion() int {
	ms, err := loadMigrations()
	if err != nil || len(ms) == 0 {
		return 0
	}
	return ms[len(ms)-1].version
}

func (d *DB) migrate(ctx context.Context) error {
	if _, err := d.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY, name TEXT NOT NULL, applied_at INTEGER NOT NULL)`); err != nil {
		return err
	}
	current, err := d.SchemaVersion(ctx)
	if err != nil {
		return err
	}
	all, err := loadMigrations()
	if err != nil {
		return err
	}
	var pending []migration
	for _, m := range all {
		if m.version > current {
			pending = append(pending, m)
		}
	}
	if len(pending) == 0 {
		if latest := LatestSchemaVersion(); current > latest {
			// Base créée par une version plus récente d'AURORA : on refuse plutôt que de corrompre.
			return fmt.Errorf("la base est en version %d, ce binaire ne connaît que %d : installez une version plus récente", current, latest)
		}
		return nil
	}
	if current > 0 {
		name := fmt.Sprintf("pre-migration-v%d-%s.db", current, time.Now().Format("20060102-150405"))
		if _, err := d.BackupTo(ctx, name); err != nil {
			return fmt.Errorf("sauvegarde avant migration impossible, migration annulée : %w", err)
		}
	}
	for _, m := range pending {
		tx, err := d.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, m.sql); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %s échouée (base laissée en version %d, sauvegarde dans %s) : %w", m.name, current, d.BackupsDir, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(version, name, applied_at) VALUES (?,?,?)`,
			m.version, m.name, time.Now().Unix()); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		current = m.version
		slog.Info("migration appliquée", "migration", m.name)
	}
	return nil
}

// BackupTo écrit une copie cohérente de la base (VACUUM INTO) dans le dossier de sauvegardes.
func (d *DB) BackupTo(ctx context.Context, name string) (string, error) {
	if name != filepath.Base(name) || !strings.HasSuffix(name, ".db") {
		return "", errors.New("nom de sauvegarde invalide")
	}
	dst := filepath.Join(d.BackupsDir, name)
	if _, err := os.Stat(dst); err == nil {
		return "", fmt.Errorf("la sauvegarde %s existe déjà", name)
	}
	if _, err := d.ExecContext(ctx, `VACUUM INTO ?`, dst); err != nil {
		return "", err
	}
	return dst, nil
}

// BackupInfo décrit une sauvegarde disponible.
type BackupInfo struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	ModTime int64  `json:"modTime"`
}

// ListBackups liste les sauvegardes, plus récentes d'abord.
func (d *DB) ListBackups() ([]BackupInfo, error) {
	entries, err := os.ReadDir(d.BackupsDir)
	if err != nil {
		return nil, err
	}
	out := []BackupInfo{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".db") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, BackupInfo{e.Name(), info.Size(), info.ModTime().Unix()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ModTime > out[j].ModTime })
	return out, nil
}

// pendingRestoreFile : présence = une restauration est demandée au prochain démarrage.
func pendingRestoreFile(dbPath string) string { return dbPath + ".restore" }

// ScheduleRestore programme la restauration d'une sauvegarde au prochain démarrage.
// La base courante n'est jamais effacée : elle est conservée sous un nom horodaté.
func (d *DB) ScheduleRestore(name string) error {
	if name != filepath.Base(name) || !strings.HasSuffix(name, ".db") {
		return errors.New("nom de sauvegarde invalide")
	}
	src := filepath.Join(d.BackupsDir, name)
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("sauvegarde introuvable : %w", err)
	}
	return os.WriteFile(pendingRestoreFile(d.Path), []byte(src), 0o600)
}

func applyPendingRestore(dbPath string) error {
	marker := pendingRestoreFile(dbPath)
	b, err := os.ReadFile(marker)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	src := strings.TrimSpace(string(b))
	if err := checkSQLiteFile(src); err != nil {
		os.Rename(marker, marker+".failed")
		return fmt.Errorf("sauvegarde %s invalide : %w", src, err)
	}
	stamp := time.Now().Format("20060102-150405")
	if _, err := os.Stat(dbPath); err == nil {
		// L'ancienne base est conservée, jamais supprimée.
		if err := os.Rename(dbPath, dbPath+".before-restore-"+stamp); err != nil {
			return err
		}
		for _, suffix := range []string{"-wal", "-shm"} {
			if _, err := os.Stat(dbPath + suffix); err == nil {
				os.Rename(dbPath+suffix, dbPath+".before-restore-"+stamp+suffix)
			}
		}
	}
	if err := copyFile(src, dbPath); err != nil {
		return err
	}
	// Copie brute faite base arrêtée (preupgrade) : son journal WAL contient peut-être les dernières écritures.
	if _, err := os.Stat(src + "-wal"); err == nil {
		if err := copyFile(src+"-wal", dbPath+"-wal"); err != nil {
			return err
		}
	}
	slog.Info("sauvegarde restaurée", "source", src)
	return os.Rename(marker, marker+".done-"+stamp)
}

func checkSQLiteFile(p string) error {
	f, err := os.Open(p)
	if err != nil {
		return err
	}
	defer f.Close()
	head := make([]byte, 16)
	if _, err := f.Read(head); err != nil {
		return err
	}
	if string(head) != "SQLite format 3\x00" {
		return errors.New("ce n'est pas une base SQLite")
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// IntegrityCheck exécute PRAGMA quick_check.
func (d *DB) IntegrityCheck(ctx context.Context) (string, error) {
	var res string
	err := d.QueryRowContext(ctx, `PRAGMA quick_check`).Scan(&res)
	return res, err
}
