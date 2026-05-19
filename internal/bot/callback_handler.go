package bot

import (
	"context"
	"download-bot/internal/downloader"
	"fmt"
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

	if strings.HasPrefix(data, "dl:") {
		parts := strings.Split(data, ":")
		if len(parts) < 3 {
			return
		}
		ext := parts[1]
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
			if opt.Extension == ext {
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
		if strings.Contains(strings.ToLower(videoURL), "tiktok.com") {
			platform = "tiktok"
		}

		// Upload file to chat, store record, cache
		s.uploadAndSave(ctx, b, chatID, userID, videoURL, result.Title, platform, ext, result.FilePath)

		// Delete the temporary status message
		b.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    chatID,
			MessageID: messageID,
		})
	}
}
