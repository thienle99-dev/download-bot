package bot

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (s *BotServer) handleInlineQuery(ctx context.Context, b *bot.Bot, query *models.InlineQuery) {
	queryText := strings.TrimSpace(query.Query)

	// If query is empty or not a valid video URL, return no results or help card
	if queryText == "" || !s.isValidURL(queryText) {
		return
	}

	// Probe URL quickly to get title
	info, err := s.dl.Probe(ctx, queryText)
	if err != nil {
		log.Printf("Inline query probe failed for %s: %v", queryText, err)
		return
	}

	title := info.Title
	if len(title) > 50 {
		title = title[:47] + "..."
	}

	// Build options as inline results
	// 1. Download Video Result
	videoResult := &models.InlineQueryResultArticle{
		ID:          "dl_video",
		Title:       fmt.Sprintf("🎬 Tải Video: %s", title),
		Description: "Gửi lệnh tải video đầy đủ chất lượng",
		InputMessageContent: &models.InputTextMessageContent{
			MessageText: fmt.Sprintf("%s", queryText),
		},
	}

	// 2. Download MP3 Result
	audioResult := &models.InlineQueryResultArticle{
		ID:          "dl_audio",
		Title:       fmt.Sprintf("🎵 Tải MP3: %s", title),
		Description: "Gửi link và tự động convert sang file nhạc MP3",
		InputMessageContent: &models.InputTextMessageContent{
			MessageText: fmt.Sprintf("%s", queryText),
		},
	}

	params := &bot.AnswerInlineQueryParams{
		InlineQueryID: query.ID,
		Results:       []models.InlineQueryResult{videoResult, audioResult},
		CacheTime:     300, // Cache results for 5 minutes
	}

	_, err = b.AnswerInlineQuery(ctx, params)
	if err != nil {
		log.Printf("Failed to answer inline query: %v", err)
	}
}
