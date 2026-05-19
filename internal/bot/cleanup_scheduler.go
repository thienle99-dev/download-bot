package bot

import (
	"context"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"
)

// StartCleanupScheduler launches a background goroutine to periodically clean up expired files on VPS.
func (s *BotServer) StartCleanupScheduler(ctx context.Context) {
	interval := time.Duration(s.cfg.CleanupIntervalMinutes) * time.Minute
	if interval <= 0 {
		interval = 60 * time.Minute // Fallback
	}

	maxAge := time.Duration(s.cfg.MaxFileAgeHours) * time.Hour
	if maxAge <= 0 {
		maxAge = 24 * time.Hour // Fallback to 24 hours
	}

	log.Printf("[Cleanup] Starting cleanup scheduler: Check every %v, Max age %v", interval, maxAge)

	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		// Run first check immediately
		s.runCleanup(maxAge)

		for {
			select {
			case <-ctx.Done():
				log.Println("[Cleanup] Stopping cleanup scheduler...")
				return
			case <-ticker.C:
				s.runCleanup(maxAge)
			}
		}
	}()
}

func (s *BotServer) runCleanup(maxAge time.Duration) {
	log.Println("[Cleanup] Running periodic cleanup check...")

	now := time.Now()
	var deletedFilesCount int
	var reclaimedBytes int64

	// Walk download directory recursively to identify files to remove
	err := filepath.WalkDir(s.cfg.DownloadDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		// Check if file is older than maxAge
		age := now.Sub(info.ModTime())
		if age > maxAge {
			size := info.Size()
			log.Printf("[Cleanup] Deleting expired file: %s (Age: %v, Size: %.2f MB)",
				path, age.Round(time.Minute), float64(size)/(1024*1024))

			if err := os.Remove(path); err != nil {
				log.Printf("[Cleanup] Error deleting file %s: %v", path, err)
			} else {
				deletedFilesCount++
				reclaimedBytes += size
			}
		}

		return nil
	})

	if err != nil {
		log.Printf("[Cleanup] Error walking download directory: %v", err)
	}

	if deletedFilesCount > 0 {
		s.LogInfo("Bộ dọn dẹp tự động đã xóa %d file hết hạn, giải phóng %.2f MB dung lượng đĩa.",
			deletedFilesCount, float64(reclaimedBytes)/(1024*1024))
	} else {
		log.Println("[Cleanup] Cleanup complete. No expired files found.")
	}
}
