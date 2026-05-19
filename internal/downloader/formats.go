package downloader

type FormatOption struct {
	Label        string // UI Label: "🎬 1080p", "🎬 720p", etc.
	YtDlpFormat  string // Format string passed to -f flag
	Extension    string // Preferred extension
	IsAudioOnly  bool   // Whether to convert to MP3
}

var AvailableFormats = []FormatOption{
	{
		Label:        "🎬 Best (Gốc)",
		YtDlpFormat:  "bestvideo+bestaudio/best",
		Extension:    "mp4",
		IsAudioOnly:  false,
	},
	{
		Label:        "🎬 1080p",
		YtDlpFormat:  "bestvideo[height<=1080]+bestaudio/best[height<=1080]",
		Extension:    "mp4",
		IsAudioOnly:  false,
	},
	{
		Label:        "🎬 720p",
		YtDlpFormat:  "bestvideo[height<=720]+bestaudio/best[height<=720]",
		Extension:    "mp4",
		IsAudioOnly:  false,
	},
	{
		Label:        "🎬 480p",
		YtDlpFormat:  "bestvideo[height<=480]+bestaudio/best[height<=480]",
		Extension:    "mp4",
		IsAudioOnly:  false,
	},
	{
		Label:        "🎵 MP3 Audio",
		YtDlpFormat:  "bestaudio/best",
		Extension:    "mp3",
		IsAudioOnly:  true,
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
	ID          string       `json:"id"`
	Title       string       `json:"title"`
	Duration    float64      `json:"duration"`
	Thumbnail   string       `json:"thumbnail"`
	Extractor   string       `json:"extractor"` // "youtube", "tiktok", etc.
	WebpageURL  string       `json:"webpage_url"`
	Formats     []FormatInfo `json:"formats"`
}
