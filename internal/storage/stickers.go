package storage

import (
	"database/sql"
	"errors"
	"time"
)

// SaveStickerSet inserts or ignores a sticker set mapping for a user.
func (db *DB) SaveStickerSet(userID int64, name, title string) error {
	query := `INSERT OR IGNORE INTO sticker_sets (user_id, name, title, created_at) VALUES (?, ?, ?, ?)`
	_, err := db.Exec(query, userID, name, title, time.Now())
	return err
}

// GetUserStickerSets fetches all sticker sets created/registered by a user, ordered by creation date.
func (db *DB) GetUserStickerSets(userID int64) ([]StickerSet, error) {
	query := `SELECT id, user_id, name, title, created_at FROM sticker_sets WHERE user_id = ? ORDER BY created_at DESC`
	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []StickerSet
	for rows.Next() {
		var s StickerSet
		var createdAt string
		if err := rows.Scan(&s.ID, &s.UserID, &s.Name, &s.Title, &createdAt); err != nil {
			return nil, err
		}

		if parsedTime, err := time.Parse("2006-01-02T15:04:05Z", createdAt); err == nil {
			s.CreatedAt = parsedTime
		} else if parsedTime, err := time.Parse("2006-01-02 15:04:05.999999999-07:00", createdAt); err == nil {
			s.CreatedAt = parsedTime
		} else if parsedTime, err := time.Parse("2006-01-02 15:04:05", createdAt); err == nil {
			s.CreatedAt = parsedTime
		} else {
			s.CreatedAt = time.Now()
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

// GetStickerSetByName retrieves a sticker set by its exact unique name.
func (db *DB) GetStickerSetByName(name string) (*StickerSet, error) {
	query := `SELECT id, user_id, name, title, created_at FROM sticker_sets WHERE name = ?`
	var s StickerSet
	var createdAt string
	err := db.QueryRow(query, name).Scan(&s.ID, &s.UserID, &s.Name, &s.Title, &createdAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if parsedTime, err := time.Parse("2006-01-02T15:04:05Z", createdAt); err == nil {
		s.CreatedAt = parsedTime
	} else if parsedTime, err := time.Parse("2006-01-02 15:04:05", createdAt); err == nil {
		s.CreatedAt = parsedTime
	} else {
		s.CreatedAt = time.Now()
	}
	return &s, nil
}
