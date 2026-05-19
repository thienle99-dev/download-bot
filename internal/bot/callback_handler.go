package bot

import (
	"context"
	"download-bot/internal/downloader"
	"fmt"
	"html"
	"log"
	"math"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (s *BotServer) handleCallback(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery) {
	chatID := callback.Message.Message.Chat.ID
	messageID := callback.Message.Message.ID
	data := callback.Data
	userID := callback.From.ID

	// Answer callback query immediately to stop loading spinner in UI
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: callback.ID,
	})

	if data == "cancel" {
		b.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    chatID,
			MessageID: messageID,
		})
		return
	}

	if data == "cancel_cut" {
		s.waitingForCutMu.Lock()
		delete(s.waitingForCut, userID)
		s.waitingForCutMu.Unlock()

		b.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    chatID,
			MessageID: messageID,
		})
		return
	}

	if strings.HasPrefix(data, "img:") {
		s.handleImageCallback(ctx, b, callback)
		return
	}

	if strings.HasPrefix(data, "adm:") {
		s.handleAdminCallback(ctx, b, callback)
		return
	}

	if strings.HasPrefix(data, "cut:") {
		s.handleCutCallback(ctx, b, callback)
		return
	}

	if strings.HasPrefix(data, "compress:") {
		s.handleCompressSelection(ctx, b, callback)
		return
	}

	if strings.HasPrefix(data, "dlcomp:") {
		s.handleCompressDownloadTrigger(ctx, b, callback)
		return
	}

	if strings.HasPrefix(data, "sub:") {
		s.handleSubtitleLanguageSelection(ctx, b, callback)
		return
	}

	if strings.HasPrefix(data, "lang:") {
		s.handleSubtitleTypeSelection(ctx, b, callback)
		return
	}

	if strings.HasPrefix(data, "dlsub:") {
		s.handleSubtitleDownloadTrigger(ctx, b, callback)
		return
	}

	if strings.HasPrefix(data, "setmodel:") {
		s.handleAIModelSelection(ctx, b, callback)
		return
	}

	if strings.HasPrefix(data, "web:") {
		s.handleWebCallback(ctx, b, callback)
		return
	}

	if strings.HasPrefix(data, "back:") {
		s.handleBackToFormats(ctx, b, callback)
		return
	}

	if strings.HasPrefix(data, "dl:") {
		parts := strings.Split(data, ":")
		if len(parts) < 3 {
			return
		}
		formatID := parts[1]
		urlHash := parts[2]

		videoURL, exists := s.getURL(urlHash)
		if !exists {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   "❌ Phiên tải xuống đã hết hạn. Vui lòng gửi lại liên kết video.",
			})
			b.DeleteMessage(ctx, &bot.DeleteMessageParams{
				ChatID:    chatID,
				MessageID: messageID,
			})
			return
		}

		// Find the selected FormatOption
		var selectedOption downloader.FormatOption
		found := false
		for _, opt := range downloader.AvailableFormats {
			if opt.ID == formatID {
				selectedOption = opt
				found = true
				break
			}
		}

		if !found {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   "❌ Định dạng không hợp lệ.",
			})
			return
		}

		// Replace selection menu with loading message
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: messageID,
			Text:      "⏳ Bắt đầu kết nối tới máy chủ...",
		})

		queueID := urlHash
		s.activeDownloadsMu.Lock()
		s.activeDownloads[queueID] = &QueueItem{
			ID:        queueID,
			UserID:    userID,
			Title:     "Đang kết nối & phân tích video...",
			URL:       videoURL,
			Progress:  0.0,
			StartedAt: time.Now(),
		}
		s.activeDownloadsMu.Unlock()

		s.LogInfo("Tạo tiến trình tải xuống %s cho user %d - URL: %s", queueID, userID, videoURL)

		// Track last updated percentage to avoid Telegram API rate limits (Too Many Requests 429)
		lastPercent := -1.0
		lastUpdateTime := time.Now()

		onProgress := func(percent float64) {
			s.activeDownloadsMu.Lock()
			if item, ok := s.activeDownloads[queueID]; ok {
				item.Progress = percent
				item.Title = "Đang tải video..."
			}
			s.activeDownloadsMu.Unlock()

			// Throttle progress updates: Update UI only if percent difference > 15% or at least 2 seconds passed
			if percent-lastPercent >= 15.0 || time.Since(lastUpdateTime) >= 2*time.Second {
				lastPercent = percent
				lastUpdateTime = time.Now()

				s.LogInfo("Tiến trình %s tải xuống: %.1f%%", queueID, percent)

				// Build dynamic visual progress bar
				barWidth := 10
				completed := int(math.Round(percent / 10.0))
				if completed > barWidth {
					completed = barWidth
				}
				progressBar := strings.Repeat("█", completed) + strings.Repeat("░", barWidth-completed)

				b.EditMessageText(ctx, &bot.EditMessageTextParams{
					ChatID:    chatID,
					MessageID: messageID,
					Text:      fmt.Sprintf("⏳ Đang tải file về máy chủ...\n\n<code>%s</code> %.1f%%", progressBar, percent),
					ParseMode: models.ParseModeHTML,
				})
			}
		}

		// Run Download
		result, err := s.dl.Download(ctx, videoURL, selectedOption, onProgress)

		s.activeDownloadsMu.Lock()
		delete(s.activeDownloads, queueID)
		s.activeDownloadsMu.Unlock()

		if err != nil {
			s.LogError("Tiến trình %s thất bại: %v", queueID, err)
			log.Printf("Download error for URL %s: %v", videoURL, err)
			b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    chatID,
				MessageID: messageID,
				Text:      "❌ Tải file thất bại. Vui lòng kiểm tra lại link hoặc thử định dạng khác.",
			})
			return
		}

		s.LogInfo("Tiến trình %s tải thành công về VPS. File: %s (%d bytes)", queueID, result.Title, result.FileSize)

		// Notify user that we are uploading
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: messageID,
			Text:      "📤 Đang tải file lên Telegram...",
		})

		// Determine platform name
		platform := "youtube"
		urlLower := strings.ToLower(videoURL)
		if strings.Contains(urlLower, "tiktok.com") || strings.Contains(urlLower, "douyin.com") {
			platform = "tiktok"
		} else if strings.Contains(urlLower, "instagram.com") {
			platform = "instagram"
		} else if strings.Contains(urlLower, "facebook.com") || strings.Contains(urlLower, "fb.watch") {
			platform = "facebook"
		} else if strings.Contains(urlLower, "twitter.com") || strings.Contains(urlLower, "x.com") {
			platform = "twitter"
		} else if strings.Contains(urlLower, "bilibili.com") {
			platform = "bilibili"
		}

		// Upload file to chat, store record, cache
		s.uploadAndSave(ctx, b, chatID, userID, videoURL, result.Title, platform, selectedOption.Extension, result.FilePath)

		// Delete the temporary status message
		b.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    chatID,
			MessageID: messageID,
		})
	}
}

func (s *BotServer) handleCutCallback(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery) {
	chatID := callback.Message.Message.Chat.ID
	messageID := callback.Message.Message.ID
	userID := callback.From.ID
	data := callback.Data

	urlHash := strings.TrimPrefix(data, "cut:")
	videoURL, exists := s.getURL(urlHash)
	if !exists {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Phiên làm việc đã hết hạn. Vui lòng gửi lại link video.",
		})
		b.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    chatID,
			MessageID: messageID,
		})
		return
	}

	// Đưa user vào trạng thái chờ nhập timestamp
	s.waitingForCutMu.Lock()
	s.waitingForCut[userID] = videoURL
	s.waitingForCutMu.Unlock()

	// Cập nhật tin nhắn hướng dẫn
	text := `✂️ <b>Chế độ Cắt Clip & Tạo GIF</b>

Vui lòng gửi khoảng thời gian bạn muốn cắt.
Định dạng: <code>phút:giây-phút:giây</code> (hoặc số giây trực tiếp).

Ví dụ:
• <code>0:10-0:40</code> (Cắt từ giây thứ 10 đến 40)
• <code>10-40</code> (Cắt từ giây thứ 10 đến 40)
• <code>1:20-2:10</code> (Cắt từ 1 phút 20s đến 2 phút 10s)

<i>Lưu ý: Thời lượng cắt tối đa là 60 giây.</i>`

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: messageID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
		ReplyMarkup: &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{
						Text:         "❌ Hủy",
						CallbackData: "cancel_cut",
					},
				},
			},
		},
	})
}

func (s *BotServer) handleSubtitleLanguageSelection(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery) {
	chatID := callback.Message.Message.Chat.ID
	messageID := callback.Message.Message.ID
	data := callback.Data

	urlHash := strings.TrimPrefix(data, "sub:")
	videoURL, exists := s.getURL(urlHash)
	if !exists {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Phiên làm việc đã hết hạn. Vui lòng gửi lại link video.",
		})
		b.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    chatID,
			MessageID: messageID,
		})
		return
	}

	// Show loading status
	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: messageID,
		Text:      "🔍 Đang kiểm tra danh sách phụ đề khả dụng từ video...",
	})

	info, err := s.dl.Probe(ctx, videoURL)
	if err != nil {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: messageID,
			Text:      "❌ Không thể phân tích thông tin phụ đề của video này.",
			ReplyMarkup: &models.InlineKeyboardMarkup{
				InlineKeyboard: [][]models.InlineKeyboardButton{
					{
						{
							Text:         "⬅️ Quay lại",
							CallbackData: fmt.Sprintf("back:%s", urlHash),
						},
					},
				},
			},
		})
		return
	}

	langs := info.GetAvailableLanguages()
	if len(langs) == 0 {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: messageID,
			Text:      "ℹ️ Video này không có phụ đề khả dụng (kể cả phụ đề tự dịch và tự động tạo).",
			ReplyMarkup: &models.InlineKeyboardMarkup{
				InlineKeyboard: [][]models.InlineKeyboardButton{
					{
						{
							Text:         "⬅️ Quay lại",
							CallbackData: fmt.Sprintf("back:%s", urlHash),
						},
					},
				},
			},
		})
		return
	}

	text := fmt.Sprintf("📝 <b>Danh sách Phụ đề khả dụng</b>\n🎬 %s\n\nVui lòng chọn ngôn ngữ phụ đề bạn muốn tải:", html.EscapeString(info.Title))
	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      chatID,
		MessageID:   messageID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: BuildSubLanguageKeyboard(urlHash, langs),
	})
}

func (s *BotServer) handleSubtitleTypeSelection(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery) {
	chatID := callback.Message.Message.Chat.ID
	messageID := callback.Message.Message.ID
	data := callback.Data

	// format: lang:[lang]:[urlHash]
	parts := strings.Split(data, ":")
	if len(parts) < 3 {
		return
	}
	lang := parts[1]
	urlHash := parts[2]

	videoURL, exists := s.getURL(urlHash)
	if !exists {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Phiên làm việc đã hết hạn. Vui lòng gửi lại link video.",
		})
		b.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    chatID,
			MessageID: messageID,
		})
		return
	}

	text := fmt.Sprintf("📝 <b>Tải Phụ đề ngôn ngữ: %s</b>\n🔗 %s\n\nBạn muốn tải phụ đề này như thế nào?", lang, videoURL)
	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      chatID,
		MessageID:   messageID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: BuildSubDownloadTypeKeyboard(urlHash, lang),
	})
}

func (s *BotServer) handleSubtitleDownloadTrigger(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery) {
	chatID := callback.Message.Message.Chat.ID
	messageID := callback.Message.Message.ID
	userID := callback.From.ID
	data := callback.Data

	// format: dlsub:[downloadType]:[lang]:[urlHash]
	parts := strings.Split(data, ":")
	if len(parts) < 4 {
		return
	}
	downloadType := parts[1]
	lang := parts[2]
	urlHash := parts[3]

	videoURL, exists := s.getURL(urlHash)
	if !exists {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Phiên làm việc đã hết hạn. Vui lòng gửi lại link video.",
		})
		b.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    chatID,
			MessageID: messageID,
		})
		return
	}

	// Delete choice keyboard to clear UI
	b.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    chatID,
		MessageID: messageID,
	})

	if downloadType == "summary" {
		go s.handleVideoSummary(ctx, b, chatID, userID, videoURL, lang)
		return
	}

	go s.handleSubtitleDownload(ctx, b, chatID, userID, videoURL, lang, downloadType)
}

func (s *BotServer) handleBackToFormats(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery) {
	chatID := callback.Message.Message.Chat.ID
	messageID := callback.Message.Message.ID
	data := callback.Data

	urlHash := strings.TrimPrefix(data, "back:")
	videoURL, exists := s.getURL(urlHash)
	if !exists {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Phiên làm việc đã hết hạn. Vui lòng gửi lại link video.",
		})
		b.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    chatID,
			MessageID: messageID,
		})
		return
	}

	// Reconstruct probe/format choosing interface
	info, err := s.dl.Probe(ctx, videoURL)
	if err != nil {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: messageID,
			Text:      "❌ Không thể tải lại thông tin video.",
		})
		return
	}

	sizeMB := 0.0
	// Try finding best format info size
	for _, f := range info.Formats {
		if f.Filesize > 0 {
			sizeMB = float64(f.Filesize) / (1024 * 1024)
			break
		}
	}

	durationStr := fmt.Sprintf("%.0f giây", info.Duration)
	if info.Duration >= 60 {
		durationStr = fmt.Sprintf("%.0f phút %.0f giây", info.Duration/60, float64(int(info.Duration)%60))
	}

	text := fmt.Sprintf("🎬 <b>%s</b>\n\n⏱ Thời lượng: %s\n📦 Dung lượng ước tính: %.2f MB\n\nChọn chất lượng hoặc định dạng bạn muốn tải xuống:",
		html.EscapeString(info.Title), durationStr, sizeMB)

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      chatID,
		MessageID:   messageID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: BuildFormatKeyboard(urlHash),
	})
}

func (s *BotServer) handleCompressSelection(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery) {
	chatID := callback.Message.Message.Chat.ID
	messageID := callback.Message.Message.ID
	data := callback.Data
	parts := strings.Split(data, ":")
	if len(parts) < 2 {
		return
	}
	urlHash := parts[1]

	_, exists := s.getURL(urlHash)
	if !exists {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Phiên tải xuống đã hết hạn. Vui lòng gửi lại liên kết video.",
		})
		b.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    chatID,
			MessageID: messageID,
		})
		return
	}

	_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      chatID,
		MessageID:   messageID,
		Text:        "🗜 <b>Nén & Hạ độ phân giải Video</b>\n\nVideo tải về sẽ được nén sang định dạng <b>H.264 MP4</b> tương thích 100% với Telegram.\nVui lòng chọn cấu hình nén:",
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: BuildCompressOptionsKeyboard(urlHash),
	})
}

func (s *BotServer) handleCompressDownloadTrigger(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery) {
	chatID := callback.Message.Message.Chat.ID
	messageID := callback.Message.Message.ID
	userID := callback.From.ID
	data := callback.Data
	parts := strings.Split(data, ":")
	if len(parts) < 3 {
		return
	}
	resolution := parts[1]
	urlHash := parts[2]

	videoURL, exists := s.getURL(urlHash)
	if !exists {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Phiên tải xuống đã hết hạn. Vui lòng gửi lại liên kết video.",
		})
		b.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    chatID,
			MessageID: messageID,
		})
		return
	}

	// Xoá tin nhắn menu
	_, _ = b.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    chatID,
		MessageID: messageID,
	})

	// Kích hoạt goroutine tải/nén
	go s.handleCompressDownload(ctx, b, chatID, userID, videoURL, resolution)
}

func (s *BotServer) handleAIModelSelection(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery) {
	chatID := callback.Message.Message.Chat.ID
	messageID := callback.Message.Message.ID
	data := callback.Data

	parts := strings.Split(data, ":")
	if len(parts) < 2 {
		return
	}
	modelHash := parts[1]

	modelID, exists := s.getURL(modelHash)
	if !exists {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Phiên làm việc đã hết hạn. Vui lòng gõ lại lệnh /ai_model.",
		})
		_, _ = b.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    chatID,
			MessageID: messageID,
		})
		return
	}

	// Load and Update Config
	cfg, err := s.db.GetAIConfig()
	if err != nil {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Không thể tải cấu hình AI để cập nhật.",
		})
		return
	}

	cfg.Model = modelID
	if err := s.db.SetAIConfig(cfg); err != nil {
		s.LogError("Failed to update AI model: %v", err)
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Không thể lưu model mới vào cơ sở dữ liệu.",
		})
		return
	}

	// Notify success
	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		ParseMode: models.ParseModeHTML,
		Text:      fmt.Sprintf("✅ Đã cập nhật Model AI hệ thống thành công!\nModel hoạt động: <code>%s</code>", html.EscapeString(modelID)),
	})

	// Delete choice keyboard
	_, _ = b.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    chatID,
		MessageID: messageID,
	})
}

func (s *BotServer) handleWebCallback(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery) {
	chatID := callback.Message.Message.Chat.ID
	messageID := callback.Message.Message.ID
	userID := callback.From.ID
	data := callback.Data

	// format: web:summary:[urlHash]
	parts := strings.Split(data, ":")
	if len(parts) < 3 {
		return
	}
	action := parts[1]
	urlHash := parts[2]

	targetURL, exists := s.getURL(urlHash)
	if !exists {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Phiên làm việc đã hết hạn. Vui lòng gửi lại link trang web.",
		})
		_, _ = b.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    chatID,
			MessageID: messageID,
		})
		return
	}

	if action == "summary" {
		// Xóa keyboard lựa chọn để tránh bấm lại
		_, _ = b.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    chatID,
			MessageID: messageID,
		})

		go s.handleWebSummary(ctx, b, chatID, userID, targetURL)
	}
}
