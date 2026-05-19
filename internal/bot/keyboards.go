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
				CallbackData: fmt.Sprintf("dl:%s:%s", format.ID, urlHash),
			},
		})
	}

	// Add Compressor button
	rows = append(rows, []models.InlineKeyboardButton{
		{
			Text:         "🗜 Nén & Hạ Độ phân giải",
			CallbackData: fmt.Sprintf("compress:%s", urlHash),
		},
	})

	// Add Subtitle button
	rows = append(rows, []models.InlineKeyboardButton{
		{
			Text:         "📝 Tải Phụ Đề (Subtitles)",
			CallbackData: fmt.Sprintf("sub:%s", urlHash),
		},
	})

	// Add Cut Clip button
	rows = append(rows, []models.InlineKeyboardButton{
		{
			Text:         "✂️ Cắt Clip & GIF (Max 60s)",
			CallbackData: fmt.Sprintf("cut:%s", urlHash),
		},
	})

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

// BuildSubLanguageKeyboard creates inline keyboard showing available subtitle languages.
func BuildSubLanguageKeyboard(urlHash string, langs []string) *models.InlineKeyboardMarkup {
	var rows [][]models.InlineKeyboardButton

	// Create buttons in rows of 2 for a clean layout
	var currentRow []models.InlineKeyboardButton
	for _, lang := range langs {
		langName := lang
		switch lang {
		case "vi":
			langName = "🇻🇳 Tiếng Việt (vi)"
		case "en":
			langName = "🇬🇧 Tiếng Anh (en)"
		case "ja":
			langName = "🇯🇵 Tiếng Nhật (ja)"
		case "ko":
			langName = "🇰🇷 Tiếng Hàn (ko)"
		case "zh":
			langName = "🇨🇳 Tiếng Trung (zh)"
		case "th":
			langName = "🇹🇭 Tiếng Thái (th)"
		}

		currentRow = append(currentRow, models.InlineKeyboardButton{
			Text:         langName,
			CallbackData: fmt.Sprintf("lang:%s:%s", lang, urlHash),
		})

		if len(currentRow) == 2 {
			rows = append(rows, currentRow)
			currentRow = nil
		}
	}
	if len(currentRow) > 0 {
		rows = append(rows, currentRow)
	}

	// Add Back & Cancel button row
	rows = append(rows, []models.InlineKeyboardButton{
		{
			Text:         "⬅️ Quay lại",
			CallbackData: fmt.Sprintf("back:%s", urlHash),
		},
		{
			Text:         "❌ Hủy",
			CallbackData: "cancel",
		},
	})

	return &models.InlineKeyboardMarkup{
		InlineKeyboard: rows,
	}
}

// BuildSubDownloadTypeKeyboard creates inline keyboard to choose subtitle download type.
func BuildSubDownloadTypeKeyboard(urlHash string, lang string) *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{
					Text:         "📝 Chỉ tải File Phụ đề (.srt)",
					CallbackData: fmt.Sprintf("dlsub:srt:%s:%s", lang, urlHash),
				},
			},
			{
				{
					Text:         "🎬 Tải cả Video + Phụ đề (.srt)",
					CallbackData: fmt.Sprintf("dlsub:both:%s:%s", lang, urlHash),
				},
			},
			{
				{
					Text:         "🤖 Tóm tắt Video bằng AI",
					CallbackData: fmt.Sprintf("dlsub:summary:%s:%s", lang, urlHash),
				},
			},
			{
				{
					Text:         "⬅️ Quay lại",
					CallbackData: fmt.Sprintf("sub:%s", urlHash),
				},
				{
					Text:         "❌ Hủy",
					CallbackData: "cancel",
				},
			},
		},
	}
}

// BuildCompressOptionsKeyboard creates inline keyboard showing compression options.
func BuildCompressOptionsKeyboard(urlHash string) *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{
					Text:         "📉 Hạ xuống 720p (Nén trung bình)",
					CallbackData: fmt.Sprintf("dlcomp:720:%s", urlHash),
				},
			},
			{
				{
					Text:         "📉 Hạ xuống 480p (Nén tiết kiệm)",
					CallbackData: fmt.Sprintf("dlcomp:480:%s", urlHash),
				},
			},
			{
				{
					Text:         "🗜 Giữ nguyên độ phân giải",
					CallbackData: fmt.Sprintf("dlcomp:same:%s", urlHash),
				},
			},
			{
				{
					Text:         "⬅️ Quay lại",
					CallbackData: fmt.Sprintf("back:%s", urlHash),
				},
				{
					Text:         "❌ Hủy",
					CallbackData: "cancel",
				},
			},
		},
	}
}

// BuildAIModelKeyboard creates inline keyboard containing available AI models.
func (s *BotServer) BuildAIModelKeyboard(modelsList []string) *models.InlineKeyboardMarkup {
	var rows [][]models.InlineKeyboardButton

	// Limit to top 25 models to avoid giant keyboard errors
	limit := len(modelsList)
	if limit > 25 {
		limit = 25
	}

	for i := 0; i < limit; i++ {
		modelID := modelsList[i]
		// Register in urlMap to get a short hash and support long model names safely
		modelHash := s.registerURL(modelID)
		
		rows = append(rows, []models.InlineKeyboardButton{
			{
				Text:         modelID,
				CallbackData: fmt.Sprintf("setmodel:%s", modelHash),
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


