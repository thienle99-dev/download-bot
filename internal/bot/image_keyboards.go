package bot

import (
	"download-bot/internal/storage"

	"github.com/go-telegram/bot/models"
)

// BuildImageKeyboard creates inline keyboard containing options for image processing
func BuildImageKeyboard() *models.InlineKeyboardMarkup {
	var rows [][]models.InlineKeyboardButton

	// Row 1: Compression options
	rows = append(rows, []models.InlineKeyboardButton{
		{
			Text:         "🗜 Nén JPEG (75%)",
			CallbackData: "img:compress:jpeg",
		},
	})

	// Row 2: Format conversion options
	rows = append(rows, []models.InlineKeyboardButton{
		{
			Text:         "🔄 Chuyển PNG",
			CallbackData: "img:convert:png",
		},
		{
			Text:         "🔄 Chuyển WebP",
			CallbackData: "img:convert:webp",
		},
	})

	// Row 3: Sticker option
	rows = append(rows, []models.InlineKeyboardButton{
		{
			Text:         "✨ Tạo Sticker Set",
			CallbackData: "img:sticker",
		},
	})

	// Row 4: Cancel button
	rows = append(rows, []models.InlineKeyboardButton{
		{
			Text:         "❌ Hủy",
			CallbackData: "img:cancel",
		},
	})

	return &models.InlineKeyboardMarkup{
		InlineKeyboard: rows,
	}
}

// BuildStickerActionKeyboard creates inline keyboard for sticker pack choices
func BuildStickerActionKeyboard(sets []storage.StickerSet) *models.InlineKeyboardMarkup {
	var rows [][]models.InlineKeyboardButton

	// Button to create new set
	rows = append(rows, []models.InlineKeyboardButton{
		{
			Text:         "🆕 Tạo Bộ Sticker Mới",
			CallbackData: "img:st_new",
		},
	})

	// List existing sets (max 4 to avoid overly large keyboard)
	for i, s := range sets {
		if i >= 4 {
			break
		}
		title := s.Title
		if len(title) > 22 {
			title = title[:19] + "..."
		}
		rows = append(rows, []models.InlineKeyboardButton{
			{
				Text:         "➕ Thêm vào: " + title,
				CallbackData: "img:st_add:" + s.Name,
			},
		})
	}

	// Back & Cancel buttons
	rows = append(rows, []models.InlineKeyboardButton{
		{
			Text:         "⬅️ Quay lại",
			CallbackData: "img:st_back",
		},
		{
			Text:         "❌ Hủy",
			CallbackData: "img:cancel",
		},
	})

	return &models.InlineKeyboardMarkup{
		InlineKeyboard: rows,
	}
}

