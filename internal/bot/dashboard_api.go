package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	aiPkg "download-bot/internal/ai"
	"download-bot/internal/storage"

	"github.com/go-telegram/bot"
)

// Helper to find the web/dist directory dynamically in different environments
func getWebDistDir() string {
	// Try 1: Relative to current working directory
	if _, err := os.Stat(filepath.Join("web", "dist", "index.html")); err == nil {
		return filepath.Join("web", "dist")
	}
	// Try 2: Docker absolute path
	if _, err := os.Stat("/data/web/dist/index.html"); err == nil {
		return "/data/web/dist"
	}
	// Try 3: Relative to executable
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		// Check execDir/web/dist
		path := filepath.Join(execDir, "web", "dist")
		if _, err := os.Stat(filepath.Join(path, "index.html")); err == nil {
			return path
		}
		// Check execDir/../web/dist (if run from cmd/bot)
		path = filepath.Join(execDir, "..", "web", "dist")
		if _, err := os.Stat(filepath.Join(path, "index.html")); err == nil {
			return path
		}
	}
	// Fallback
	return "web/dist"
}

// RegisterWebRoutes registers both the file delivery, Svelte static frontend, and API endpoints
func (s *BotServer) RegisterWebRoutes(mux *http.ServeMux) {
	// Serve files for public download > 50MB
	mux.Handle("/files/", http.StripPrefix("/files/", http.FileServer(http.Dir(s.cfg.DownloadDir))))

	// Serve the Admin Dashboard Svelte static bundle
	distDir := getWebDistDir()
	log.Printf("[Dashboard] Serving static files from: %s", distDir)
	mux.Handle("/dashboard/", http.StripPrefix("/dashboard/", http.FileServer(http.Dir(distDir))))

	// WebSocket log stream endpoint
	mux.HandleFunc("/dashboard/api/ws", s.handleWebSocketLogs)

	// Admin API endpoints
	mux.HandleFunc("/dashboard/api/stats", s.handleStatsAPI)
	mux.HandleFunc("/dashboard/api/history", s.handleHistoryAPI)
	mux.HandleFunc("/dashboard/api/users", s.handleUsersAPI)
	mux.HandleFunc("/dashboard/api/delete", s.handleDeleteAPI)
	mux.HandleFunc("/dashboard/api/broadcast", s.handleBroadcastAPI)
	mux.HandleFunc("/dashboard/api/config", s.handleConfigAPI)
	mux.HandleFunc("/dashboard/api/queue", s.handleQueueAPI)

	// AI configuration endpoints
	mux.HandleFunc("/dashboard/api/ai/config", s.handleAIConfigAPI)
	mux.HandleFunc("/dashboard/api/ai/models", s.handleAIModelsAPI)

	log.Printf("----------------------------------------------------------------")
	log.Printf("🚀 Dashboard: %s/dashboard/", s.cfg.PublicURL)
	log.Printf("🔑 Password:  %s", s.cfg.AdminPassword)
	log.Printf("----------------------------------------------------------------")
}

// enableCORS writes CORS headers and handles preflight OPTIONS requests
func (s *BotServer) enableCORS(w http.ResponseWriter, r *http.Request) bool {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return true
	}
	return false
}

// authCheck validates the Authorization header
func (s *BotServer) authCheck(w http.ResponseWriter, r *http.Request) bool {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Missing authorization token"})
		return false
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token != s.cfg.AdminPassword {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Invalid admin password"})
		return false
	}

	return true
}

func (s *BotServer) handleStatsAPI(w http.ResponseWriter, r *http.Request) {
	if s.enableCORS(w, r) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if !s.authCheck(w, r) {
		return
	}

	history, err := s.db.GetAllHistory(1000)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	users, err := s.db.GetUsersStats()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Calculate storage space
	var totalSpace int64
	files, err := os.ReadDir(s.cfg.DownloadDir)
	if err == nil {
		for _, f := range files {
			if info, err := f.Info(); err == nil {
				totalSpace += info.Size()
			}
		}
	}

	stats := map[string]interface{}{
		"total_downloads": len(history),
		"total_users":     len(users),
		"storage_used":    totalSpace,
		"max_concurrent":  s.cfg.MaxConcurrent,
	}

	_ = json.NewEncoder(w).Encode(stats)
}

func (s *BotServer) handleHistoryAPI(w http.ResponseWriter, r *http.Request) {
	if s.enableCORS(w, r) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if !s.authCheck(w, r) {
		return
	}

	history, err := s.db.GetAllHistory(200)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Double check if local files still exist physically on disk
	type responseItem struct {
		ID          int64     `json:"id"`
		UserID      int64     `json:"user_id"`
		ChatID      int64     `json:"chat_id"`
		URL         string    `json:"url"`
		Platform    string    `json:"platform"`
		Title       string    `json:"title"`
		Format      string    `json:"format"`
		FileSize    int64     `json:"file_size"`
		FileID      string    `json:"file_id"`
		CreatedAt   time.Time `json:"created_at"`
		FileExist   bool      `json:"file_exist"`
		DownloadURL string    `json:"download_url"`
	}

	resp := make([]responseItem, 0, len(history))
	for _, h := range history {
		exist := false
		if _, err := os.Stat(h.FilePath); err == nil {
			exist = true
		}
		var dlURL string
		if exist {
			dlURL = fmt.Sprintf("/files/%s", url.PathEscape(filepath.Base(h.FilePath)))
		}
		resp = append(resp, responseItem{
			ID:          h.ID,
			UserID:      h.UserID,
			ChatID:      h.ChatID,
			URL:         h.URL,
			Platform:    h.Platform,
			Title:       h.Title,
			Format:      h.Format,
			FileSize:    h.FileSize,
			FileID:      h.FileID,
			CreatedAt:   h.CreatedAt,
			FileExist:   exist,
			DownloadURL: dlURL,
		})
	}

	_ = json.NewEncoder(w).Encode(resp)
}

func (s *BotServer) handleUsersAPI(w http.ResponseWriter, r *http.Request) {
	if s.enableCORS(w, r) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if !s.authCheck(w, r) {
		return
	}

	users, err := s.db.GetUsersStats()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(users)
}

func (s *BotServer) handleDeleteAPI(w http.ResponseWriter, r *http.Request) {
	if s.enableCORS(w, r) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if !s.authCheck(w, r) {
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Invalid ID parameter"})
		return
	}

	// Fetch history record to delete the file physically too
	h, err := s.db.GetDownloadByID(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if h != nil {
		// Attempt to physically remove the file
		if err := os.Remove(h.FilePath); err != nil && !os.IsNotExist(err) {
			log.Printf("Failed to physically remove file %s: %v", h.FilePath, err)
		}
		// Also clean cache references
		s.cache.Remove(h.UserID, h.URL)
	}

	// Delete from DB
	if err := s.db.DeleteDownload(id); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	s.LogInfo("Bản ghi tải xuống ID %d và file vật lý đã bị xóa khỏi VPS.", id)
	_ = json.NewEncoder(w).Encode(map[string]string{"success": "Record and file deleted successfully"})
}

func (s *BotServer) handleBroadcastAPI(w http.ResponseWriter, r *http.Request) {
	if s.enableCORS(w, r) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if !s.authCheck(w, r) {
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Message string `json:"message"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Message == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Message is required"})
		return
	}

	users, err := s.db.GetUsersStats()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	s.LogWarn("Bắt đầu phát sóng tin thông báo (broadcast) tới %d users...", len(users))

	// Launch broadcast concurrently
	var wg sync.WaitGroup
	var successCount int64
	var failureCount int64
	var mu sync.Mutex

	ctx := context.Background()

	for _, u := range users {
		wg.Add(1)
		go func(chatID int64) {
			defer wg.Done()
			_, err := s.tgBot.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   req.Message,
			})
			mu.Lock()
			if err != nil {
				failureCount++
				log.Printf("Broadcast failure to chat %d: %v", chatID, err)
			} else {
				successCount++
			}
			mu.Unlock()
		}(u.ChatID)
	}

	wg.Wait()

	s.LogInfo("Phát sóng thành công: %d | Lỗi: %d | Tổng: %d", successCount, failureCount, len(users))

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": successCount,
		"failed":  failureCount,
		"total":   len(users),
	})
}

func (s *BotServer) handleConfigAPI(w http.ResponseWriter, r *http.Request) {
	if s.enableCORS(w, r) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if !s.authCheck(w, r) {
		return
	}

	// Return config parameters without secrets
	configInfo := map[string]interface{}{
		"download_dir":   s.cfg.DownloadDir,
		"cache_dir":      s.cfg.CacheDir,
		"db_path":        s.cfg.DBPath,
		"max_concurrent": s.cfg.MaxConcurrent,
		"public_url":     s.cfg.PublicURL,
		"server_port":    s.cfg.ServerPort,
	}

	_ = json.NewEncoder(w).Encode(configInfo)
}

func (s *BotServer) handleQueueAPI(w http.ResponseWriter, r *http.Request) {
	if s.enableCORS(w, r) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if !s.authCheck(w, r) {
		return
	}

	s.activeDownloadsMu.RLock()
	defer s.activeDownloadsMu.RUnlock()

	queue := make([]*QueueItem, 0, len(s.activeDownloads))
	for _, item := range s.activeDownloads {
		queue = append(queue, item)
	}

	_ = json.NewEncoder(w).Encode(queue)
}

// handleAIConfigAPI handles GET (read config) and POST (write config) for AI settings.
func (s *BotServer) handleAIConfigAPI(w http.ResponseWriter, r *http.Request) {
	if s.enableCORS(w, r) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if !s.authCheck(w, r) {
		return
	}

	switch r.Method {
	case http.MethodGet:
		cfg, err := s.db.GetAIConfig()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		body, _ := storage.MarshalAIConfig(cfg, true) // Mask API key in response
		w.Write(body)

	case http.MethodPost:
		var incoming storage.AIConfig
		if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body"})
			return
		}

		// If API key is "***" (masked), preserve the existing key
		if incoming.APIKey == "***" {
			existing, err := s.db.GetAIConfig()
			if err == nil {
				incoming.APIKey = existing.APIKey
			}
		}

		if err := s.db.SetAIConfig(incoming); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		s.LogInfo("AI config updated: model=%s enabled=%v", incoming.Model, incoming.Enabled)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// handleAIModelsAPI calls /v1/models on the configured base URL and returns the model list.
func (s *BotServer) handleAIModelsAPI(w http.ResponseWriter, r *http.Request) {
	if s.enableCORS(w, r) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if !s.authCheck(w, r) {
		return
	}

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	cfg, err := s.db.GetAIConfig()
	if err != nil || cfg.BaseURL == "" || cfg.APIKey == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "AI base_url and api_key must be configured first"})
		return
	}

	ctx := r.Context()
	client := aiPkg.NewClient(cfg.BaseURL, cfg.APIKey, "")
	models, err := client.ListModels(ctx)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{"models": models})
}
