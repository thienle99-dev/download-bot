package bot

import (
	"bufio"
	"context"
	"fmt"
	"html"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"download-bot/internal/ai"
	"download-bot/internal/downloader"
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
	extractorLower := strings.ToLower(info.Extractor)
	if strings.Contains(extractorLower, "tiktok") || strings.Contains(extractorLower, "douyin") {
		platform = "TikTok/Douyin"
	} else if strings.Contains(extractorLower, "instagram") {
		platform = "Instagram"
	} else if strings.Contains(extractorLower, "facebook") {
		platform = "Facebook"
	} else if strings.Contains(extractorLower, "twitter") || strings.Contains(extractorLower, "x") {
		platform = "Twitter/X"
	} else if strings.Contains(extractorLower, "bilibili") {
		platform = "Bilibili"
	} else if info.Extractor != "" {
		platform = strings.Title(info.Extractor)
	}

	// Send selection prompt
	title := info.Title
	if len(title) > 60 {
		title = title[:57] + "..."
	}

	durationMin := int(info.Duration) / 60
	durationSec := int(info.Duration) % 60

	text := fmt.Sprintf("🎬 <b>%s</b>\n⏱ Thời lượng: <code>%02d:%02d</code> | Nền tảng: <code>%s</code>\n\nChọn chất lượng tải về hoặc chuyển đổi sang MP3:",
		html.EscapeString(title), durationMin, durationSec, platform)

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
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
			Text:      fmt.Sprintf("⚡ Video này đã được tải trước đó. Bạn có thể tải file tốc độ cao trực tiếp tại đây:\n🔗 <a href=\"%s\">Nhấn vào để tải xuống</a>", fileID),
			ParseMode: models.ParseModeHTML,
		})
		return
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   "⚡ Video này đã được tải trước đó. Đang gửi lại ngay lập tức...",
	})

	if format == "mp3" || format == "m4a" || format == "flac" {
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
		publicLink := fmt.Sprintf("%s/videos/downloads/%s", s.cfg.PublicURL, encodedName)

		s.LogInfo("Tệp quá giới hạn 50MB (%.2f MB). Phát hành liên kết trực tiếp tốc độ cao: %s", sizeMB, publicLink)

		caption := fmt.Sprintf("⚠️ <b>File vượt quá giới hạn 50MB của Telegram!</b> (%.2f MB)\n🎬 <b>%s</b>\n\n🚀 Bạn có thể tải file tốc độ cao trực tiếp từ link máy chủ tại đây:\n🔗 <a href=\"%s\">Nhấn vào để tải xuống</a>",
			sizeMB, html.EscapeString(title), publicLink)

		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    chatID,
			Text:      caption,
			ParseMode: models.ParseModeHTML,
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

	caption := fmt.Sprintf("✅ <b>Tải thành công!</b>\n🎬 %s\n📦 Dung lượng: %.2f MB",
		html.EscapeString(title), sizeMB)

	s.LogInfo("Đang tải tệp %s (%.2f MB) trực tiếp lên đám mây Telegram...", format, sizeMB)

	// Send info/title message first
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      caption,
		ParseMode: models.ParseModeHTML,
	})
	if err != nil {
		log.Printf("Failed to send download completion notification: %v", err)
	}

	var fileID string

	if format == "mp3" || format == "m4a" || format == "flac" {
		msg, err := b.SendAudio(ctx, &bot.SendAudioParams{
			ChatID: chatID,
			Audio: &models.InputFileUpload{
				Filename: filepath.Base(localPath),
				Data:     file,
			},
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
			ChatID: chatID,
			Video: &models.InputFileUpload{
				Filename: filepath.Base(localPath),
				Data:     file,
			},
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

	// Close file handle explicitly to allow deletion on all systems
	_ = file.Close()

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

		// Delete local file to save VPS storage since Telegram already has the file cloud cached
		if err := os.Remove(localPath); err != nil {
			log.Printf("Failed to remove downloaded local file %s: %v", localPath, err)
		} else {
			s.LogInfo("Đã xóa file vật lý cục bộ %s sau khi gửi thành công lên Telegram Cloud.", filepath.Base(localPath))
		}
	}
}

func (s *BotServer) handleCutProcess(ctx context.Context, b *bot.Bot, msg *models.Message, videoURL string, rangeStr string) {
	chatID := msg.Chat.ID
	userID := msg.From.ID

	// 1. Send processing status
	statusMsg, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   "🔍 Đang kiểm tra khoảng thời gian và phân tích video...",
	})
	if err != nil {
		log.Printf("Failed to send cut status message: %v", err)
		return
	}

	// 2. Parse range
	start, end, err := downloader.ParseRange(rangeStr)
	if err != nil {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: statusMsg.ID,
			Text:      fmt.Sprintf("❌ Lỗi: %v\nVui lòng thử lại với định dạng đúng, ví dụ: <code>10-40</code> hoặc <code>0:10-0:40</code>", err),
			ParseMode: models.ParseModeHTML,
		})
		return
	}

	duration := end - start
	if duration > 60 {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: statusMsg.ID,
			Text:      fmt.Sprintf("⚠️ Thời lượng cắt yêu cầu là %.0f giây. Chỉ hỗ trợ cắt tối đa 60 giây để tránh quá tải máy chủ.", duration),
		})
		return
	}

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: statusMsg.ID,
		Text:      fmt.Sprintf("⏳ Đang tải và cắt phân đoạn từ %.0fs đến %.0fs (độ dài: %.0fs)...", start, end, duration),
	})

	// 3. Download and slice section
	result, err := s.dl.DownloadSection(ctx, videoURL, start, end)
	if err != nil {
		s.LogError("Cắt video thất bại: %v", err)
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: statusMsg.ID,
			Text:      "❌ Quá trình tải và cắt phân đoạn thất bại. Vui lòng kiểm tra lại liên kết hoặc thử lại sau.",
		})
		return
	}
	defer os.Remove(result.FilePath) // Cleanup MP4 after processing

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: statusMsg.ID,
		Text:      "🎞 Đang chuyển đổi thành ảnh động GIF...",
	})

	// 4. Convert to GIF
	gifPath := strings.TrimSuffix(result.FilePath, ".mp4") + ".gif"
	err = downloader.ConvertToGIF(ctx, result.FilePath, gifPath)
	if err != nil {
		s.LogError("Convert GIF thất bại: %v", err)
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: statusMsg.ID,
			Text:      "❌ Lỗi chuyển đổi file GIF, đang thử gửi lại clip MP4...",
		})
	}
	defer os.Remove(gifPath) // Cleanup GIF after sending

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: statusMsg.ID,
		Text:      "📤 Đang gửi clip MP4 và ảnh động GIF lên Telegram...",
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

	// 5. Send MP4 and GIF
	mp4File, err := os.Open(result.FilePath)
	if err != nil {
		log.Printf("Failed to open cut MP4 for sending: %v", err)
		return
	}
	defer mp4File.Close()

	// Gửi tin nhắn text thông tin riêng biệt để user dễ forward video sạch
	infoText := fmt.Sprintf("✅ <b>Cắt phân đoạn thành công!</b>\n🎬 %s\n⏱ Khoảng: <code>%.0f-%.0fs</code> (độ dài: <code>%.0fs</code>)",
		html.EscapeString(result.Title), start, end, duration)

	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      infoText,
		ParseMode: models.ParseModeHTML,
	})

	// Gửi Video
	var videoMsg *models.Message
	videoMsg, err = b.SendVideo(ctx, &bot.SendVideoParams{
		ChatID: chatID,
		Video: &models.InputFileUpload{
			Filename: filepath.Base(result.FilePath),
			Data:     mp4File,
		},
	})
	if err != nil {
		s.LogError("Gửi clip MP4 cắt thất bại: %v", err)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Không thể gửi clip MP4 qua Telegram.",
		})
	}

	// Gửi GIF (Telegram Animation)
	if _, err := os.Stat(gifPath); err == nil {
		gifFile, err := os.Open(gifPath)
		if err == nil {
			defer gifFile.Close()
			_, err = b.SendAnimation(ctx, &bot.SendAnimationParams{
				ChatID: chatID,
				Animation: &models.InputFileUpload{
					Filename: filepath.Base(gifPath),
					Data:     gifFile,
				},
			})
			if err != nil {
				s.LogError("Gửi ảnh động GIF thất bại: %v", err)
			}
		}
	}

	// Xóa tin nhắn trạng thái
	b.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    chatID,
		MessageID: statusMsg.ID,
	})

	// Lưu lịch sử tải
	if videoMsg != nil && videoMsg.Video != nil {
		history := &storage.DownloadHistory{
			UserID:   userID,
			ChatID:   chatID,
			URL:      videoURL,
			Platform: platform,
			Title:    result.Title,
			Format:   "mp4",
			FileSize: result.FileSize,
			FilePath: result.FilePath,
			FileID:   videoMsg.Video.FileID,
		}
		if err := s.db.SaveDownload(history); err != nil {
			log.Printf("Failed to save cut history: %v", err)
		}
	}
}

func (s *BotServer) handleSubtitleDownload(ctx context.Context, b *bot.Bot, chatID int64, userID int64, videoURL string, lang string, downloadType string) {
	// 1. Send status message
	statusMsg, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   fmt.Sprintf("⏳ Đang tải phụ đề [%s]...", lang),
	})
	if err != nil {
		log.Printf("Failed to send subtitle status: %v", err)
		return
	}

	// 2. Download Subtitle (.srt)
	subPath, err := s.dl.DownloadSubtitle(ctx, videoURL, lang)
	if err != nil {
		s.LogError("Tải phụ đề [%s] thất bại: %v", lang, err)
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: statusMsg.ID,
			Text:      fmt.Sprintf("❌ Tải phụ đề [%s] thất bại. Có thể video không cung cấp phụ đề này.", lang),
		})
		return
	}
	defer os.Remove(subPath) // Cleanup srt file from VPS after function returns

	// Helper to send SRT file
	sendSRT := func() bool {
		file, err := os.Open(subPath)
		if err != nil {
			log.Printf("Failed to open SRT file: %v", err)
			return false
		}
		defer file.Close()

		_, err = b.SendDocument(ctx, &bot.SendDocumentParams{
			ChatID: chatID,
			Document: &models.InputFileUpload{
				Filename: filepath.Base(subPath),
				Data:     file,
			},
			Caption: fmt.Sprintf("📝 Phụ đề [%s] của video.", lang),
		})
		if err != nil {
			s.LogError("Gửi file phụ đề thất bại: %v", err)
			return false
		}
		return true
	}

	if downloadType == "srt" {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: statusMsg.ID,
			Text:      "📤 Đang gửi file phụ đề lên Telegram...",
		})

		if sendSRT() {
			b.DeleteMessage(ctx, &bot.DeleteMessageParams{
				ChatID:    chatID,
				MessageID: statusMsg.ID,
			})
		} else {
			b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    chatID,
				MessageID: statusMsg.ID,
				Text:      "❌ Không thể gửi file phụ đề .srt lên Telegram.",
			})
		}
		return
	}

	// Case: both (Video + Subtitle)
	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: statusMsg.ID,
		Text:      "⏳ Đang tải file video chất lượng tốt nhất...",
	})

	// Download Video format option "best"
	bestFormat := downloader.AvailableFormats[0] // ID: "best"
	result, err := s.dl.Download(ctx, videoURL, bestFormat, func(percent float64) {
		// Update progress info periodically if needed
	})
	if err != nil {
		s.LogError("Tải video thất bại trong tiến trình subtitle: %v", err)
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: statusMsg.ID,
			Text:      "❌ Tải video thất bại. Tuy nhiên bạn có thể thử chỉ tải phụ đề riêng.",
		})
		return
	}
	defer os.Remove(result.FilePath) // Cleanup video after sending

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: statusMsg.ID,
		Text:      "📤 Đang gửi video và phụ đề lên Telegram...",
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
	}

	// 1. Upload Video & Save to Database / Cache
	s.uploadAndSave(ctx, b, chatID, userID, videoURL, result.Title, platform, bestFormat.Extension, result.FilePath)

	// 2. Upload Subtitle file
	sendSRT()

	// Cleanup the status message
	b.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    chatID,
		MessageID: statusMsg.ID,
	})
}

// handleCompressDownload handles download, compression (resolution & bitrate scale) and uploading the compressed video.
func (s *BotServer) handleCompressDownload(ctx context.Context, b *bot.Bot, chatID int64, userID int64, videoURL string, resolution string) {
	// Send initial status message
	statusMsg, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   "⏳ Đang tải video gốc chất lượng tốt nhất...",
	})
	if err != nil {
		s.LogError("Send status message failed: %v", err)
		return
	}

	cleanup := func() {
		_, _ = b.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    chatID,
			MessageID: statusMsg.ID,
		})
	}
	defer cleanup()

	// 1. Download original video (using best available format option)
	bestFormat := downloader.AvailableFormats[0]
	result, err := s.dl.Download(ctx, videoURL, bestFormat, nil)
	if err != nil {
		s.LogError("Download original video failed: %v", err)
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Tải video gốc thất bại. Vui lòng thử lại sau.",
		})
		return
	}
	defer os.Remove(result.FilePath)

	// 2. Define output compressed file path
	dir := filepath.Dir(result.FilePath)
	ext := filepath.Ext(result.FilePath)
	base := strings.TrimSuffix(filepath.Base(result.FilePath), ext)
	compressedPath := filepath.Join(dir, fmt.Sprintf("%s_compressed_%s.mp4", base, resolution))

	// Edit status message
	resLabel := resolution + "p"
	if resolution == "same" {
		resLabel = "độ phân giải gốc"
	}
	_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: statusMsg.ID,
		Text:      fmt.Sprintf("⚙️ Đang nén video sang định dạng %s (H.264)...", resLabel),
	})

	// 3. Compress video via ffmpeg
	if err := s.dl.CompressVideo(ctx, result.FilePath, compressedPath, resolution); err != nil {
		s.LogError("CompressVideo failed: %v", err)
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Nén video thất bại. Vui lòng thử lại sau.",
		})
		return
	}
	defer os.Remove(compressedPath)

	// Edit status message
	_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: statusMsg.ID,
		Text:      "📤 Đang gửi video đã nén lên Telegram...",
	})

	// Determine platform
	platform := "youtube"
	urlLower := strings.ToLower(videoURL)
	if strings.Contains(urlLower, "tiktok.com") || strings.Contains(urlLower, "douyin.com") {
		platform = "tiktok"
	} else if strings.Contains(urlLower, "instagram.com") {
		platform = "instagram"
	} else if strings.Contains(urlLower, "facebook.com") || strings.Contains(urlLower, "fb.watch") {
		platform = "facebook"
	}

	// 4. Upload and Save to Database & cache
	title := result.Title
	if resolution != "same" {
		title = fmt.Sprintf("%s (%sp)", title, resolution)
	} else {
		title = fmt.Sprintf("%s (Compressed)", title)
	}

	s.uploadAndSave(ctx, b, chatID, userID, videoURL, title, platform, "mp4", compressedPath)
}

// handleVideoSummary downloads the subtitles, cleans them, and sends a summary prompt to AI.
func (s *BotServer) handleVideoSummary(ctx context.Context, b *bot.Bot, chatID int64, userID int64, videoURL string, lang string) {
	// 1. Send status message
	statusMsg, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   fmt.Sprintf("⏳ Đang tải phụ đề [%s] để phân tích...", lang),
	})
	if err != nil {
		return
	}

	// 2. Download Subtitle (.srt)
	subPath, err := s.dl.DownloadSubtitle(ctx, videoURL, lang)
	if err != nil {
		s.LogError("Tải phụ đề cho tóm tắt thất bại: %v", err)
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: statusMsg.ID,
			Text:      fmt.Sprintf("❌ Tải phụ đề [%s] thất bại. Có thể video không có phụ đề để tóm tắt.", lang),
		})
		return
	}
	defer os.Remove(subPath)

	// 3. Clean SRT to plain text
	textTranscript, err := cleanSRT(subPath)
	if err != nil || textTranscript == "" {
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: statusMsg.ID,
			Text:      "❌ Lỗi khi đọc và làm sạch phụ đề video.",
		})
		return
	}

	// 4. Load AI Config
	cfg, err := s.GetActiveAIConfig()
	if err != nil || !cfg.Enabled || cfg.BaseURL == "" || cfg.APIKey == "" || cfg.Model == "" {
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: statusMsg.ID,
			Text:      "⚠️ Tính năng AI chưa được bật hoặc cấu hình chưa đầy đủ.",
		})
		return
	}

	// 5. Update Status
	_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: statusMsg.ID,
		Text:      "🤖 AI đang phân tích nội dung phụ đề và tóm tắt...",
	})

	// 6. Build prompt and ask AI (with streaming)
	summaryPrompt := "Bạn là một trợ lý tóm tắt video chuyên nghiệp. Hãy đọc bản phụ đề sau và viết bản tóm tắt các nội dung cốt lõi, các mốc quan trọng và bài học chính của video. Trình bày ngắn gọn, súc tích bằng tiếng Việt dạng gạch đầu dòng Markdown."
	history := []ai.Message{
		{
			Role:    "user",
			Content: fmt.Sprintf("Nội dung phụ đề video cần tóm tắt:\n\n%s", textTranscript),
		},
	}

	client := ai.NewClient(cfg.BaseURL, cfg.APIKey, cfg.Model)
	var fullReply string
	var lastEditTime time.Time

	err = client.ChatStream(ctx, summaryPrompt, history, func(token string) {
		fullReply += token
		if time.Since(lastEditTime) > 1200*time.Millisecond {
			lastEditTime = time.Now()
			replyText := fmt.Sprintf("📝 <b>Tóm tắt Video bằng AI</b>\n\n%s", html.EscapeString(fullReply))
			_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    chatID,
				MessageID: statusMsg.ID,
				Text:      replyText,
				ParseMode: models.ParseModeHTML,
			})
		}
	})

	if err != nil {
		s.LogError("Tóm tắt bằng AI thất bại: %v", err)
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: statusMsg.ID,
			Text:      fmt.Sprintf("❌ Lỗi AI tóm tắt: %v", err),
		})
		return
	}

	// Final update
	replyText := fmt.Sprintf("📝 <b>Tóm tắt Video bằng AI</b>\n\n%s", html.EscapeString(fullReply))
	_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: statusMsg.ID,
		Text:      replyText,
		ParseMode: models.ParseModeHTML,
	})
}

// cleanSRT reads an SRT file and extracts clean plain text by stripping numbers, timestamps, and formatting.
func cleanSRT(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var sb strings.Builder
	scanner := bufio.NewScanner(file)

	// Precompile regexes
	timestampRegex := regexp.MustCompile(`\d{2}:\d{2}:\d{2}`)
	numberRegex := regexp.MustCompile(`^\d+$`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// Skip line numbers
		if numberRegex.MatchString(line) {
			continue
		}
		// Skip timestamps
		if strings.Contains(line, "-->") || timestampRegex.MatchString(line) {
			continue
		}

		sb.WriteString(line)
		sb.WriteString(" ")

		// Hard limit to prevent token overflow (~30,000 words)
		if sb.Len() > 150000 {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return strings.TrimSpace(sb.String()), nil
}
