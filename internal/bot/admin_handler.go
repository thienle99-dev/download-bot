package bot

import (
	"context"
	"fmt"
	"html"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const adminPageSize = 5

func (s *BotServer) handleAdminCommand(ctx context.Context, b *bot.Bot, msg *models.Message) {
	if s.cfg.AdminTelegramID == 0 {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "⚠️ AdminTelegramID chưa được cấu hình trên hệ thống. Vui lòng cấu hình biến môi trường ADMIN_TELEGRAM_ID.",
		})
		return
	}

	if msg.From.ID != s.cfg.AdminTelegramID {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "❌ Bạn không có quyền quản trị viên.",
		})
		return
	}

	s.sendAdminDashboard(ctx, b, msg.Chat.ID, 1, 0)
}

func (s *BotServer) sendAdminDashboard(ctx context.Context, b *bot.Bot, chatID int64, page int, editMessageID int) {
	// Fetch all history to paginate locally
	history, err := s.db.GetAllHistory(1000)
	if err != nil {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Không thể truy vấn danh sách file tải về từ database.",
		})
		return
	}

	totalItems := len(history)
	if totalItems == 0 {
		text := "📋 Chưa có video nào được tải về trên hệ thống."
		if editMessageID > 0 {
			_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    chatID,
				MessageID: editMessageID,
				Text:      text,
			})
		} else {
			_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   text,
			})
		}
		return
	}

	totalPages := int(math.Ceil(float64(totalItems) / float64(adminPageSize)))
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}

	startIndex := (page - 1) * adminPageSize
	endIndex := startIndex + adminPageSize
	if endIndex > totalItems {
		endIndex = totalItems
	}

	// Adjust page if current page became empty after deletions
	if startIndex >= totalItems && page > 1 {
		page = totalPages
		startIndex = (page - 1) * adminPageSize
		endIndex = startIndex + adminPageSize
		if endIndex > totalItems {
			endIndex = totalItems
		}
	}

	pageItems := history[startIndex:endIndex]

	var sb strings.Builder
	sb.WriteString("🛠️ <b>Quản lý tệp tải về hệ thống (Admin):</b>\n")
	sb.WriteString(fmt.Sprintf("<i>Trang %d/%d (Tổng số: %d bản ghi)</i>\n\n", page, totalPages, totalItems))
	sb.WriteString("Chọn nút ❌ bên cạnh video để xóa vĩnh viễn khỏi VPS và SQLite database.\n\n")

	for i, h := range pageItems {
		existSymbol := "🟢"
		if _, err := os.Stat(h.FilePath); err != nil {
			existSymbol = "🔴"
		}
		title := h.Title
		if len(title) > 35 {
			title = title[:32] + "..."
		}
		sizeMB := float64(h.FileSize) / (1024 * 1024)
		itemNum := startIndex + i + 1
		sb.WriteString(fmt.Sprintf("%d. %s <b>%s</b>\n", itemNum, existSymbol, html.EscapeString(title)))
		sb.WriteString(fmt.Sprintf("   • ID: <code>%d</code> | Định dạng: <code>%s</code> | %.1f MB\n", h.ID, h.Format, sizeMB))
		sb.WriteString(fmt.Sprintf("   • User ID: <code>%d</code>\n\n", h.UserID))
	}

	// Build Inline Keyboard
	var rows [][]models.InlineKeyboardButton

	// Video List buttons with Delete button
	for _, h := range pageItems {
		title := h.Title
		if len(title) > 22 {
			title = title[:19] + "..."
		}
		existSymbol := "🟢"
		if _, err := os.Stat(h.FilePath); err != nil {
			existSymbol = "🔴"
		}
		rows = append(rows, []models.InlineKeyboardButton{
			{
				Text:         fmt.Sprintf("%s %s", existSymbol, title),
				CallbackData: fmt.Sprintf("adm:info:%d", h.ID),
			},
			{
				Text:         "❌ Xóa",
				CallbackData: fmt.Sprintf("adm:del:%d:%d", h.ID, page),
			},
		})
	}

	// Pagination row
	var navRow []models.InlineKeyboardButton
	if page > 1 {
		navRow = append(navRow, models.InlineKeyboardButton{
			Text:         "⬅️ Trước",
			CallbackData: fmt.Sprintf("adm:list:%d", page-1),
		})
	} else {
		navRow = append(navRow, models.InlineKeyboardButton{
			Text:         "▪️",
			CallbackData: "adm:noop",
		})
	}

	navRow = append(navRow, models.InlineKeyboardButton{
		Text:         fmt.Sprintf("Trang %d/%d", page, totalPages),
		CallbackData: "adm:noop",
	})

	if page < totalPages {
		navRow = append(navRow, models.InlineKeyboardButton{
			Text:         "Sau ➡️",
			CallbackData: fmt.Sprintf("adm:list:%d", page+1),
		})
	} else {
		navRow = append(navRow, models.InlineKeyboardButton{
			Text:         "▪️",
			CallbackData: "adm:noop",
		})
	}
	rows = append(rows, navRow)

	// Close / Refresh row
	rows = append(rows, []models.InlineKeyboardButton{
		{
			Text:         "🔄 Làm mới",
			CallbackData: fmt.Sprintf("adm:list:%d", page),
		},
		{
			Text:         "🛑 Đóng menu",
			CallbackData: "adm:close",
		},
	})

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: rows,
	}

	if editMessageID > 0 {
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:      chatID,
			MessageID:   editMessageID,
			Text:        sb.String(),
			ParseMode:   models.ParseModeHTML,
			ReplyMarkup: keyboard,
		})
	} else {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      chatID,
			Text:        sb.String(),
			ParseMode:   models.ParseModeHTML,
			ReplyMarkup: keyboard,
		})
	}
}

func (s *BotServer) handleAdminCallback(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery) {
	chatID := callback.Message.Message.Chat.ID
	messageID := callback.Message.Message.ID
	data := callback.Data
	userID := callback.From.ID

	if userID != s.cfg.AdminTelegramID {
		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: callback.ID,
			Text:            "❌ Bạn không có quyền quản trị viên.",
			ShowAlert:       true,
		})
		return
	}

	parts := strings.Split(data, ":")
	if len(parts) < 2 {
		return
	}

	action := parts[1]

	switch action {
	case "noop":
		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: callback.ID,
		})

	case "close":
		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: callback.ID,
		})
		b.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    chatID,
			MessageID: messageID,
		})

	case "list":
		page := 1
		if len(parts) >= 3 {
			if p, err := strconv.Atoi(parts[2]); err == nil {
				page = p
			}
		}
		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: callback.ID,
		})
		s.sendAdminDashboard(ctx, b, chatID, page, messageID)

	case "info":
		idStr := parts[2]
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return
		}
		h, err := s.db.GetDownloadByID(id)
		if err != nil || h == nil {
			b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: callback.ID,
				Text:            "❌ Không tìm thấy thông tin video này.",
				ShowAlert:       true,
			})
			return
		}
		sizeMB := float64(h.FileSize) / (1024 * 1024)
		infoText := fmt.Sprintf("📺 %s\n\n• Platform: %s\n• Format: %s\n• Size: %.1f MB\n• Path: %s\n• URL: %s",
			h.Title, h.Platform, h.Format, sizeMB, h.FilePath, h.URL)
		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: callback.ID,
			Text:            infoText,
			ShowAlert:       true,
		})

	case "del":
		if len(parts) < 4 {
			return
		}
		idStr := parts[2]
		pageStr := parts[3]

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return
		}
		page, _ := strconv.Atoi(pageStr)

		// Fetch history record to delete the file physically too
		h, err := s.db.GetDownloadByID(id)
		if err != nil || h == nil {
			b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: callback.ID,
				Text:            "❌ Bản ghi không tồn tại hoặc đã được xóa trước đó.",
				ShowAlert:       true,
			})
			s.sendAdminDashboard(ctx, b, chatID, page, messageID)
			return
		}

		// Attempt to physically remove the file
		_ = os.Remove(h.FilePath)
		s.cache.Remove(h.UserID, h.URL)
		_ = s.db.DeleteDownload(id)

		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: callback.ID,
			Text:            "✅ Đã xóa file và bản ghi thành công!",
			ShowAlert:       false,
		})

		s.sendAdminDashboard(ctx, b, chatID, page, messageID)
	}
}
