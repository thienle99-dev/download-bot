package bot

import (
	"context"
	"fmt"
	"html"
	"log"
	"net/url"
	"os"
	"strconv"
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
	isAdmin := s.cfg.AdminTelegramID != 0 && msg.From.ID == s.cfg.AdminTelegramID

	if strings.HasPrefix(cmd, "/del_") && isAdmin {
		s.handleDelShortcut(ctx, b, msg, cmd)
		return
	}

	switch cmd {
	case "/start":
		s.sendStartMessage(ctx, b, msg.Chat.ID, msg.From.ID)
	case "/help":
		s.sendHelpMessage(ctx, b, msg.Chat.ID, msg.From.ID)
	case "/history":
		s.sendHistoryMessage(ctx, b, msg.From.ID, msg.Chat.ID)
	case "/clean":
		s.handleCleanCommand(ctx, b, msg)
	case "/id":
		s.handleIDCommand(ctx, b, msg)
	case "/admin":
		if isAdmin {
			s.handleAdminCommand(ctx, b, msg)
		} else {
			s.sendHelpMessage(ctx, b, msg.Chat.ID, msg.From.ID)
		}
	case "/del":
		if isAdmin {
			s.handleDelCommand(ctx, b, msg)
		} else {
			s.sendHelpMessage(ctx, b, msg.Chat.ID, msg.From.ID)
		}
	case "/ai":
		question := strings.TrimSpace(strings.TrimPrefix(msg.Text, cmd))
		if question == "" {
			_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:    msg.Chat.ID,
				ParseMode: models.ParseModeHTML,
				Text:      "🤖 <b>Trợ lý AI</b>\n\nCách dùng: <code>/ai [câu hỏi của bạn]</code>\n\nVí dụ: <code>/ai Thủ đô của Việt Nam là gì?</code>\n\nDùng <code>/ai_reset</code> để xóa lịch sử hội thoại.",
			})
		} else {
			go s.handleAIChat(ctx, b, msg, question)
		}
	case "/ai_reset":
		s.aiSessions.Clear(msg.From.ID)
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "✅ Đã xóa lịch sử hội thoại AI của bạn. Phiên chat mới bắt đầu!",
		})
	case "/ai_chat":
		s.aiChatMu.Lock()
		enabled := !s.aiChatEnabled[msg.From.ID]
		s.aiChatEnabled[msg.From.ID] = enabled
		s.aiChatMu.Unlock()

		statusText := "🔴 Đã TẮT Chế độ Chat liên tục. Bạn phải dùng lệnh /ai để hỏi AI."
		if enabled {
			statusText = "🟢 Đã BẬT Chế độ Chat liên tục. Mọi tin nhắn văn bản bạn gửi (không phải link) sẽ tự động được gửi tới AI."
		}
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   statusText,
		})
	case "/ai_model":
		if isAdmin {
			go s.handleAIModelCommand(ctx, b, msg)
		} else {
			s.sendHelpMessage(ctx, b, msg.Chat.ID, msg.From.ID)
		}
	default:
		s.sendHelpMessage(ctx, b, msg.Chat.ID, msg.From.ID)
	}
}

func (s *BotServer) sendStartMessage(ctx context.Context, b *bot.Bot, chatID int64, userID int64) {
	text := `👋 <b>Xin chào! Tôi là Bot Tải Video/MP3 & Xử lý ảnh.</b>

Tôi hỗ trợ:
• <b>Tải Video/Audio</b> từ YouTube, TikTok, Facebook, Instagram, Twitter/X, Bilibili, Douyin (tự động xóa watermark).
• <b>Xử lý ảnh</b> (Gửi 1 hoặc nhiều ảnh để nén JPEG hoặc convert định dạng PNG/WebP đóng gói ZIP).

🚀 <b>Cách sử dụng:</b>
1. Gửi trực tiếp link video hoặc các bức ảnh của bạn vào đây.
2. Chọn định dạng video/audio hoặc thao tác xử lý ảnh tương ứng ở menu hiện ra.
3. Bot sẽ tự xử lý và gửi trả file cho bạn nhanh chóng!

<i>Tip: Bot có hỗ trợ Inline Mode cho video! Gõ @username_bot &lt;link&gt; ở cuộc chat bất kỳ để tải nhanh.</i>`

	isAdmin := s.cfg.AdminTelegramID != 0 && userID == s.cfg.AdminTelegramID

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: BuildReplyKeyboard(isAdmin),
	})
	if err != nil {
		log.Printf("Failed to send start message: %v", err)
	}
}

func (s *BotServer) sendHelpMessage(ctx context.Context, b *bot.Bot, chatID int64, userID int64) {
	text := `📖 <b>HƯỚNG DẪN SỬ DỤNG BOT</b>

• <b>Tải video/audio:</b> Gửi trực tiếp link video (YouTube/TikTok/Facebook/Instagram/Twitter...) vào chat.
• <b>Xử lý ảnh:</b> Gửi 1 hoặc nhiều ảnh (album) vào chat, sau đó chọn thao tác nén/chuyển đổi định dạng để nhận file ZIP.
• <b>Hỏi đáp AI:</b> Gõ <code>/ai [câu hỏi]</code> hoặc gửi ảnh kèm caption <code>/ai [câu hỏi]</code> để hỏi trợ lý AI.
• <b>Làm sạch link:</b> Gõ lệnh /clean &lt;link&gt; để loại bỏ các mã theo dõi khỏi liên kết của bạn.
• <b>Lịch sử tải:</b> Gõ lệnh /history để xem lại 10 video bạn đã tải gần đây.
• <b>Inline mode:</b> Gõ @username_bot &lt;link&gt; để chia sẻ video trực tiếp vào cuộc chat của bạn bè.

<b>Các lệnh hiện có:</b>
/start - Bắt đầu sử dụng bot
/help - Hướng dẫn sử dụng
/ai [câu hỏi] - Hỏi đáp trực tiếp với trợ lý AI
/ai_chat - Bật/tắt chế độ Chat liên tục không cần gõ /ai
/ai_reset - Xóa ngữ cảnh/lịch sử trò chuyện AI hiện tại
/clean - Xóa mã theo dõi khỏi link
/id - Lấy User ID & Chat ID hiện tại
/history - Xem lịch sử tải gần nhất`

	isAdmin := s.cfg.AdminTelegramID != 0 && userID == s.cfg.AdminTelegramID
	if isAdmin {
		text += `

🛠️ <b>LỆNH QUẢN TRỊ VIÊN (ADMIN):</b>
/admin - Liệt kê 15 lượt tải gần đây trên toàn hệ thống kèm link xóa nhanh
/del &lt;ID&gt; - Xóa tệp vật lý trên VPS và bản ghi SQLite theo ID
/ai_model - Chọn trực tiếp Model AI cho toàn hệ thống từ danh sách provider`
	}

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: BuildReplyKeyboard(isAdmin),
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
		if h.Format == "mp3" || h.Format == "m4a" || h.Format == "flac" {
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

func (s *BotServer) handleIDCommand(ctx context.Context, b *bot.Bot, msg *models.Message) {
	text := fmt.Sprintf("👤 <b>Thông tin tài khoản:</b>\n\n• User ID: <code>%d</code>\n• Chat ID: <code>%d</code>", msg.From.ID, msg.Chat.ID)
	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    msg.Chat.ID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
	})
}

func (s *BotServer) handleDelCommand(ctx context.Context, b *bot.Bot, msg *models.Message) {
	parts := strings.SplitN(msg.Text, " ", 2)
	if len(parts) < 2 {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    msg.Chat.ID,
			Text:      "⚠️ Vui lòng sử dụng cú pháp: <code>/del &lt;ID&gt;</code>",
			ParseMode: models.ParseModeHTML,
		})
		return
	}
	id, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
	if err != nil {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "❌ ID không hợp lệ.",
		})
		return
	}
	s.performDelete(ctx, b, msg.Chat.ID, msg.From.ID, id)
}

func (s *BotServer) handleDelShortcut(ctx context.Context, b *bot.Bot, msg *models.Message, cmd string) {
	idStr := strings.TrimPrefix(cmd, "/del_")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "❌ ID không hợp lệ.",
		})
		return
	}
	s.performDelete(ctx, b, msg.Chat.ID, msg.From.ID, id)
}

func (s *BotServer) performDelete(ctx context.Context, b *bot.Bot, chatID int64, userID int64, id int64) {
	if s.cfg.AdminTelegramID == 0 {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "⚠️ AdminTelegramID chưa được cấu hình trên hệ thống. Vui lòng cấu hình biến môi trường ADMIN_TELEGRAM_ID.",
		})
		return
	}

	if userID != s.cfg.AdminTelegramID {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Bạn không có quyền quản trị viên.",
		})
		return
	}

	// Fetch history record to delete the file physically too
	h, err := s.db.GetDownloadByID(id)
	if err != nil {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   fmt.Sprintf("❌ Lỗi truy vấn dữ liệu: %v", err),
		})
		return
	}

	if h == nil {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Không tìm thấy bản ghi tải xuống với ID này.",
		})
		return
	}

	// Attempt to physically remove the file
	fileDeleted := false
	if err := os.Remove(h.FilePath); err == nil {
		fileDeleted = true
	} else if os.IsNotExist(err) {
		fileDeleted = true // Already gone
	} else {
		log.Printf("Failed to physically remove file %s: %v", h.FilePath, err)
	}

	// Also clean cache references
	s.cache.Remove(h.UserID, h.URL)

	// Delete from DB
	if err := s.db.DeleteDownload(id); err != nil {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   fmt.Sprintf("❌ Lỗi xóa khỏi database: %v", err),
		})
		return
	}

	statusText := "Đã xóa bản ghi khỏi SQLite."
	if fileDeleted {
		statusText = "Đã xóa file vật lý khỏi VPS và bản ghi khỏi SQLite."
	} else {
		statusText = "Đã xóa bản ghi khỏi SQLite (không tìm thấy file vật lý trên VPS)."
	}

	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      fmt.Sprintf("✅ <b>Đã xóa thành công ID %d!</b>\n📝 Trạng thái: %s", id, statusText),
		ParseMode: models.ParseModeHTML,
	})
	s.LogInfo("Admin %d đã xóa bản ghi ID %d và file vật lý thành công.", userID, id)
}
