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
	return db.GetUserHistoryPage(userID, 0, limit)
}

func (db *DB) GetUserHistoryPage(userID int64, offset, limit int) ([]DownloadHistory, error) {
	query := `
	SELECT id, user_id, chat_id, url, platform, title, format, file_size, file_path, file_id, created_at
	FROM download_history
	WHERE user_id = ?
	ORDER BY created_at DESC
	LIMIT ? OFFSET ?
	`
	rows, err := db.Query(query, userID, limit, offset)
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

func (db *DB) GetUserHistoryCount(userID int64) (int, error) {
	query := "SELECT COUNT(id) FROM download_history WHERE user_id = ?"
	var count int
	err := db.QueryRow(query, userID).Scan(&count)
	return count, err
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

// GetAllHistory fetches all download records across all users
func (db *DB) GetAllHistory(limit int) ([]DownloadHistory, error) {
	query := `
	SELECT id, user_id, chat_id, url, platform, title, format, file_size, file_path, file_id, created_at
	FROM download_history
	ORDER BY created_at DESC
	LIMIT ?
	`
	rows, err := db.Query(query, limit)
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
		if parsedTime, err := time.Parse("2006-01-02T15:04:05Z", createdAt); err == nil {
			h.CreatedAt = parsedTime
		} else if parsedTime, err := time.Parse("2006-01-02 15:04:05.999999999-07:00", createdAt); err == nil {
			h.CreatedAt = parsedTime
		} else if parsedTime, err := time.Parse("2006-01-02 15:04:05", createdAt); err == nil {
			h.CreatedAt = parsedTime
		} else {
			h.CreatedAt = time.Now()
		}
		list = append(list, h)
	}
	return list, rows.Err()
}

// GetUsersStats lists users with aggregate download counts
func (db *DB) GetUsersStats() ([]UserStat, error) {
	query := `
	SELECT user_id, chat_id, COUNT(id) as download_count, MAX(created_at) as last_download
	FROM download_history
	GROUP BY user_id, chat_id
	ORDER BY download_count DESC
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []UserStat
	for rows.Next() {
		var s UserStat
		var lastDownloadStr string
		err := rows.Scan(&s.UserID, &s.ChatID, &s.DownloadCount, &lastDownloadStr)
		if err != nil {
			return nil, err
		}
		if parsedTime, err := time.Parse("2006-01-02T15:04:05Z", lastDownloadStr); err == nil {
			s.LastDownload = parsedTime
		} else if parsedTime, err := time.Parse("2006-01-02 15:04:05.999999999-07:00", lastDownloadStr); err == nil {
			s.LastDownload = parsedTime
		} else if parsedTime, err := time.Parse("2006-01-02 15:04:05", lastDownloadStr); err == nil {
			s.LastDownload = parsedTime
		} else {
			s.LastDownload = time.Now()
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

// GetDownloadByID fetches a specific record by its ID
func (db *DB) GetDownloadByID(id int64) (*DownloadHistory, error) {
	query := `
	SELECT id, user_id, chat_id, url, platform, title, format, file_size, file_path, file_id, created_at
	FROM download_history
	WHERE id = ?
	`
	var h DownloadHistory
	var createdAt string
	err := db.QueryRow(query, id).Scan(&h.ID, &h.UserID, &h.ChatID, &h.URL, &h.Platform, &h.Title, &h.Format, &h.FileSize, &h.FilePath, &h.FileID, &createdAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
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

// DeleteDownload deletes a specific record
func (db *DB) DeleteDownload(id int64) error {
	_, err := db.Exec("DELETE FROM download_history WHERE id = ?", id)
	return err
}
