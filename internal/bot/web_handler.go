package bot

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"download-bot/internal/ai"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// handleWebSummary downloads the HTML page, cleans the content, and streams a summary using the AI client.
func (s *BotServer) handleWebSummary(ctx context.Context, b *bot.Bot, chatID int64, userID int64, targetURL string) {
	// 1. Send initial loading message
	statusMsg, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   "⏳ Đang cào nội dung bài viết từ liên kết...",
	})
	if err != nil {
		return
	}

	// 2. Fetch and Clean HTML content
	title, textContent, err := FetchAndCleanHTML(targetURL)
	if err != nil {
		s.LogError("Failed to crawl web URL %s: %v", targetURL, err)
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: statusMsg.ID,
			Text:      fmt.Sprintf("❌ Không thể cào dữ liệu từ trang web này: %v", err),
		})
		return
	}

	if len(textContent) < 50 {
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: statusMsg.ID,
			Text:      "⚠️ Không tìm thấy đủ nội dung văn bản hữu ích trên trang web này để tóm tắt.",
		})
		return
	}

	// 3. Load AI configuration
	cfg, err := s.GetActiveAIConfig()
	if err != nil || !cfg.Enabled || cfg.BaseURL == "" || cfg.APIKey == "" || cfg.Model == "" {
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: statusMsg.ID,
			Text:      "⚠️ Tính năng AI chưa được bật hoặc cấu hình chưa đầy đủ.",
		})
		return
	}

	// 4. Update status message
	_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: statusMsg.ID,
		Text:      "🤖 AI đang đọc hiểu và tóm tắt bài viết...",
	})

	// 5. Ask AI via stream
	systemPrompt := "Bạn là một trợ lý đọc hiểu và tóm tắt văn bản chuyên nghiệp. Hãy đọc nội dung bài viết dưới đây và viết một bản tóm tắt ngắn gọn, súc tích bằng tiếng Việt dạng gạch đầu dòng Markdown, nêu rõ các luận điểm chính và kết luận quan trọng của bài báo."
	history := []ai.Message{
		{
			Role:    "user",
			Content: fmt.Sprintf("Tiêu đề bài viết: %s\n\nNội dung chi tiết:\n%s", title, textContent),
		},
	}

	client := ai.NewClient(cfg.BaseURL, cfg.APIKey, cfg.Model)
	var fullReply string
	var lastEditTime time.Time

	err = client.ChatStream(ctx, systemPrompt, history, func(token string) {
		fullReply += token

		if time.Since(lastEditTime) > 1200*time.Millisecond {
			lastEditTime = time.Now()
			replyText := fmt.Sprintf("📰 <b>Tóm tắt Bài viết bằng AI</b>\n🎬 <i>%s</i>\n\n%s", html.EscapeString(title), html.EscapeString(fullReply))
			_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    chatID,
				MessageID: statusMsg.ID,
				Text:      replyText,
				ParseMode: models.ParseModeHTML,
			})
		}
	})

	if err != nil {
		s.LogError("AI Web Summarization failed: %v", err)
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: statusMsg.ID,
			Text:      fmt.Sprintf("❌ Lỗi AI khi tóm tắt bài viết: %v", err),
		})
		return
	}

	// 6. Final Message Update
	replyText := fmt.Sprintf("📰 <b>Tóm tắt Bài viết bằng AI</b>\n🎬 <i>%s</i>\n\n%s", html.EscapeString(title), html.EscapeString(fullReply))
	_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: statusMsg.ID,
		Text:      replyText,
		ParseMode: models.ParseModeHTML,
	})
}

// FetchAndCleanHTML sends a GET request to targetURL, extracts the page title and returns stripped clean text.
func FetchAndCleanHTML(urlStr string) (string, string, error) {
	client := &http.Client{
		Timeout: 12 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		return "", "", err
	}
	// Chrome User-Agent to prevent bots blocking
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("server returned status code %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	htmlContent := string(bodyBytes)

	// Extract Title
	title := "Bài viết"
	titleRegex := regexp.MustCompile(`(?i)<title[^>]*>(.*?)</title>`)
	if matches := titleRegex.FindStringSubmatch(htmlContent); len(matches) > 1 {
		title = strings.TrimSpace(matches[1])
	}

	// Stripping tags using regexes
	reScript := regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	htmlContent = reScript.ReplaceAllString(htmlContent, "")

	reStyle := regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	htmlContent = reStyle.ReplaceAllString(htmlContent, "")

	reHead := regexp.MustCompile(`(?is)<head[^>]*>.*?</head>`)
	htmlContent = reHead.ReplaceAllString(htmlContent, "")

	reNav := regexp.MustCompile(`(?is)<nav[^>]*>.*?</nav>`)
	htmlContent = reNav.ReplaceAllString(htmlContent, "")

	reFooter := regexp.MustCompile(`(?is)<footer[^>]*>.*?</footer>`)
	htmlContent = reFooter.ReplaceAllString(htmlContent, "")

	// Extract everything inside body if body exists, otherwise clean all
	bodyRegex := regexp.MustCompile(`(?is)<body[^>]*>(.*?)</body>`)
	if matches := bodyRegex.FindStringSubmatch(htmlContent); len(matches) > 1 {
		htmlContent = matches[1]
	}

	// Strip remaining HTML tags
	reTags := regexp.MustCompile(`<[^>]*>`)
	cleanText := reTags.ReplaceAllString(htmlContent, " ")

	// Unescape HTML entities (&amp;, &nbsp;, &quot;, &lt;, &gt;, etc)
	cleanText = html.UnescapeString(cleanText)

	// Clean whitespace lines
	lines := strings.Split(cleanText, "\n")
	var sb strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			sb.WriteString(trimmed)
			sb.WriteString("\n")
		}
	}
	result := sb.String()

	// Soft token limit (~25,000 words) to avoid OpenAI request overflow
	if len(result) > 120000 {
		result = result[:120000]
	}

	return title, result, nil
}
