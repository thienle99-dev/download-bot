package imgproc

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type ImageFormat string

const (
	FormatJPEG ImageFormat = "jpeg"
	FormatPNG  ImageFormat = "png"
	FormatWEBP ImageFormat = "webp"
)

type ProcessOption struct {
	Format  ImageFormat
	Quality int // Quality scale from 1 to 100
}

// ProcessImage calls ffmpeg to convert/compress an image file
func ProcessImage(ctx context.Context, inputPath string, outputDir string, opt ProcessOption) (string, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	ext := "." + string(opt.Format)
	if opt.Format == FormatJPEG {
		ext = ".jpg"
	}

	baseName := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	outputPath := filepath.Join(outputDir, baseName+ext)

	// Clean inputs to avoid commands issues (though filepath.Clean is safe)
	inputPath = filepath.Clean(inputPath)
	outputPath = filepath.Clean(outputPath)

	// ffmpeg command arguments
	args := []string{"-i", inputPath}

	switch opt.Format {
	case FormatJPEG:
		// Map quality 1-100 to ffmpeg -q:v quality (1-31, lower is better)
		// 100 quality -> -q:v 1
		// 75 quality -> -q:v 5
		// 50 quality -> -q:v 10
		// 1 quality -> -q:v 31
		qScale := 31 - int(float64(opt.Quality)*30.0/100.0)
		if qScale < 1 {
			qScale = 1
		}
		if qScale > 31 {
			qScale = 31
		}
		args = append(args, "-q:v", fmt.Sprintf("%d", qScale))

	case FormatWEBP:
		// libwebp uses -quality flag from 0 to 100
		args = append(args, "-c:v", "libwebp", "-quality", fmt.Sprintf("%d", opt.Quality))

	case FormatPNG:
		// PNG is lossless, compression level is 0-9 (default is 5)
		// We map 1-100 quality to 0-9 compression level
		compLevel := int(float64(opt.Quality) * 9.0 / 100.0)
		if compLevel < 0 {
			compLevel = 0
		}
		if compLevel > 9 {
			compLevel = 9
		}
		args = append(args, "-compression_level", fmt.Sprintf("%d", compLevel))
	}

	// Always overwrite output file (-y)
	args = append(args, "-y", outputPath)

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("ffmpeg failed: %s: %w", string(output), err)
	}

	return outputPath, nil
}

// ProcessSticker resizes an image to fit Telegram sticker specification (512x512, WEBP, aspect ratio preserved)
func ProcessSticker(ctx context.Context, inputPath string, outputPath string) error {
	inputPath = filepath.Clean(inputPath)
	outputPath = filepath.Clean(outputPath)

	// scale='if(gt(iw,ih),512,-1)':'if(gt(iw,ih),-1,512)'
	args := []string{
		"-i", inputPath,
		"-vf", "scale='if(gt(iw,ih),512,-1)':'if(gt(iw,ih),-1,512)'",
		"-c:v", "libwebp",
		"-y", outputPath,
	}

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg sticker failed: %s: %w", string(output), err)
	}
	return nil
}

