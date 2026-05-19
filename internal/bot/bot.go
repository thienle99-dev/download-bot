package bot

import (
	"context"
	"crypto/md5"
	"download-bot/internal/cache"
	"download-bot/internal/config"
	"download-bot/internal/downloader"
	"download-bot/internal/storage"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type QueueItem struct {
	ID        string    `json:"id"`
	UserID    int64     `json:"user_id"`
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	Progress  float64   `json:"progress"`
	StartedAt time.Time `json:"started_at"`
}

type BotServer struct {
	tgBot             *bot.Bot
	db                *storage.DB
	dl                *downloader.Downloader
	cache             *cache.FileCache
	cfg               *config.Config
	urlMap            map[string]string
	urlMapMu          sync.RWMutex
	logHub            *LogHub
	activeDownloads   map[string]*QueueItem
	activeDownloadsMu sync.RWMutex
	imageSessions     map[int64]*ImageSession
	imageSessionsMu   sync.RWMutex
	waitingForCut     map[int64]string
	waitingForCutMu   sync.Mutex
}

func NewBotServer(cfg *config.Config, db *storage.DB) (*BotServer, error) {
	// Initialize Downloader
	dl := downloader.NewDownloader(cfg.DownloadDir, cfg.MaxConcurrent)

	// Initialize File Cache
	fileCache := cache.NewFileCache(3) // "Cho lưu 3 video gần nhất"

	server := &BotServer{
		db:              db,
		dl:              dl,
		cache:           fileCache,
		cfg:             cfg,
		urlMap:          make(map[string]string),
		logHub:          NewLogHub(),
		activeDownloads: make(map[string]*QueueItem),
		imageSessions:   make(map[int64]*ImageSession),
		waitingForCut:   make(map[int64]string),
	}

	// Try pre-populating the cache from SQLite
	server.prepopulateCache()

	opts := []bot.Option{
		bot.WithDefaultHandler(server.routeUpdate),
	}

	// Check if we need to use a self-hosted API URL
	if cfg.APIURL != "" {
		log.Printf("Using self-hosted Telegram Bot API Server at: %s", cfg.APIURL)
		opts = append(opts, bot.WithServerURL(cfg.APIURL))
	}

	b, err := bot.New(cfg.BotToken, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	server.tgBot = b
	return server, nil
}

func (s *BotServer) Start(ctx context.Context) {
	log.Println("Starting Telegram Bot listener...")
	go s.StartSessionCleaner(ctx)
	s.StartCleanupScheduler(ctx)
	s.tgBot.Start(ctx)
}

// routeUpdate delegates incoming Telegram updates to their respective handlers
func (s *BotServer) routeUpdate(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update == nil {
		return
	}

	// Handle Callback Queries (button clicks)
	if update.CallbackQuery != nil {
		s.handleCallback(ctx, b, update.CallbackQuery)
		return
	}

	// Handle Inline Queries
	if update.InlineQuery != nil {
		s.handleInlineQuery(ctx, b, update.InlineQuery)
		return
	}

	// Handle standard messages
	if update.Message != nil {
		// Handle photo messages (image processing feature)
		if update.Message.Photo != nil && len(update.Message.Photo) > 0 {
			s.handlePhoto(ctx, b, update.Message)
			return
		}

		text := strings.TrimSpace(update.Message.Text)
		userID := update.Message.From.ID

		// Check if it's a command
		if len(text) > 0 && text[0] == '/' {
			s.handleCommand(ctx, b, update.Message)
			return
		}

		// Check if user is in waitingForCut state
		s.waitingForCutMu.Lock()
		videoURLForCut, isWaiting := s.waitingForCut[userID]
		s.waitingForCutMu.Unlock()

		if isWaiting && text != "" {
			// Clear waiting state
			s.waitingForCutMu.Lock()
			delete(s.waitingForCut, userID)
			s.waitingForCutMu.Unlock()

			// Call handler to slice the video
			s.handleCutProcess(ctx, b, update.Message, videoURLForCut, text)
			return
		}

		// Try parsing: <URL> <timestamp> (e.g. "https://youtu.be/... 10-30")
		parts := strings.Fields(text)
		if len(parts) >= 2 {
			urlPart := parts[0]
			rangePart := parts[1]

			cleanedURL, err := CleanURL(urlPart)
			if err == nil && s.isValidURL(cleanedURL) {
				s.handleCutProcess(ctx, b, update.Message, cleanedURL, rangePart)
				return
			}
		}

		// Otherwise check if it's a valid video URL
		cleanedURL, err := CleanURL(text)
		if err == nil && s.isValidURL(cleanedURL) {
			s.promptFormatSelection(ctx, b, update.Message, cleanedURL)
			return
		}

		// Fallback start message
		s.sendHelpMessage(ctx, b, update.Message.Chat.ID, update.Message.From.ID)
	}
}

// registerURL maps a long URL to a short hash to resolve Telegram's 64-char callback limit
func (s *BotServer) registerURL(url string) string {
	s.urlMapMu.Lock()
	defer s.urlMapMu.Unlock()

	hash := fmt.Sprintf("%x", md5.Sum([]byte(url)))[:12]
	s.urlMap[hash] = url
	return hash
}

func (s *BotServer) getURL(hash string) (string, bool) {
	s.urlMapMu.RLock()
	defer s.urlMapMu.RUnlock()

	url, exists := s.urlMap[hash]
	return url, exists
}

func (s *BotServer) prepopulateCache() {
	// Query last 100 entries from history to populate cache state
	rows, err := s.db.Query(`
		SELECT user_id, url, file_path, file_id, created_at 
		FROM download_history 
		ORDER BY created_at ASC
	`)
	if err != nil {
		log.Printf("Warning: failed to fetch history for pre-populating cache: %v", err)
		return
	}
	defer rows.Close()

	var history []cache.CacheEntry
	var userIDs []int64

	for rows.Next() {
		var uID int64
		var entry cache.CacheEntry
		var createdAt string
		err := rows.Scan(&uID, &entry.URL, &entry.FilePath, &entry.FileID, &createdAt)
		if err == nil {
			history = append(history, entry)
			userIDs = append(userIDs, uID)
		}
	}

	s.cache.LoadFromHistory(history, userIDs)
}
