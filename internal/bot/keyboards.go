package bot

import (
	"download-bot/internal/downloader"
	"fmt"

	"github.com/go-telegram/bot/models"
)

// BuildFormatKeyboard creates inline keyboard containing options to download in different video/audio formats.
// The urlHash is a short unique key mapping to the full URL (solving Telegram's 64-character callback limit).
func BuildFormatKeyboard(urlHash string) *models.InlineKeyboardMarkup {
	var rows [][]models.InlineKeyboardButton

	// Create rows of format options
	for _, format := range downloader.AvailableFormats {
		rows = append(rows, []models.InlineKeyboardButton{
			{
				Text:         format.Label,
				CallbackData: fmt.Sprintf("dl:%s:%s", format.Extension, urlHash),
			},
		})
	}

	// Add Cancel button
	rows = append(rows, []models.InlineKeyboardButton{
		{
			Text:         "❌ Hủy",
			CallbackData: "cancel",
		},
	})

	return &models.InlineKeyboardMarkup{
		InlineKeyboard: rows,
	}
}
