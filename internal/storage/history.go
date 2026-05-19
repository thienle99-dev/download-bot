package storage

import (
	"database/sql"
	"errors"
	"time"
)

func (db *DB) SaveDownload(h *DownloadHistory) error {
	query := `
	INSERT INTO download_history (user_id, chat_id, url, platform, title, format, file_size, file_path, file_id, created_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	res, err := db.Exec(query, h.UserID, h.ChatID, h.URL, h.Platform, h.Title, h.Format, h.FileSize, h.FilePath, h.FileID, time.Now())
	if err != nil {
		return err
	}

	id, err := res.LastInsertId()
	if err == nil {
		h.ID = id
	}
	return nil
}

func (db *DB) GetUserHistory(userID int64, limit int) ([]DownloadHistory, error) {
	query := `
	SELECT id, user_id, chat_id, url, platform, title, format, file_size, file_path, file_id, created_at
	FROM download_history
	WHERE user_id = ?
	ORDER BY created_at DESC
	LIMIT ?
	`
	rows, err := db.Query(query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []DownloadHistory
	for rows.Next() {
		var h DownloadHistory
		var createdAt string
		err := rows.Scan(&h.ID, &h.UserID, &h.ChatID, &h.URL, &h.Platform, &h.Title, &h.Format, &h.FileSize, &h.FilePath, &h.FileID, &createdAt)
		if err != nil {
			return nil, err
		}
		// Parse SQLite datetime
		if parsedTime, err := time.Parse("2006-01-02T15:04:05Z", createdAt); err == nil {
			h.CreatedAt = parsedTime
		} else if parsedTime, err := time.Parse("2006-01-02 15:04:05.999999999-07:00", createdAt); err == nil {
			h.CreatedAt = parsedTime
		} else if parsedTime, err := time.Parse("2006-01-02 15:04:05", createdAt); err == nil {
			h.CreatedAt = parsedTime
		} else {
			h.CreatedAt = time.Now() // Fallback
		}
		list = append(list, h)
	}
	return list, rows.Err()
}

// GetRecentByURL retrieves the most recent successful download of the same URL by the same user.
// This is useful for reusing cached Telegram file_ids instantly.
func (db *DB) GetRecentByURL(userID int64, url string) (*DownloadHistory, error) {
	query := `
	SELECT id, user_id, chat_id, url, platform, title, format, file_size, file_path, file_id, created_at
	FROM download_history
	WHERE user_id = ? AND url = ?
	ORDER BY created_at DESC
	LIMIT 1
	`
	var h DownloadHistory
	var createdAt string
	err := db.QueryRow(query, userID, url).Scan(&h.ID, &h.UserID, &h.ChatID, &h.URL, &h.Platform, &h.Title, &h.Format, &h.FileSize, &h.FilePath, &h.FileID, &createdAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	// Parse datetime
	if parsedTime, err := time.Parse("2006-01-02T15:04:05Z", createdAt); err == nil {
		h.CreatedAt = parsedTime
	} else if parsedTime, err := time.Parse("2006-01-02 15:04:05.999999999-07:00", createdAt); err == nil {
		h.CreatedAt = parsedTime
	} else if parsedTime, err := time.Parse("2006-01-02 15:04:05", createdAt); err == nil {
		h.CreatedAt = parsedTime
	} else {
		h.CreatedAt = time.Now()
	}
	return &h, nil
}
