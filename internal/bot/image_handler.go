package bot

import (
	"context"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"download-bot/internal/imgproc"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type ImageSession struct {
	UserID    int64
	ChatID    int64
	Photos    []string
	MessageID int
	CreatedAt time.Time
	Timer     *time.Timer
	Mu        sync.Mutex
}

func (s *BotServer) handlePhoto(ctx context.Context, b *bot.Bot, msg *models.Message) {
	userID := msg.From.ID
	chatID := msg.Chat.ID

	if len(msg.Photo) == 0 {
		return
	}

	// Get largest photo size
	photo := msg.Photo[len(msg.Photo)-1]

	// Create directory for temp files
	userTempDir := filepath.Join(s.cfg.DownloadDir, "img_tmp", fmt.Sprintf("%d", userID))
	if err := os.MkdirAll(userTempDir, 0755); err != nil {
		log.Printf("Failed to create user temp dir: %v", err)
		return
	}

	// Retrieve file path from Telegram
	tgFile, err := b.GetFile(ctx, &bot.GetFileParams{
		FileID: photo.FileID,
	})
	if err != nil {
		log.Printf("GetFile failed for photo: %v", err)
		return
	}

	// Download the photo
	localPath := filepath.Join(userTempDir, filepath.Base(tgFile.FilePath))
	if err := s.downloadTelegramFile(ctx, tgFile.FilePath, localPath); err != nil {
		log.Printf("Failed to download telegram file: %v", err)
		return
	}

	s.imageSessionsMu.Lock()
	sess, exists := s.imageSessions[userID]
	if !exists {
		sess = &ImageSession{
			UserID:    userID,
			ChatID:    chatID,
			Photos:    []string{localPath},
			CreatedAt: time.Now(),
		}
		s.imageSessions[userID] = sess
	} else {
		sess.Mu.Lock()
		sess.Photos = append(sess.Photos, localPath)
		sess.Mu.Unlock()
	}
	s.imageSessionsMu.Unlock()

	// Reset timer to wait for more photos in the media group / album
	sess.Mu.Lock()
	if sess.Timer != nil {
		sess.Timer.Stop()
	}
	sess.Timer = time.AfterFunc(800*time.Millisecond, func() {
		s.promptImageAction(ctx, b, userID)
	})
	sess.Mu.Unlock()
}

func (s *BotServer) downloadTelegramFile(ctx context.Context, filePath string, localPath string) error {
	var downloadURL string
	if s.cfg.APIURL != "" {
		downloadURL = fmt.Sprintf("%s/file/bot%s/%s", strings.TrimSuffix(s.cfg.APIURL, "/"), s.cfg.BotToken, filePath)
	} else {
		downloadURL = fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", s.cfg.BotToken, filePath)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status code from Telegram API: %d", resp.StatusCode)
	}

	out, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func (s *BotServer) promptImageAction(ctx context.Context, b *bot.Bot, userID int64) {
	s.imageSessionsMu.RLock()
	sess, exists := s.imageSessions[userID]
	s.imageSessionsMu.RUnlock()

	if !exists {
		return
	}

	sess.Mu.Lock()
	photoCount := len(sess.Photos)
	chatID := sess.ChatID
	messageID := sess.MessageID
	sess.Mu.Unlock()

	if photoCount > 10 {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "⚠️ Chỉ hỗ trợ xử lý tối đa 10 ảnh trong một lần gửi. Vui lòng gửi ít ảnh hơn.",
		})
		s.cleanupImageSession(userID)
		return
	}

	text := fmt.Sprintf("📸 Đã nhận <b>%d ảnh</b> từ bạn.\n\nChọn thao tác nén hoặc chuyển đổi định dạng ảnh:", photoCount)

	if messageID == 0 {
		msg, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      chatID,
			Text:        text,
			ParseMode:   models.ParseModeHTML,
			ReplyMarkup: BuildImageKeyboard(),
		})
		if err == nil {
			sess.Mu.Lock()
			sess.MessageID = msg.ID
			sess.Mu.Unlock()
		} else {
			log.Printf("Failed to send image keyboard: %v", err)
		}
	} else {
		_, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:      chatID,
			MessageID:   messageID,
			Text:        text,
			ParseMode:   models.ParseModeHTML,
			ReplyMarkup: BuildImageKeyboard(),
		})
		if err != nil {
			log.Printf("Failed to edit image keyboard: %v", err)
		}
	}
}

func (s *BotServer) handleImageCallback(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery) {
	userID := callback.From.ID
	chatID := callback.Message.Message.Chat.ID
	messageID := callback.Message.Message.ID
	data := callback.Data

	s.imageSessionsMu.RLock()
	sess, exists := s.imageSessions[userID]
	s.imageSessionsMu.RUnlock()

	// Answer callback to stop loading spinner in user interface
	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: callback.ID,
	})

	if !exists {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Phiên xử lý ảnh đã hết hạn. Vui lòng gửi lại ảnh mới.",
		})
		_, _ = b.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    chatID,
			MessageID: messageID,
		})
		return
	}

	if data == "img:cancel" {
		_, _ = b.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    chatID,
			MessageID: messageID,
		})
		s.cleanupImageSession(userID)
		return
	}

	if data == "img:sticker" {
		s.handleStickerCallback(ctx, b, callback)
		return
	}
	if data == "img:st_new" {
		s.handleStickerNewCallback(ctx, b, callback)
		return
	}
	if strings.HasPrefix(data, "img:st_add:") {
		setName := strings.TrimPrefix(data, "img:st_add:")
		s.handleStickerAddCallback(ctx, b, callback, setName)
		return
	}
	if data == "img:st_back" {
		s.handleStickerBackCallback(ctx, b, callback)
		return
	}

	parts := strings.Split(data, ":")
	if len(parts) < 3 {
		return
	}
	action := parts[1]
	formatStr := parts[2]

	var format imgproc.ImageFormat
	var quality int

	switch formatStr {
	case "jpeg":
		format = imgproc.FormatJPEG
		if action == "compress" {
			quality = 75
		} else {
			quality = 95
		}
	case "png":
		format = imgproc.FormatPNG
		quality = 90
	case "webp":
		format = imgproc.FormatWEBP
		quality = 80
	default:
		return
	}

	sess.Mu.Lock()
	photos := make([]string, len(sess.Photos))
	copy(photos, sess.Photos)
	sess.Mu.Unlock()

	_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: messageID,
		Text:      fmt.Sprintf("⏳ Đang xử lý %d ảnh bằng ffmpeg...", len(photos)),
	})

	outDir := filepath.Join(s.cfg.DownloadDir, "img_tmp", fmt.Sprintf("%d_out", userID))
	if err := os.MkdirAll(outDir, 0755); err != nil {
		s.LogError("Tạo thư mục output ảnh thất bại: %v", err)
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: messageID,
			Text:      "❌ Lỗi máy chủ khi chuẩn bị thư mục xử lý ảnh.",
		})
		s.cleanupImageSession(userID)
		return
	}

	var processedFiles []string
	opt := imgproc.ProcessOption{
		Format:  format,
		Quality: quality,
	}

	for _, photo := range photos {
		outPath, err := imgproc.ProcessImage(ctx, photo, outDir, opt)
		if err != nil {
			log.Printf("ProcessImage failed for %s: %v", photo, err)
			continue
		}
		processedFiles = append(processedFiles, outPath)
	}

	if len(processedFiles) == 0 {
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: messageID,
			Text:      "❌ Không xử lý được bức ảnh nào. Vui lòng thử lại.",
		})
		s.cleanupImageSession(userID)
		return
	}

	// Pack ZIP file
	zipPath := filepath.Join(s.cfg.DownloadDir, "img_tmp", fmt.Sprintf("images_%d.zip", userID))
	_ = os.Remove(zipPath) // Clean up old one

	_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: messageID,
		Text:      "🗜 Đang đóng gói file ZIP...",
	})

	if err := imgproc.CreateZip(zipPath, processedFiles); err != nil {
		s.LogError("Tạo file ZIP thất bại: %v", err)
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: messageID,
			Text:      "❌ Lỗi máy chủ khi đóng gói ZIP.",
		})
		s.cleanupImageSession(userID)
		return
	}

	zipFile, err := os.Open(zipPath)
	if err != nil {
		log.Printf("Failed to open zip file: %v", err)
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: messageID,
			Text:      "❌ Không thể mở file ZIP đã tạo.",
		})
		s.cleanupImageSession(userID)
		return
	}
	defer zipFile.Close()

	_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: messageID,
		Text:      "📤 Đang gửi file ZIP qua Telegram...",
	})

	_, err = b.SendDocument(ctx, &bot.SendDocumentParams{
		ChatID: chatID,
		Document: &models.InputFileUpload{
			Filename: "images_processed.zip",
			Data:     zipFile,
		},
		Caption: fmt.Sprintf("✅ Đã xử lý xong %d/%d ảnh của bạn!", len(processedFiles), len(photos)),
	})
	if err != nil {
		log.Printf("SendDocument failed: %v", err)
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Không thể gửi file ZIP cho bạn qua Telegram.",
		})
	}

	_, _ = b.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    chatID,
		MessageID: messageID,
	})

	s.cleanupImageSession(userID)
}

func (s *BotServer) cleanupImageSession(userID int64) {
	s.imageSessionsMu.Lock()
	sess, exists := s.imageSessions[userID]
	if exists {
		delete(s.imageSessions, userID)
	}
	s.imageSessionsMu.Unlock()

	if sess != nil {
		sess.Mu.Lock()
		if sess.Timer != nil {
			sess.Timer.Stop()
		}
		sess.Mu.Unlock()
	}

	// Run cleanup in background
	go func() {
		userTempDir := filepath.Join(s.cfg.DownloadDir, "img_tmp", fmt.Sprintf("%d", userID))
		_ = os.RemoveAll(userTempDir)
		outDir := filepath.Join(s.cfg.DownloadDir, "img_tmp", fmt.Sprintf("%d_out", userID))
		_ = os.RemoveAll(outDir)
		zipPath := filepath.Join(s.cfg.DownloadDir, "img_tmp", fmt.Sprintf("images_%d.zip", userID))
		_ = os.Remove(zipPath)
	}()
}

func (s *BotServer) StartSessionCleaner(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.imageSessionsMu.Lock()
			now := time.Now()
			var toCleanup []int64
			for uID, sess := range s.imageSessions {
				if now.Sub(sess.CreatedAt) > 5*time.Minute {
					toCleanup = append(toCleanup, uID)
				}
			}
			s.imageSessionsMu.Unlock()

			for _, uID := range toCleanup {
				s.cleanupImageSession(uID)
			}
		}
	}
}

func (s *BotServer) handleStickerCallback(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery) {
	userID := callback.From.ID
	chatID := callback.Message.Message.Chat.ID
	messageID := callback.Message.Message.ID

	// Check if user already has sticker sets registered in SQLite
	sets, err := s.db.GetUserStickerSets(userID)
	if err != nil {
		s.LogError("Get user sticker sets failed: %v", err)
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Đã xảy ra lỗi khi kiểm tra các bộ Sticker của bạn. Vui lòng thử lại sau.",
		})
		s.cleanupImageSession(userID)
		return
	}

	if len(sets) == 0 {
		// User has no sets yet, automatically route to creating a new set
		s.handleStickerNewCallback(ctx, b, callback)
		return
	}

	// User has sets, prompt them to choose
	text := fmt.Sprintf("✨ <b>Tạo Sticker</b>\n\nBạn đã có <b>%d bộ Sticker</b> được tạo qua bot.\nBạn muốn tạo một bộ mới hay thêm ảnh vừa gửi vào bộ hiện có?", len(sets))
	_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      chatID,
		MessageID:   messageID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: BuildStickerActionKeyboard(sets),
	})
}

func (s *BotServer) handleStickerNewCallback(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery) {
	userID := callback.From.ID
	chatID := callback.Message.Message.Chat.ID
	messageID := callback.Message.Message.ID

	s.imageSessionsMu.RLock()
	sess, exists := s.imageSessions[userID]
	s.imageSessionsMu.RUnlock()

	if !exists {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Phiên xử lý ảnh đã hết hạn. Vui lòng gửi lại ảnh mới.",
		})
		return
	}

	sess.Mu.Lock()
	photos := make([]string, len(sess.Photos))
	copy(photos, sess.Photos)
	sess.Mu.Unlock()

	if len(photos) == 0 {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Không tìm thấy ảnh nào để tạo Sticker.",
		})
		s.cleanupImageSession(userID)
		return
	}

	// Lấy ảnh cuối cùng gửi lên để làm sticker
	photoPath := photos[len(photos)-1]

	_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: messageID,
		Text:      "⏳ Đang khởi tạo bộ Sticker mới của bạn...",
	})

	userTempDir := filepath.Join(s.cfg.DownloadDir, "img_tmp", fmt.Sprintf("%d", userID))
	stickerPath := filepath.Join(userTempDir, fmt.Sprintf("sticker_%d.webp", time.Now().UnixNano()))

	if err := imgproc.ProcessSticker(ctx, photoPath, stickerPath); err != nil {
		s.LogError("ProcessSticker failed: %v", err)
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: messageID,
			Text:      "❌ Không thể xử lý ảnh thành chuẩn Sticker Telegram.",
		})
		s.cleanupImageSession(userID)
		return
	}

	me, err := b.GetMe(ctx)
	if err != nil {
		s.LogError("GetMe failed: %v", err)
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: messageID,
			Text:      "❌ Lỗi kết nối với Telegram API.",
		})
		s.cleanupImageSession(userID)
		return
	}
	botUsername := me.Username

	// Generate safe, unique sticker set name ending with _by_<bot_username>
	setName := fmt.Sprintf("st_%d_%d_by_%s", userID, time.Now().Unix(), botUsername)

	title := callback.From.FirstName
	if title == "" {
		title = "Bạn"
	}
	setSubTitle := fmt.Sprintf("Sticker Pack của %s #%d", title, time.Now().Unix()%1000)

	file, err := os.Open(stickerPath)
	if err != nil {
		s.LogError("Open sticker file failed: %v", err)
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: messageID,
			Text:      "❌ Lỗi mở tệp sticker trên máy chủ.",
		})
		s.cleanupImageSession(userID)
		return
	}
	defer file.Close()

	_, err = b.CreateNewStickerSet(ctx, &bot.CreateNewStickerSetParams{
		UserID: userID,
		Name:   setName,
		Title:  setSubTitle,
		Stickers: []models.InputSticker{
			{
				Sticker:           "attach://sticker_file",
				Format:            "static",
				EmojiList:         []string{"✨"},
				StickerAttachment: file,
			},
		},
	})
	if err != nil {
		s.LogError("CreateNewStickerSet failed: %v", err)
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: messageID,
			Text:      "❌ Không thể tạo bộ Sticker trên Telegram. Hãy thử lại sau.",
		})
		s.cleanupImageSession(userID)
		return
	}

	// Lưu thông tin set vào SQLite
	if err := s.db.SaveStickerSet(userID, setName, setSubTitle); err != nil {
		s.LogError("SaveStickerSet DB failed: %v", err)
	}

	link := fmt.Sprintf("https://t.me/addstickers/%s", setName)
	successText := fmt.Sprintf("✅ <b>Đã tạo thành công bộ Sticker mới!</b>\n\n📌 Tiêu đề: <code>%s</code>\n\n🔗 Nhấn vào liên kết dưới đây để thêm bộ Sticker vào Telegram của bạn:\n👉 %s", html.EscapeString(setSubTitle), link)

	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      successText,
		ParseMode: models.ParseModeHTML,
	})

	_, _ = b.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    chatID,
		MessageID: messageID,
	})

	s.cleanupImageSession(userID)
}

func (s *BotServer) handleStickerAddCallback(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, setName string) {
	userID := callback.From.ID
	chatID := callback.Message.Message.Chat.ID
	messageID := callback.Message.Message.ID

	s.imageSessionsMu.RLock()
	sess, exists := s.imageSessions[userID]
	s.imageSessionsMu.RUnlock()

	if !exists {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Phiên xử lý ảnh đã hết hạn. Vui lòng gửi lại ảnh mới.",
		})
		return
	}

	sess.Mu.Lock()
	photos := make([]string, len(sess.Photos))
	copy(photos, sess.Photos)
	sess.Mu.Unlock()

	if len(photos) == 0 {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Không tìm thấy ảnh nào để làm Sticker.",
		})
		s.cleanupImageSession(userID)
		return
	}

	// Lấy ảnh cuối cùng gửi lên
	photoPath := photos[len(photos)-1]

	_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: messageID,
		Text:      "⏳ Đang thêm sticker mới vào bộ của bạn...",
	})

	userTempDir := filepath.Join(s.cfg.DownloadDir, "img_tmp", fmt.Sprintf("%d", userID))
	stickerPath := filepath.Join(userTempDir, fmt.Sprintf("sticker_%d.webp", time.Now().UnixNano()))

	if err := imgproc.ProcessSticker(ctx, photoPath, stickerPath); err != nil {
		s.LogError("ProcessSticker failed: %v", err)
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: messageID,
			Text:      "❌ Không thể xử lý ảnh thành chuẩn Sticker Telegram.",
		})
		s.cleanupImageSession(userID)
		return
	}

	file, err := os.Open(stickerPath)
	if err != nil {
		s.LogError("Open sticker file failed: %v", err)
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: messageID,
			Text:      "❌ Lỗi mở tệp sticker trên máy chủ.",
		})
		s.cleanupImageSession(userID)
		return
	}
	defer file.Close()

	_, err = b.AddStickerToSet(ctx, &bot.AddStickerToSetParams{
		UserID: userID,
		Name:   setName,
		Sticker: models.InputSticker{
			Sticker:           "attach://sticker_file",
			Format:            "static",
			EmojiList:         []string{"✨"},
			StickerAttachment: file,
		},
	})
	if err != nil {
		s.LogError("AddStickerToSet failed: %v", err)
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: messageID,
			Text:      "❌ Không thể thêm Sticker vào bộ này. Vui lòng kiểm tra lại giới hạn bộ Sticker (tối đa 120 cái).",
		})
		s.cleanupImageSession(userID)
		return
	}

	link := fmt.Sprintf("https://t.me/addstickers/%s", setName)
	successText := fmt.Sprintf("✅ <b>Đã thêm Sticker mới thành công!</b>\n\n🔗 Nhấn vào liên kết dưới đây để cập nhật/xem bộ Sticker của bạn:\n👉 %s", link)

	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      successText,
		ParseMode: models.ParseModeHTML,
	})

	_, _ = b.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    chatID,
		MessageID: messageID,
	})

	s.cleanupImageSession(userID)
}

func (s *BotServer) handleStickerBackCallback(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery) {
	userID := callback.From.ID
	chatID := callback.Message.Message.Chat.ID
	messageID := callback.Message.Message.ID

	s.imageSessionsMu.RLock()
	sess, exists := s.imageSessions[userID]
	s.imageSessionsMu.RUnlock()

	if !exists {
		return
	}

	sess.Mu.Lock()
	photoCount := len(sess.Photos)
	sess.Mu.Unlock()

	text := fmt.Sprintf("📸 Đã nhận <b>%d ảnh</b> từ bạn.\n\nChọn thao tác nén hoặc chuyển đổi định dạng ảnh:", photoCount)
	_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      chatID,
		MessageID:   messageID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: BuildImageKeyboard(),
	})
}
