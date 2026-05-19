package bot

import (
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

	// Row 3: Cancel button
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
