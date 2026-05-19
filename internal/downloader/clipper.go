package downloader

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ParseTimestamp parses hh:mm:ss, mm:ss, or ss into seconds.
func ParseTimestamp(ts string) (float64, error) {
	ts = strings.TrimSpace(ts)
	parts := strings.Split(ts, ":")
	if len(parts) == 1 {
		return strconv.ParseFloat(parts[0], 64)
	}
	if len(parts) == 2 {
		min, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return 0, err
		}
		sec, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return 0, err
		}
		return min*60 + sec, nil
	}
	if len(parts) == 3 {
		hr, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return 0, err
		}
		min, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return 0, err
		}
		sec, err := strconv.ParseFloat(parts[2], 64)
		if err != nil {
			return 0, err
		}
		return hr*3600 + min*60 + sec, nil
	}
	return 0, fmt.Errorf("invalid timestamp format")
}

// ParseRange parses a "start-end" string into start and end seconds.
func ParseRange(rangeStr string) (float64, float64, error) {
	rangeStr = strings.TrimSpace(rangeStr)
	parts := strings.Split(rangeStr, "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("range must be in start-end format (e.g. 10-30 or 0:10-0:40)")
	}
	start, err := ParseTimestamp(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid start time: %w", err)
	}
	end, err := ParseTimestamp(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid end time: %w", err)
	}
	if start < 0 || end < 0 {
		return 0, 0, fmt.Errorf("timestamps cannot be negative")
	}
	if end <= start {
		return 0, 0, fmt.Errorf("end time must be greater than start time")
	}
	return start, end, nil
}

// DownloadSection runs yt-dlp to download only the specified video portion.
// If direct section download fails, it falls back to downloading the full video and cutting it via ffmpeg.
func (d *Downloader) DownloadSection(ctx context.Context, url string, start, end float64) (*DownloadResult, error) {
	// First, probe to get exact title and metadata
	info, err := d.Probe(ctx, url)
	if err != nil {
		return nil, err
	}

	safeTitle := cleanFilename(info.Title)
	if safeTitle == "" {
		safeTitle = "video_" + info.ID
	}
	// Append a clip suffix to avoid caching collision with full video
	clipTitle := fmt.Sprintf("%s_clip_%.0f_%.0f", safeTitle, start, end)
	outputPath := filepath.Join(d.downloadDir, clipTitle+".mp4")

	// Ensure concurrency control
	select {
	case d.sem <- struct{}{}:
		defer func() { <-d.sem }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Try Direct Section Download
	log.Printf("[Clipper] Attempting direct section download: %.2f to %.2f", start, end)
	err = d.downloadSectionDirect(ctx, url, start, end, outputPath)
	if err == nil {
		// Double check file exists and is not empty
		if stat, err := os.Stat(outputPath); err == nil && stat.Size() > 0 {
			log.Printf("[Clipper] Direct section download succeeded!")
			return &DownloadResult{
				FilePath: outputPath,
				Title:    info.Title + fmt.Sprintf(" (Clip %.0f-%.0fs)", start, end),
				Duration: end - start,
				FileSize: stat.Size(),
			}, nil
		}
	}

	log.Printf("[Clipper] Direct section download failed or not supported (%v). Falling back to full download and ffmpeg slice...", err)

	// Fallback: Download full video
	// Construct a temporary full video path
	tempFullTemplate := filepath.Join(d.downloadDir, fmt.Sprintf("temp_full_%s_%.0f.%%(ext)s", safeTitle, float64(time.Now().UnixNano())))

	args := []string{
		"--no-playlist",
		"--no-warnings",
		"-f", "bestvideo[ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]/best",
		"--merge-output-format", "mp4",
		"-o", tempFullTemplate,
		url,
	}

	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("fallback full download failed: %w", err)
	}

	// Find the temp downloaded file
	pattern := filepath.Join(d.downloadDir, fmt.Sprintf("temp_full_%s_*.*", safeTitle))
	matches, _ := filepath.Glob(pattern)
	if len(matches) == 0 {
		return nil, fmt.Errorf("could not find downloaded full video for slicing")
	}
	tempFullFile := matches[0]
	defer os.Remove(tempFullFile) // Always cleanup the huge full file

	// Cut the file using ffmpeg with re-encoding to ensure frame precision
	duration := end - start
	ffmpegArgs := []string{
		"-y",
		"-ss", fmt.Sprintf("%.3f", start),
		"-i", tempFullFile,
		"-t", fmt.Sprintf("%.3f", duration),
		"-c:v", "libx264",
		"-c:a", "aac",
		"-preset", "superfast",
		"-crf", "22",
		outputPath,
	}

	ffmpegCmd := exec.CommandContext(ctx, "ffmpeg", ffmpegArgs...)
	if err := ffmpegCmd.Run(); err != nil {
		// Fallback to -c copy if re-encode fails
		log.Printf("[Clipper] Re-encode slice failed, trying -c copy: %v", err)
		ffmpegArgsCopy := []string{
			"-y",
			"-ss", fmt.Sprintf("%.3f", start),
			"-i", tempFullFile,
			"-t", fmt.Sprintf("%.3f", duration),
			"-c", "copy",
			outputPath,
		}
		if errCopy := exec.CommandContext(ctx, "ffmpeg", ffmpegArgsCopy...).Run(); errCopy != nil {
			return nil, fmt.Errorf("ffmpeg slicing failed: %w", errCopy)
		}
	}

	stat, err := os.Stat(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat sliced video: %w", err)
	}

	return &DownloadResult{
		FilePath: outputPath,
		Title:    info.Title + fmt.Sprintf(" (Clip %.0f-%.0fs)", start, end),
		Duration: duration,
		FileSize: stat.Size(),
	}, nil
}

func (d *Downloader) downloadSectionDirect(ctx context.Context, url string, start, end float64, outputPath string) error {
	sectionArg := fmt.Sprintf("*%.3f-%.3f", start, end)
	args := []string{
		"--no-playlist",
		"--no-warnings",
		"-f", "bestvideo[ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]/best",
		"--merge-output-format", "mp4",
		"--download-sections", sectionArg,
		"-o", outputPath,
		url,
	}
	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	return cmd.Run()
}

// ConvertToGIF converts a local MP4 file to a high-quality GIF using ffmpeg palette.
func ConvertToGIF(ctx context.Context, mp4Path, gifPath string) error {
	filter := "fps=12,scale=480:-1:flags=lanczos,split[s0][s1];[s0]palettegen[p];[s1][p]paletteuse"
	args := []string{
		"-y",
		"-i", mp4Path,
		"-vf", filter,
		"-loop", "0",
		gifPath,
	}
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	return cmd.Run()
}
