package bot

import (
	"context"
	"fmt"
	"html"
	"os"
	"strings"

	"download-bot/internal/ai"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// handleAIChat orchestrates the full AI chat flow:
// loads config, sends status, calls API with session history, updates message.
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

	// Apply .env defaults if DB config is still empty (first-run bootstrap)
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
		Text:   "🤖 Đang xử lý...",
	})
	if err != nil {
		return
	}

	// 3. Add user message to session history
	s.aiSessions.Add(userID, "user", question)

	// 4. Call AI with full conversation history
	client := ai.NewClient(cfg.BaseURL, cfg.APIKey, cfg.Model)
	history := s.aiSessions.Get(userID)

	reply, err := client.Chat(ctx, cfg.SystemPrompt, history)
	if err != nil {
		s.LogError("AI Chat failed for user %d: %v", userID, err)
		errMsg := "❌ AI không phản hồi. Vui lòng thử lại sau."
		if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "Unauthorized") {
			errMsg = "❌ API Key không hợp lệ. Admin cần kiểm tra lại cấu hình."
		} else if strings.Contains(err.Error(), "429") {
			errMsg = "⏳ Quá nhiều yêu cầu. Vui lòng chờ một lúc rồi thử lại."
		} else if strings.Contains(err.Error(), "model") {
			errMsg = fmt.Sprintf("❌ Model không hợp lệ: <code>%s</code>. Admin cần chọn lại model từ Dashboard.", html.EscapeString(cfg.Model))
		}
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: statusMsg.ID,
			Text:      errMsg,
			ParseMode: models.ParseModeHTML,
		})
		// Remove the failed user message from history to avoid polluting context
		s.aiSessions.Clear(userID)
		return
	}

	// 5. Add assistant reply to session
	s.aiSessions.Add(userID, "assistant", reply)

	// 6. Send reply — format as HTML with escaped content
	replyText := fmt.Sprintf("🤖 <b>Trợ lý AI</b>\n\n%s", html.EscapeString(reply))

	_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: statusMsg.ID,
		Text:      replyText,
		ParseMode: models.ParseModeHTML,
	})
}
