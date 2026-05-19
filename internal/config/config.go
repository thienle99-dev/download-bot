package config

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	BotToken      string
	APIURL        string // Telegram local API URL if self-hosted
	DownloadDir   string
	CacheDir      string
	DBPath        string
	MaxConcurrent int
	PublicURL     string // URL to serve files (e.g. http://vps-ip:8080)
	ServerPort    string // Local port to run HTTP file server (e.g. :8080)
	AdminPassword          string // Dashboard access password
	AdminTelegramID        int64  // Telegram User ID of the administrator
	CleanupIntervalMinutes int    // Cleanup schedule check interval in minutes
	MaxFileAgeHours        int    // Maximum file age in hours before cleanup
}

func Load() *Config {
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Println("WARNING: BOT_TOKEN is empty! Please set it in your environment.")
	}

	apiURL := os.Getenv("API_URL") // If empty, the client will default to official server

	downloadDir := os.Getenv("DOWNLOAD_DIR")
	if downloadDir == "" {
		downloadDir = "./downloads"
	}

	cacheDir := os.Getenv("CACHE_DIR")
	if cacheDir == "" {
		cacheDir = "./cache"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./bot.db"
	}

	publicURL := os.Getenv("PUBLIC_URL")
	if publicURL == "" {
		publicURL = "http://localhost:8080"
	}

	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = ":8080"
	}

	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminPassword == "" {
		adminPassword = "admin123"
	}

	maxConcurrentStr := os.Getenv("MAX_CONCURRENT")
	maxConcurrent := 3
	if maxConcurrentStr != "" {
		if val, err := strconv.Atoi(maxConcurrentStr); err == nil && val > 0 {
			maxConcurrent = val
		}
	}

	// Create directories if they do not exist
	for _, dir := range []string{downloadDir, cacheDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Printf("Failed to create directory %s: %v", dir, err)
		}
	}

	adminTelegramIDStr := os.Getenv("ADMIN_TELEGRAM_ID")
	var adminTelegramID int64
	if adminTelegramIDStr != "" {
		if val, err := strconv.ParseInt(adminTelegramIDStr, 10, 64); err == nil {
			adminTelegramID = val
		}
	}

	cleanupInterval := 60
	if valStr := os.Getenv("CLEANUP_INTERVAL_MINUTES"); valStr != "" {
		if val, err := strconv.Atoi(valStr); err == nil && val > 0 {
			cleanupInterval = val
		}
	}

	maxFileAge := 24
	if valStr := os.Getenv("MAX_FILE_AGE_HOURS"); valStr != "" {
		if val, err := strconv.Atoi(valStr); err == nil && val > 0 {
			maxFileAge = val
		}
	}

	return &Config{
		BotToken:               token,
		APIURL:                 apiURL,
		DownloadDir:            downloadDir,
		CacheDir:               cacheDir,
		DBPath:                 dbPath,
		MaxConcurrent:          maxConcurrent,
		PublicURL:              publicURL,
		ServerPort:             serverPort,
		AdminPassword:          adminPassword,
		AdminTelegramID:        adminTelegramID,
		CleanupIntervalMinutes: cleanupInterval,
		MaxFileAgeHours:        maxFileAge,
	}
}
