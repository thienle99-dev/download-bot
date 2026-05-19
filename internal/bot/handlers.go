package bot

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (s *BotServer) isValidURL(rawURL string) bool {
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Host)
	return strings.Contains(host, "youtube.com") ||
		strings.Contains(host, "youtu.be") ||
		strings.Contains(host, "tiktok.com")
}

func (s *BotServer) handleCommand(ctx context.Context, b *bot.Bot, msg *models.Message) {
	cmd := strings.Split(msg.Text, " ")[0]
	switch cmd {
	case "/start":
		s.sendStartMessage(ctx, b, msg.Chat.ID)
	case "/help":
		s.sendHelpMessage(ctx, b, msg.Chat.ID)
	case "/history":
		s.sendHistoryMessage(ctx, b, msg.From.ID, msg.Chat.ID)
	default:
		s.sendHelpMessage(ctx, b, msg.Chat.ID)
	}
}

func (s *BotServer) sendStartMessage(ctx context.Context, b *bot.Bot, chatID int64) {
	text := `👋 *Xin chào! Tôi là Bot Tải Video/MP3.*

Tôi hỗ trợ tải video từ:
• *YouTube* (Chọn mọi chất lượng từ 480p đến 1080p, hoặc MP3)
• *TikTok* (Tải nhanh, tự động xóa watermark)

🚀 *Cách sử dụng:*
1. Gửi trực tiếp link YouTube hoặc TikTok cho tôi.
2. Chọn chất lượng bạn muốn tải ở menu hiện ra.
3. Bot sẽ tự tải, convert và gửi trả file cho bạn nhanh chóng!

_Tip: Bot có hỗ trợ Inline Mode! Gõ @username_bot <link> ở cuộc chat bất kỳ để tải nhanh._`

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      text,
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		log.Printf("Failed to send start message: %v", err)
	}
}

func (s *BotServer) sendHelpMessage(ctx context.Context, b *bot.Bot, chatID int64) {
	text := `📖 *HƯỚNG DẪN SỬ DỤNG BOT*

• *Tải video/audio:* Chỉ cần gửi trực tiếp link video (YouTube/TikTok) vào chat.
• *Lịch sử tải:* Gõ lệnh /history để xem lại 10 video bạn đã tải gần đây.
• *Inline mode:* Gõ @username_bot <link> để chia sẻ video trực tiếp vào cuộc chat của bạn bè.

*Các lệnh hiện có:*
/start - Bắt đầu sử dụng bot
/help - Hướng dẫn sử dụng
/history - Xem lịch sử tải gần nhất`

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      text,
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		log.Printf("Failed to send help message: %v", err)
	}
}

func (s *BotServer) sendHistoryMessage(ctx context.Context, b *bot.Bot, userID int64, chatID int64) {
	history, err := s.db.GetUserHistory(userID, 10)
	if err != nil {
		log.Printf("Failed to fetch user history: %v", err)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Đã xảy ra lỗi khi lấy lịch sử tải. Vui lòng thử lại sau.",
		})
		return
	}

	if len(history) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "📋 Lịch sử tải của bạn hiện đang trống. Hãy gửi link video để tải ngay nhé!",
		})
		return
	}

	var sb strings.Builder
	sb.WriteString("📋 *Lịch sử 10 lượt tải gần nhất của bạn:*\n\n")

	for i, h := range history {
		// Clean and limit title length for presentation
		title := h.Title
		if len(title) > 40 {
			title = title[:37] + "..."
		}

		sizeMB := float64(h.FileSize) / (1024 * 1024)
		typeIcon := "🎬"
		if h.Format == "mp3" {
			typeIcon = "🎵"
		}

		sb.WriteString(fmt.Sprintf("%d. %s *%s*\n", i+1, typeIcon, bot.EscapeMarkdown(title)))
		sb.WriteString(fmt.Sprintf("   • Định dạng: `%s` | Dung lượng: `%.2f MB`\n", h.Format, sizeMB))
		sb.WriteString(fmt.Sprintf("   • Nền tảng: `%s` | 📅 `%s`\n\n", h.Platform, h.CreatedAt.Format("02-01-15:04")))
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      sb.String(),
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		log.Printf("Failed to send history message: %v", err)
	}
}
