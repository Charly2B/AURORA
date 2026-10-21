package catalog

import (
	"context"
	"database/sql"
	"time"
)

// WatchedThreshold : une lecture est considérée comme terminée au-delà de 92 % de la durée,
// ou s'il reste moins de 3 minutes (générique de fin).
func WatchedThreshold(position, duration float64) bool {
	if duration <= 0 {
		return false
	}
	return position >= duration*0.92 || (duration > 600 && duration-position < 180)
}

// SaveProgress enregistre la position de lecture. Renvoie true si l'œuvre vient d'être marquée vue.
func (s *Store) SaveProgress(ctx context.Context, userID, itemID, fileID int64, position, duration float64) (bool, error) {
	now := time.Now().Unix()
	if WatchedThreshold(position, duration) {
		_, err := s.DB.ExecContext(ctx, `INSERT INTO user_state(user_id, item_id, position, duration, watched, play_count, last_played_at, last_file_id)
			VALUES (?,?,0,?,1,1,?,?) ON CONFLICT(user_id, item_id) DO UPDATE SET
			play_count = play_count + CASE WHEN position > 0 OR watched = 0 THEN 1 ELSE 0 END,
			position=0, duration=excluded.duration, watched=1, last_played_at=excluded.last_played_at, last_file_id=excluded.last_file_id`,
			userID, itemID, duration, now, fileID)
		return err == nil, err
	}
	if position < 10 {
		// Quelques secondes seulement : on note l'activité sans créer de « reprise » parasite.
		_, err := s.DB.ExecContext(ctx, `INSERT INTO user_state(user_id, item_id, position, duration, last_played_at, last_file_id)
			VALUES (?,?,0,?,?,?) ON CONFLICT(user_id, item_id) DO UPDATE SET duration=excluded.duration,
			last_played_at=excluded.last_played_at, last_file_id=excluded.last_file_id`, userID, itemID, duration, now, fileID)
		return false, err
	}
	_, err := s.DB.ExecContext(ctx, `INSERT INTO user_state(user_id, item_id, position, duration, last_played_at, last_file_id)
		VALUES (?,?,?,?,?,?) ON CONFLICT(user_id, item_id) DO UPDATE SET position=excluded.position, duration=excluded.duration,
		watched=0, last_played_at=excluded.last_played_at, last_file_id=excluded.last_file_id`,
		userID, itemID, position, duration, now, fileID)
	return false, err
}

// SetFlag modifie favori / ma liste. flag ∈ {favorite, watchlist}.
func (s *Store) SetFlag(ctx context.Context, userID, itemID int64, flag string, on bool) error {
	v := 0
	if on {
		v = 1
	}
	switch flag {
	case "favorite":
		_, err := s.DB.ExecContext(ctx, `INSERT INTO user_state(user_id, item_id, favorite) VALUES (?,?,?)
			ON CONFLICT(user_id, item_id) DO UPDATE SET favorite=excluded.favorite`, userID, itemID, v)
		return err
	case "watchlist":
		_, err := s.DB.ExecContext(ctx, `INSERT INTO user_state(user_id, item_id, watchlist, watchlist_at) VALUES (?,?,?,?)
			ON CONFLICT(user_id, item_id) DO UPDATE SET watchlist=excluded.watchlist, watchlist_at=excluded.watchlist_at`,
			userID, itemID, v, time.Now().Unix())
		return err
	}
	return nil
}

// HideFromContinue retire un film ou une série des rangées Continuer / Prochain épisode (un épisode retire sa
// série). Progression et fichiers intacts ; l'œuvre revient dès une nouvelle lecture.
func (s *Store) HideFromContinue(ctx context.Context, userID, itemID int64) error {
	var kind string
	var parent sql.NullInt64
	if err := s.DB.QueryRowContext(ctx, `SELECT kind, parent_id FROM items WHERE id=?`, itemID).Scan(&kind, &parent); err != nil {
		return err
	}
	if kind == "episode" && parent.Valid {
		itemID = parent.Int64
	}
	_, err := s.DB.ExecContext(ctx, `INSERT INTO continue_hidden(user_id, item_id, hidden_at) VALUES (?,?,?)
		ON CONFLICT(user_id, item_id) DO UPDATE SET hidden_at=excluded.hidden_at`, userID, itemID, time.Now().Unix())
	return err
}

// SetWatched marque vu / non vu un film, un épisode, une saison (season >= 0) ou une série entière.
func (s *Store) SetWatched(ctx context.Context, userID, itemID int64, watched bool, season *int) error {
	var kind string
	if err := s.DB.QueryRowContext(ctx, `SELECT kind FROM items WHERE id=?`, itemID).Scan(&kind); err != nil {
		return err
	}
	ids := []int64{itemID}
	if kind == "series" {
		q := `SELECT id FROM items WHERE parent_id=?`
		args := []any{itemID}
		if season != nil {
			q += ` AND season=?`
			args = append(args, *season)
		}
		rows, err := s.DB.QueryContext(ctx, q, args...)
		if err != nil {
			return err
		}
		ids = ids[:0]
		for rows.Next() {
			var id int64
			if rows.Scan(&id) == nil {
				ids = append(ids, id)
			}
		}
		rows.Close()
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().Unix()
	for _, id := range ids {
		if watched {
			_, err = tx.ExecContext(ctx, `INSERT INTO user_state(user_id, item_id, watched, play_count, position, last_played_at) VALUES (?,?,1,1,0,?)
				ON CONFLICT(user_id, item_id) DO UPDATE SET watched=1, position=0,
				play_count = CASE WHEN play_count=0 THEN 1 ELSE play_count END,
				last_played_at = COALESCE(last_played_at, excluded.last_played_at)`, userID, id, now)
		} else {
			_, err = tx.ExecContext(ctx, `UPDATE user_state SET watched=0, position=0 WHERE user_id=? AND item_id=?`, userID, id)
		}
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}
