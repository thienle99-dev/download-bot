package downloader

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

type Downloader struct {
	downloadDir string
	sem         chan struct{}
}

func NewDownloader(downloadDir string, maxConcurrent int) *Downloader {
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		log.Printf("Failed to create download dir %s: %v", downloadDir, err)
	}
	return &Downloader{
		downloadDir: downloadDir,
		sem:         make(chan struct{}, maxConcurrent),
	}
}

// Probe uses yt-dlp --dump-json to gather video metadata without downloading.
func (d *Downloader) Probe(ctx context.Context, url string) (*VideoInfo, error) {
	// Simple validation to prevent command injection or unsafe URLs
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return nil, fmt.Errorf("invalid URL protocol")
	}

	cmd := exec.CommandContext(ctx, "yt-dlp", "--dump-json", "--no-warnings", "--no-playlist", url)
	output, err := cmd.Output()
	if err != nil {
		// Log full error for diagnostics
		if exitErr, ok := err.(*exec.ExitError); ok {
			log.Printf("yt-dlp probe exit error: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to probe URL: %w", err)
	}

	var info VideoInfo
	if err := json.Unmarshal(output, &info); err != nil {
		return nil, fmt.Errorf("failed to parse video info: %w", err)
	}

	return &info, nil
}

type DownloadResult struct {
	FilePath string
	Title    string
	Duration float64
	FileSize int64
}

// Download runs yt-dlp and writes output into the download directory.
func (d *Downloader) Download(ctx context.Context, url string, option FormatOption, onProgress func(percent float64)) (*DownloadResult, error) {
	// Wait for slot in concurrency semaphore
	select {
	case d.sem <- struct{}{}:
		defer func() { <-d.sem }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// First, probe to get exact title and metadata
	info, err := d.Probe(ctx, url)
	if err != nil {
		return nil, err
	}

	// Clean title to be safe for filenames
	safeTitle := cleanFilename(info.Title)
	if safeTitle == "" {
		safeTitle = "video_" + info.ID
	}

	tempTemplate := filepath.Join(d.downloadDir, fmt.Sprintf("%s.%%(ext)s", safeTitle))

	// Construct yt-dlp arguments
	args := []string{
		"--no-playlist",
		"--no-warnings",
		"--newline", // Output newline for progress parsing
	}

	if option.IsAudioOnly {
		args = append(args,
			"-f", option.YtDlpFormat,
			"--extract-audio",
			"--audio-format", option.AudioFormat,
			"--audio-quality", "0",
			"--embed-metadata",
			"--embed-thumbnail",
			"-o", tempTemplate,
		)
	} else {
		args = append(args,
			"-f", option.YtDlpFormat,
			"--merge-output-format", "mp4",
			"-o", tempTemplate,
		)
	}

	args = append(args, url)

	cmd := exec.CommandContext(ctx, "yt-dlp", args...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	defer stdoutPipe.Close()

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	defer stderrPipe.Close()

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start yt-dlp download: %w", err)
	}

	// Parse progress from stdout in goroutine
	percentRegex := regexp.MustCompile(`\[download\]\s+(\d+(\.\d+)?)%`)
	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			line := scanner.Text()
			matches := percentRegex.FindStringSubmatch(line)
			if len(matches) > 1 {
				if pct, err := strconv.ParseFloat(matches[1], 64); err == nil && onProgress != nil {
					onProgress(pct)
				}
			}
		}
	}()

	// Read stderr in background to log if needed
	var stderrOutput strings.Builder
	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			stderrOutput.WriteString(scanner.Text() + "\n")
		}
	}()

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("yt-dlp download failed: %s: %w", stderrOutput.String(), err)
	}

	// Find the output file
	expectedExt := "mp4"
	if option.IsAudioOnly {
		expectedExt = option.Extension
	}

	pattern := filepath.Join(d.downloadDir, safeTitle+".*")
	matches, _ := filepath.Glob(pattern)
	var finalPath string
	for _, m := range matches {
		ext := strings.ToLower(filepath.Ext(m))
		if ext == "."+expectedExt {
			finalPath = m
			break
		}
	}

	// If no perfect match, just pick first matching format or try base ext
	if finalPath == "" && len(matches) > 0 {
		finalPath = matches[0]
	}

	if finalPath == "" {
		return nil, fmt.Errorf("could not find downloaded file in %s", d.downloadDir)
	}

	stat, err := os.Stat(finalPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat downloaded file: %w", err)
	}

	return &DownloadResult{
		FilePath: finalPath,
		Title:    info.Title,
		Duration: info.Duration,
		FileSize: stat.Size(),
	}, nil
}

// DownloadSubtitle runs yt-dlp to download and convert the subtitle to .srt.
func (d *Downloader) DownloadSubtitle(ctx context.Context, url string, lang string) (string, error) {
	select {
	case d.sem <- struct{}{}:
		defer func() { <-d.sem }()
	case <-ctx.Done():
		return "", ctx.Err()
	}

	info, err := d.Probe(ctx, url)
	if err != nil {
		return "", err
	}

	safeTitle := cleanFilename(info.Title)
	if safeTitle == "" {
		safeTitle = "sub_" + info.ID
	}

	outputTemplate := filepath.Join(d.downloadDir, fmt.Sprintf("%s_%s.%%(ext)s", safeTitle, lang))

	args := []string{
		"--no-playlist",
		"--no-warnings",
		"--skip-download",
		"--write-subs",
		"--write-auto-subs",
		"--sub-lang", lang,
		"--sub-format", "srt/vtt/ass/best",
		"--convert-subs", "srt",
		"-o", outputTemplate,
		url,
	}

	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to download subtitle: %w", err)
	}

	expectedFile := filepath.Join(d.downloadDir, fmt.Sprintf("%s_%s.srt", safeTitle, lang))
	if _, err := os.Stat(expectedFile); err == nil {
		return expectedFile, nil
	}

	pattern := filepath.Join(d.downloadDir, fmt.Sprintf("%s_%s.*", safeTitle, lang))
	matches, _ := filepath.Glob(pattern)
	if len(matches) > 0 {
		return matches[0], nil
	}

	return "", fmt.Errorf("subtitle file not found after download")
}

func removeAccents(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, _ := transform.String(t, s)
	result = strings.ReplaceAll(result, "đ", "d")
	result = strings.ReplaceAll(result, "Đ", "D")
	return result
}

func cleanFilename(name string) string {
	name = removeAccents(name)
	// Strip characters that are unsafe or problematic across systems/terminals
	reg := regexp.MustCompile(`[^a-zA-Z0-9.-]+`)
	cleaned := reg.ReplaceAllString(name, " ")
	cleaned = strings.TrimSpace(cleaned)
	// Replace spaces with underscores
	cleaned = strings.ReplaceAll(cleaned, " ", "_")

	// Trim length to avoid filesystem path limit issues
	if len(cleaned) > 100 {
		cleaned = cleaned[:100]
	}
	return strings.Trim(cleaned, "_")
}

// CompressVideo performs H.264 video compression and optional scaling using ffmpeg
func (d *Downloader) CompressVideo(ctx context.Context, inputPath string, outputPath string, resolution string) error {
	inputPath = filepath.Clean(inputPath)
	outputPath = filepath.Clean(outputPath)

	var scaleFilter string
	var crf string

	switch resolution {
	case "720":
		scaleFilter = "scale='min(1280,iw)':-2"
		crf = "24"
	case "480":
		scaleFilter = "scale='min(854,iw)':-2"
		crf = "26"
	case "same":
		// Ensure width/height are even numbers (trunc(iw/2)*2) to avoid libx264 scaling errors
		scaleFilter = "scale='trunc(iw/2)*2':'trunc(ih/2)*2'"
		crf = "24"
	default:
		return fmt.Errorf("unsupported compression resolution: %s", resolution)
	}

	args := []string{
		"-i", inputPath,
		"-vf", scaleFilter,
		"-c:v", "libx264",
		"-preset", "fast",
		"-crf", crf,
		"-c:a", "aac",
		"-b:a", "128k",
		"-y", outputPath,
	}

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg compression failed: %s: %w", string(output), err)
	}

	return nil
}

