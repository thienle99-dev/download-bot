package bot

import (
	"context"
	"encoding/base64"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"time"

	"download-bot/internal/ai"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// handleAIChat orchestrates the streaming AI chat flow:
// loads config, sends status, calls Stream API with session history, updates message progressively.
func (s *BotServer) handleAIChat(ctx context.Context, b *bot.Bot, msg *models.Message, question string) {
	chatID := msg.Chat.ID
	userID := msg.From.ID

	// 1. Load AI config from DB (with env var fallbacks for first-run)
	cfg, err := s.db.GetAIConfig()
	if err != nil {
		s.LogError("GetAIConfig failed: %v", err)
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Không thể tải cấu hình AI. Vui lòng thử lại sau.",
		})
		return
	}

	// Apply .env defaults if DB config is still empty
	if cfg.BaseURL == "" {
		if v := os.Getenv("OPEN_AI_URL"); v != "" {
			cfg.BaseURL = v
		}
	}
	if cfg.APIKey == "" {
		if v := os.Getenv("OPEN_AI_KEY"); v != "" {
			cfg.APIKey = v
		}
	}

	// Validate minimal required config
	if !cfg.Enabled && cfg.BaseURL == "" {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    chatID,
			ParseMode: models.ParseModeHTML,
			Text:      "⚠️ Tính năng AI chưa được bật.\n\nAdmin cần cấu hình <b>Base URL</b>, <b>API Key</b> và bật tính năng từ Dashboard.",
		})
		return
	}
	if cfg.BaseURL == "" || cfg.APIKey == "" || cfg.Model == "" {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    chatID,
			ParseMode: models.ParseModeHTML,
			Text:      "⚠️ Cấu hình AI chưa đầy đủ.\n\nAdmin cần điền <b>Base URL</b>, <b>API Key</b> và chọn <b>Model</b> từ Dashboard.",
		})
		return
	}

	// 2. Send status message
	statusMsg, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   "🤖 Đang suy nghĩ...",
	})
	if err != nil {
		return
	}

	// 3. Add user message to session history
	s.aiSessions.Add(userID, "user", question)

	// 4. Call AI Stream API with full conversation history
	client := ai.NewClient(cfg.BaseURL, cfg.APIKey, cfg.Model)
	history := s.aiSessions.Get(userID)

	var fullReply string
	var lastEditTime time.Time

	err = client.ChatStream(ctx, cfg.SystemPrompt, history, func(token string) {
		fullReply += token

		// To prevent Telegram rate limiting (max ~1 edit/sec per message), throttle edits
		if time.Since(lastEditTime) > 1200*time.Millisecond {
			lastEditTime = time.Now()
			replyText := fmt.Sprintf("🤖 <b>Trợ lý AI</b>\n\n%s", html.EscapeString(fullReply))
			_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    chatID,
				MessageID: statusMsg.ID,
				Text:      replyText,
				ParseMode: models.ParseModeHTML,
			})
		}
	})

	if err != nil {
		s.LogError("AI Stream Chat failed for user %d: %v", userID, err)
		errMsg := fmt.Sprintf("❌ Lỗi AI: %v", err)
		if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "Unauthorized") {
			errMsg = "❌ API Key không hợp lệ. Admin cần kiểm tra lại cấu hình."
		} else if strings.Contains(err.Error(), "429") {
			errMsg = "⏳ Quá nhiều yêu cầu. Vui lòng chờ một lúc rồi thử lại."
		}

		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: statusMsg.ID,
			Text:      errMsg,
			ParseMode: models.ParseModeHTML,
		})
		// Clear session on hard failure to prevent dirty state
		s.aiSessions.Clear(userID)
		return
	}

	// 5. Final update to ensure the message is fully completed
	replyText := fmt.Sprintf("🤖 <b>Trợ lý AI</b>\n\n%s", html.EscapeString(fullReply))
	_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: statusMsg.ID,
		Text:      replyText,
		ParseMode: models.ParseModeHTML,
	})

	// 6. Add assistant reply to session
	s.aiSessions.Add(userID, "assistant", fullReply)
}

// handleAIVision downloads the photo, encodes to base64 and calls OpenAI-compatible Multi-modal endpoint.
func (s *BotServer) handleAIVision(ctx context.Context, b *bot.Bot, msg *models.Message) {
	chatID := msg.Chat.ID
	userID := msg.From.ID

	// 1. Clean caption command prefix if any
	caption := strings.TrimSpace(msg.Caption)
	question := strings.TrimSpace(strings.TrimPrefix(caption, "/ai"))
	if question == "" {
		question = "Phân tích bức ảnh này và mô tả chi tiết."
	}

	// 2. Load AI config
	cfg, err := s.db.GetAIConfig()
	if err != nil || !cfg.Enabled || cfg.BaseURL == "" || cfg.APIKey == "" {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "⚠️ AI chưa được bật hoặc cấu hình chưa đầy đủ.",
		})
		return
	}

	if len(msg.Photo) == 0 {
		return
	}

	// 3. Send initial status
	statusMsg, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   "🤖 Đang tải và phân tích hình ảnh...",
	})
	if err != nil {
		return
	}

	// 4. Download largest photo
	photo := msg.Photo[len(msg.Photo)-1]
	tgFile, err := b.GetFile(ctx, &bot.GetFileParams{FileID: photo.FileID})
	if err != nil {
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: statusMsg.ID,
			Text:      "❌ Không thể lấy thông tin ảnh từ Telegram.",
		})
		return
	}

	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("ai_vision_%d_%s", userID, filepath.Base(tgFile.FilePath)))
	_ = os.MkdirAll(filepath.Dir(tempFile), 0755)

	if err := s.downloadTelegramFile(ctx, tgFile.FilePath, tempFile); err != nil {
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: statusMsg.ID,
			Text:      "❌ Lỗi tải ảnh về hệ thống.",
		})
		return
	}
	defer os.Remove(tempFile)

	// 5. Read image and encode to base64 data URL
	imgData, err := os.ReadFile(tempFile)
	if err != nil {
		return
	}
	mimeType := "image/jpeg"
	ext := strings.ToLower(filepath.Ext(tempFile))
	if ext == ".png" {
		mimeType = "image/png"
	} else if ext == ".webp" {
		mimeType = "image/webp"
	}
	base64Data := base64.StdEncoding.EncodeToString(imgData)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, base64Data)

	// 6. Build content parts with text prompt and base64 image URL
	contentParts := []ai.ContentPart{
		{
			Type: "text",
			Text: question,
		},
		{
			Type: "image_url",
			ImageURL: &ai.ImageURLParam{
				URL: dataURL,
			},
		},
	}

	userMsg := ai.Message{
		Role:    "user",
		Content: contentParts,
	}

	// We only send the single vision query to avoid context window explosion
	history := []ai.Message{userMsg}

	// 7. Stream Vision reply
	client := ai.NewClient(cfg.BaseURL, cfg.APIKey, cfg.Model)

	var lastEditTime time.Time
	var fullReply string

	err = client.ChatStream(ctx, cfg.SystemPrompt, history, func(token string) {
		fullReply += token

		if time.Since(lastEditTime) > 1200*time.Millisecond {
			lastEditTime = time.Now()
			replyText := fmt.Sprintf("🤖 <b>Trợ lý AI (Vision)</b>\n\n%s", html.EscapeString(fullReply))
			_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    chatID,
				MessageID: statusMsg.ID,
				Text:      replyText,
				ParseMode: models.ParseModeHTML,
			})
		}
	})

	if err != nil {
		s.LogError("Vision Chat failed: %v", err)
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: statusMsg.ID,
			Text:      fmt.Sprintf("❌ Lỗi xử lý Vision. Hãy đảm bảo model hỗ trợ Multi-modal API. Chi tiết: %v", err),
		})
		return
	}

	// Final message update
	replyText := fmt.Sprintf("🤖 <b>Trợ lý AI (Vision)</b>\n\n%s", html.EscapeString(fullReply))
	_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: statusMsg.ID,
		Text:      replyText,
		ParseMode: models.ParseModeHTML,
	})

	// Add simplified text session context so text continuation works
	s.aiSessions.Add(userID, "user", fmt.Sprintf("[Phân tích ảnh] %s", question))
	s.aiSessions.Add(userID, "assistant", fullReply)
}

// handleAIModelCommand fetches the list of models and shows an inline keyboard for selection.
func (s *BotServer) handleAIModelCommand(ctx context.Context, b *bot.Bot, msg *models.Message) {
	chatID := msg.Chat.ID

	// 1. Send temporary loading message
	statusMsg, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   "🔍 Đang tải danh sách models từ AI provider...",
	})
	if err != nil {
		return
	}

	// 2. Load AI Config
	cfg, err := s.db.GetAIConfig()
	if err != nil {
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: statusMsg.ID,
			Text:      "❌ Không thể tải cấu hình AI từ database.",
		})
		return
	}

	// Fallback environment settings
	if cfg.BaseURL == "" {
		cfg.BaseURL = os.Getenv("OPEN_AI_URL")
	}
	if cfg.APIKey == "" {
		cfg.APIKey = os.Getenv("OPEN_AI_KEY")
	}

	if cfg.BaseURL == "" || cfg.APIKey == "" {
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: statusMsg.ID,
			Text:      "⚠️ Cấu hình AI chưa đầy đủ. Hãy điền Base URL và API Key trong Dashboard.",
		})
		return
	}

	// 3. Request models from provider
	client := ai.NewClient(cfg.BaseURL, cfg.APIKey, "")
	modelsList, err := client.ListModels(ctx)
	if err != nil {
		s.LogError("Failed to list models via bot command: %v", err)
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: statusMsg.ID,
			Text:      fmt.Sprintf("❌ Lỗi khi lấy danh sách model: %v", err),
		})
		return
	}

	if len(modelsList) == 0 {
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: statusMsg.ID,
			Text:      "⚠️ Không tìm thấy model khả dụng nào từ nhà cung cấp.",
		})
		return
	}

	// 4. Update the message with the list of models inside inline keyboard
	text := fmt.Sprintf("🤖 <b>Chọn Model AI Hệ Thống</b>\n\nBase URL: <code>%s</code>\nModel hiện tại: <code>%s</code>\n\nVui lòng chọn model mới từ danh sách dưới đây:", html.EscapeString(cfg.BaseURL), html.EscapeString(cfg.Model))

	_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      chatID,
		MessageID:   statusMsg.ID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: s.BuildAIModelKeyboard(modelsList),
	})
}
