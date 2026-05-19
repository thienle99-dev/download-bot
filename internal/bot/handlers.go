package bot

import (
	"context"
	"fmt"
	"html"
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
		strings.Contains(host, "tiktok.com") ||
		strings.Contains(host, "instagram.com") ||
		strings.Contains(host, "facebook.com") ||
		strings.Contains(host, "fb.watch") ||
		strings.Contains(host, "twitter.com") ||
		strings.Contains(host, "x.com") ||
		strings.Contains(host, "bilibili.com") ||
		strings.Contains(host, "douyin.com")
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
	case "/clean":
		s.handleCleanCommand(ctx, b, msg)
	default:
		s.sendHelpMessage(ctx, b, msg.Chat.ID)
	}
}

func (s *BotServer) sendStartMessage(ctx context.Context, b *bot.Bot, chatID int64) {
	text := `👋 <b>Xin chào! Tôi là Bot Tải Video/MP3 & Xử lý ảnh.</b>

Tôi hỗ trợ:
• <b>Tải Video/Audio</b> từ YouTube, TikTok, Facebook, Instagram, Twitter/X, Bilibili, Douyin (tự động xóa watermark).
• <b>Xử lý ảnh</b> (Gửi 1 hoặc nhiều ảnh để nén JPEG hoặc convert định dạng PNG/WebP đóng gói ZIP).

🚀 <b>Cách sử dụng:</b>
1. Gửi trực tiếp link video hoặc các bức ảnh của bạn vào đây.
2. Chọn định dạng video/audio hoặc thao tác xử lý ảnh tương ứng ở menu hiện ra.
3. Bot sẽ tự xử lý và gửi trả file cho bạn nhanh chóng!

<i>Tip: Bot có hỗ trợ Inline Mode cho video! Gõ @username_bot &lt;link&gt; ở cuộc chat bất kỳ để tải nhanh.</i>`

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
	})
	if err != nil {
		log.Printf("Failed to send start message: %v", err)
	}
}

func (s *BotServer) sendHelpMessage(ctx context.Context, b *bot.Bot, chatID int64) {
	text := `📖 <b>HƯỚNG DẪN SỬ DỤNG BOT</b>

• <b>Tải video/audio:</b> Gửi trực tiếp link video (YouTube/TikTok/Facebook/Instagram/Twitter...) vào chat.
• <b>Xử lý ảnh:</b> Gửi 1 hoặc nhiều ảnh (album) vào chat, sau đó chọn thao tác nén/chuyển đổi định dạng để nhận file ZIP.
• <b>Làm sạch link:</b> Gõ lệnh /clean &lt;link&gt; để loại bỏ các mã theo dõi khỏi liên kết của bạn.
• <b>Lịch sử tải:</b> Gõ lệnh /history để xem lại 10 video bạn đã tải gần đây.
• <b>Inline mode:</b> Gõ @username_bot &lt;link&gt; để chia sẻ video trực tiếp vào cuộc chat của bạn bè.

<b>Các lệnh hiện có:</b>
/start - Bắt đầu sử dụng bot
/help - Hướng dẫn sử dụng
/clean - Xóa mã theo dõi khỏi link
/history - Xem lịch sử tải gần nhất`

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
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
	sb.WriteString("📋 <b>Lịch sử 10 lượt tải gần nhất của bạn:</b>\n\n")

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

		sb.WriteString(fmt.Sprintf("%d. %s <b>%s</b>\n", i+1, typeIcon, html.EscapeString(title)))
		sb.WriteString(fmt.Sprintf("   • Định dạng: <code>%s</code> | Dung lượng: <code>%.2f MB</code>\n", h.Format, sizeMB))
		sb.WriteString(fmt.Sprintf("   • Nền tảng: <code>%s</code> | 📅 <code>%s</code>\n\n", h.Platform, h.CreatedAt.Format("02-01-15:04")))
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      sb.String(),
		ParseMode: models.ParseModeHTML,
	})
	if err != nil {
		log.Printf("Failed to send history message: %v", err)
	}
}

func (s *BotServer) handleCleanCommand(ctx context.Context, b *bot.Bot, msg *models.Message) {
	parts := strings.SplitN(msg.Text, " ", 2)
	if len(parts) < 2 {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    msg.Chat.ID,
			Text:      "⚠️ Vui lòng sử dụng cú pháp: <code>/clean &lt;link&gt;</code>",
			ParseMode: models.ParseModeHTML,
		})
		return
	}

	targetURL := strings.TrimSpace(parts[1])
	cleaned, err := CleanURL(targetURL)
	if err != nil {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "❌ Link không hợp lệ hoặc không thể phân tích cú pháp.",
		})
		return
	}

	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    msg.Chat.ID,
		Text:      fmt.Sprintf("🧹 <b>Link đã làm sạch:</b>\n<code>%s</code>", html.EscapeString(cleaned)),
		ParseMode: models.ParseModeHTML,
	})
}

