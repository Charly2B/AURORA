package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
)

// Execer : *sql.DB ou *sql.Tx.
type Execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// ReindexItem met à jour l'entrée plein texte d'un item (titre, titre original, personnes, genres/année).
func ReindexItem(ctx context.Context, q Execer, itemID int64) error {
	var title, original, genres, kind string
	var year sql.NullInt64
	err := q.QueryRowContext(ctx, `SELECT title, original_title, genres, year, kind FROM items WHERE id = ?`, itemID).
		Scan(&title, &original, &genres, &year, &kind)
	if err == sql.ErrNoRows {
		_, err = q.ExecContext(ctx, `DELETE FROM items_fts WHERE rowid = ?`, itemID)
		return err
	}
	if err != nil {
		return err
	}
	var people []string
	rows, err := q.QueryContext(ctx, `SELECT name FROM credits WHERE item_id = ? ORDER BY role, ord LIMIT 30`, itemID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var n string
		if rows.Scan(&n) == nil {
			people = append(people, n)
		}
	}
	rows.Close()
	var g []string
	json.Unmarshal([]byte(genres), &g)
	extra := strings.Join(g, " ")
	if year.Valid {
		extra += " " + strconv.FormatInt(year.Int64, 10)
	}
	if _, err := q.ExecContext(ctx, `DELETE FROM items_fts WHERE rowid = ?`, itemID); err != nil {
		return err
	}
	_, err = q.ExecContext(ctx, `INSERT INTO items_fts(rowid, title, original_title, people, extra) VALUES (?,?,?,?,?)`,
		itemID, title, original, strings.Join(people, " "), extra)
	return err
}
