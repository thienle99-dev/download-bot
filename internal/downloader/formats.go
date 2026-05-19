package downloader

type FormatOption struct {
	ID          string // Unique ID: "best", "1080p", etc.
	Label       string // UI Label: "🎬 1080p", "🎬 720p", etc.
	YtDlpFormat string // Format string passed to -f flag
	Extension   string // Preferred extension ("mp4", "mp3", "m4a", "flac")
	IsAudioOnly bool   // Whether to convert to Audio
	AudioFormat string // Audio format for yt-dlp like "mp3", "m4a", "flac"
}

var AvailableFormats = []FormatOption{
	{
		ID:          "best",
		Label:       "🎬 Best (Gốc)",
		YtDlpFormat: "bestvideo+bestaudio/best",
		Extension:   "mp4",
		IsAudioOnly: false,
	},
	{
		ID:          "1080p",
		Label:       "🎬 1080p",
		YtDlpFormat: "bestvideo[height<=1080]+bestaudio/best[height<=1080]",
		Extension:   "mp4",
		IsAudioOnly: false,
	},
	{
		ID:          "720p",
		Label:       "🎬 720p",
		YtDlpFormat: "bestvideo[height<=720]+bestaudio/best[height<=720]",
		Extension:   "mp4",
		IsAudioOnly: false,
	},
	{
		ID:          "480p",
		Label:       "🎬 480p",
		YtDlpFormat: "bestvideo[height<=480]+bestaudio/best[height<=480]",
		Extension:   "mp4",
		IsAudioOnly: false,
	},
	{
		ID:          "mp3",
		Label:       "🎵 MP3 Audio (320kbps)",
		YtDlpFormat: "bestaudio/best",
		Extension:   "mp3",
		IsAudioOnly: true,
		AudioFormat: "mp3",
	},
	{
		ID:          "m4a",
		Label:       "🎵 M4A Audio (AAC)",
		YtDlpFormat: "bestaudio/best",
		Extension:   "m4a",
		IsAudioOnly: true,
		AudioFormat: "m4a",
	},
	{
		ID:          "flac",
		Label:       "🎵 FLAC Audio (Lossless)",
		YtDlpFormat: "bestaudio/best",
		Extension:   "flac",
		IsAudioOnly: true,
		AudioFormat: "flac",
	},
}

type FormatInfo struct {
	FormatID   string `json:"format_id"`
	Extension  string `json:"ext"`
	Resolution string `json:"resolution"`
	Filesize   int64  `json:"filesize"`
	Height     int    `json:"height"`
}

type VideoInfo struct {
	ID                string                 `json:"id"`
	Title             string                 `json:"title"`
	Duration          float64                `json:"duration"`
	Thumbnail         string                 `json:"thumbnail"`
	Extractor         string                 `json:"extractor"` // "youtube", "tiktok", etc.
	WebpageURL        string                 `json:"webpage_url"`
	Formats           []FormatInfo           `json:"formats"`
	Subtitles         map[string]interface{} `json:"subtitles"`
	AutomaticCaptions map[string]interface{} `json:"automatic_captions"`
}

// GetAvailableLanguages returns unique language codes found in subtitles and automatic_captions.
// It prioritizes standard popular languages like vi, en, ja, ko, zh, th.
func (v *VideoInfo) GetAvailableLanguages() []string {
	langMap := make(map[string]bool)

	// Gather from manual subtitles
	for lang := range v.Subtitles {
		langMap[lang] = true
	}

	// Gather from automatic captions
	for lang := range v.AutomaticCaptions {
		langMap[lang] = true
	}

	if len(langMap) == 0 {
		return nil
	}

	// Popular languages we want to prioritize
	popular := []string{"vi", "en", "ja", "ko", "zh", "th"}
	var result []string

	// Add popular languages first if available
	for _, p := range popular {
		if langMap[p] {
			result = append(result, p)
			delete(langMap, p)
		}
	}

	// Add the rest
	for lang := range langMap {
		result = append(result, lang)
	}

	// Limit to maximum 12 languages to keep Telegram UI clean
	if len(result) > 12 {
		result = result[:12]
	}

	return result
}

