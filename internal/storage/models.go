package storage

import "time"

type DownloadHistory struct {
	ID        int64     `db:"id"`
	UserID    int64     `db:"user_id"`
	ChatID    int64     `db:"chat_id"`
	URL       string    `db:"url"`
	Platform  string    `db:"platform"` // "youtube", "tiktok"
	Title     string    `db:"title"`
	Format    string    `db:"format"` // "1080p", "720p", "mp3"
	FileSize  int64     `db:"file_size"`
	FilePath  string    `db:"file_path"` // Local path on disk if still cached
	FileID    string    `db:"file_id"`   // Telegram file_id (sent to user)
	CreatedAt time.Time `db:"created_at"`
}

type UserStat struct {
	UserID        int64     `json:"user_id"`
	ChatID        int64     `json:"chat_id"`
	DownloadCount int       `json:"download_count"`
	LastDownload  time.Time `json:"last_download"`
}

type StickerSet struct {
	ID        int64     `db:"id"`
	UserID    int64     `db:"user_id"`
	Name      string    `db:"name"`
	Title     string    `db:"title"`
	CreatedAt time.Time `db:"created_at"`
}
