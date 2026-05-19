package bot

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"download-bot/internal/storage"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// promptFormatSelection probes the URL and offers format selection keyboards.
// If the URL is already cached, it skips downloading and resends the file instantly.
func (s *BotServer) promptFormatSelection(ctx context.Context, b *bot.Bot, msg *models.Message, videoURL string) {
	chatID := msg.Chat.ID
	userID := msg.From.ID

	// 1. Check if the URL is cached (in-memory or in database)
	if cachedID, found := s.cache.Get(userID, videoURL); found {
		s.resendCachedFile(ctx, b, chatID, cachedID, "video")
		return
	}

	recent, err := s.db.GetRecentByURL(userID, videoURL)
	if err == nil && recent != nil && recent.FileID != "" {
		// Try resending using Telegram file_id
		s.resendCachedFile(ctx, b, chatID, recent.FileID, recent.Format)
		// Cache it in memory too
		s.cache.Add(userID, videoURL, recent.FilePath, recent.FileID)
		return
	}

	// Send temporary status
	statusMsg, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   "🔍 Đang kiểm tra liên kết và định dạng khả dụng...",
	})
	if err != nil {
		log.Printf("Failed to send probe status message: %v", err)
		return
	}

	// 2. Probe URL to get metadata
	info, err := s.dl.Probe(ctx, videoURL)
	if err != nil {
		log.Printf("Failed to probe video URL %s: %v", videoURL, err)
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: statusMsg.ID,
			Text:      "❌ Không thể kiểm tra liên kết này. Vui lòng đảm bảo link hợp lệ hoặc thử lại sau.",
		})
		return
	}

	// Delete temporary status message
	b.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    chatID,
		MessageID: statusMsg.ID,
	})

	// 3. Register URL and get a short hash to bypass Telegram's 64-char callback limit
	urlHash := s.registerURL(videoURL)

	// Format platform name nicely
	platform := "YouTube"
	if strings.Contains(strings.ToLower(info.Extractor), "tiktok") {
		platform = "TikTok"
	}

	// Send selection prompt
	title := info.Title
	if len(title) > 60 {
		title = title[:57] + "..."
	}

	durationMin := int(info.Duration) / 60
	durationSec := int(info.Duration) % 60

	text := fmt.Sprintf("🎬 *%s*\n⏱ Thời lượng: `%02d:%02d` | Nền tảng: `%s`\n\nChọn chất lượng tải về hoặc chuyển đổi sang MP3:",
		bot.EscapeMarkdown(title), durationMin, durationSec, platform)

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: BuildFormatKeyboard(urlHash),
	})
	if err != nil {
		log.Printf("Failed to send format selection keyboard: %v", err)
	}
}

// resendCachedFile sends a video/audio using its Telegram file_id (instant forward) or public HTTP link
func (s *BotServer) resendCachedFile(ctx context.Context, b *bot.Bot, chatID int64, fileID string, format string) {
	if strings.HasPrefix(fileID, "http://") || strings.HasPrefix(fileID, "https://") {
		// It's a public download link!
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    chatID,
			Text:      fmt.Sprintf("⚡ Video này đã được tải trước đó. Bạn có thể tải file tốc độ cao trực tiếp tại đây:\n🔗 [Nhấn vào để tải xuống](%s)", fileID),
			ParseMode: models.ParseModeMarkdown,
		})
		return
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   "⚡ Video này đã được tải trước đó. Đang gửi lại ngay lập tức...",
	})

	if format == "mp3" {
		_, err := b.SendAudio(ctx, &bot.SendAudioParams{
			ChatID: chatID,
			Audio:  &models.InputFileString{Data: fileID},
		})
		if err != nil {
			log.Printf("Failed to resend cached audio fileID: %v", err)
		}
	} else {
		_, err := b.SendVideo(ctx, &bot.SendVideoParams{
			ChatID: chatID,
			Video:  &models.InputFileString{Data: fileID},
		})
		if err != nil {
			log.Printf("Failed to resend cached video fileID: %v", err)
		}
	}
}

// uploadAndSave sends the downloaded file to Telegram, saves its file_id to SQLite & FileCache.
// If the file exceeds 50MB, it serves it as an HTTP download link.
func (s *BotServer) uploadAndSave(ctx context.Context, b *bot.Bot, chatID int64, userID int64, videoURL string, title string, platform string, format string, localPath string) {
	file, err := os.Open(localPath)
	if err != nil {
		log.Printf("Failed to open downloaded file for upload: %v", err)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Không thể mở file đã tải trên máy chủ.",
		})
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		log.Printf("Failed to stat downloaded file: %v", err)
		return
	}

	sizeMB := float64(stat.Size()) / (1024 * 1024)

	// Telegram standard API limit is 50MB
	if stat.Size() > 50*1024*1024 {
		fileName := filepath.Base(localPath)
		// URL encode filename for web safety
		encodedName := url.PathEscape(fileName)
		publicLink := fmt.Sprintf("%s/files/%s", s.cfg.PublicURL, encodedName)

		s.LogInfo("Tệp quá giới hạn 50MB (%.2f MB). Phát hành liên kết trực tiếp tốc độ cao: %s", sizeMB, publicLink)

		caption := fmt.Sprintf("⚠️ *File vượt quá giới hạn 50MB của Telegram!* (%.2f MB)\n🎬 *%s*\n\n🚀 Bạn có thể tải file tốc độ cao trực tiếp từ link máy chủ tại đây:\n🔗 [Nhấn vào để tải xuống](%s)",
			sizeMB, bot.EscapeMarkdown(title), publicLink)

		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    chatID,
			Text:      caption,
			ParseMode: models.ParseModeMarkdown,
		})
		if err != nil {
			s.LogError("Gửi link tải trực tiếp thất bại: %v", err)
			log.Printf("Failed to send large file download link: %v", err)
			return
		}

		// Save the link in place of fileID to reuse it easily!
		fileID := publicLink

		history := &storage.DownloadHistory{
			UserID:   userID,
			ChatID:   chatID,
			URL:      videoURL,
			Platform: platform,
			Title:    title,
			Format:   format,
			FileSize: stat.Size(),
			FilePath: localPath,
			FileID:   fileID,
		}
		if err := s.db.SaveDownload(history); err != nil {
			s.LogError("Lưu lịch sử database thất bại: %v", err)
			log.Printf("Failed to save download history to DB: %v", err)
		}

		// Save to per-user LRU file cache
		s.cache.Add(userID, videoURL, localPath, fileID)
		return
	}

	caption := fmt.Sprintf("✅ *Tải thành công!*\n🎬 %s\n📦 Dung lượng: %.2f MB",
		bot.EscapeMarkdown(title), sizeMB)

	s.LogInfo("Đang tải tệp %s (%.2f MB) trực tiếp lên đám mây Telegram...", format, sizeMB)

	var fileID string

	if format == "mp3" {
		msg, err := b.SendAudio(ctx, &bot.SendAudioParams{
			ChatID:  chatID,
			Caption: caption,
			Audio: &models.InputFileUpload{
				Filename: filepath.Base(localPath),
				Data:     file,
			},
			ParseMode: models.ParseModeMarkdown,
		})
		if err != nil {
			s.LogError("Upload file audio thất bại: %v", err)
			log.Printf("Failed to upload audio file: %v", err)
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   "❌ Gặp lỗi khi upload file audio lên Telegram.",
			})
			return
		}
		if msg.Audio != nil {
			fileID = msg.Audio.FileID
		}
	} else {
		msg, err := b.SendVideo(ctx, &bot.SendVideoParams{
			ChatID:  chatID,
			Caption: caption,
			Video: &models.InputFileUpload{
				Filename: filepath.Base(localPath),
				Data:     file,
			},
			ParseMode: models.ParseModeMarkdown,
		})
		if err != nil {
			s.LogError("Upload file video thất bại: %v", err)
			log.Printf("Failed to upload video file: %v", err)
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   "❌ Gặp lỗi khi upload file video lên Telegram.",
			})
			return
		}
		if msg.Video != nil {
			fileID = msg.Video.FileID
		}
	}

	if fileID != "" {
		s.LogInfo("Upload thành công. Telegram FileID: %s", fileID)

		// Save database history
		history := &storage.DownloadHistory{
			UserID:   userID,
			ChatID:   chatID,
			URL:      videoURL,
			Platform: platform,
			Title:    title,
			Format:   format,
			FileSize: stat.Size(),
			FilePath: localPath,
			FileID:   fileID,
		}
		if err := s.db.SaveDownload(history); err != nil {
			s.LogError("Lưu lịch sử database thất bại: %v", err)
			log.Printf("Failed to save download history to DB: %v", err)
		}

		// Save to per-user LRU file cache
		s.cache.Add(userID, videoURL, localPath, fileID)
	}
}
