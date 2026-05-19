package imgproc

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CreateZip packages multiple files into a single zip file at outputPath
func CreateZip(outputPath string, files []string) error {
	// Create directories if they don't exist
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create zip target directory: %w", err)
	}

	zipFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipFile.Close()

	archive := zip.NewWriter(zipFile)
	defer archive.Close()

	for i, file := range files {
		err := func() error {
			f, err := os.Open(file)
			if err != nil {
				return fmt.Errorf("failed to open file %s: %w", file, err)
			}
			defer f.Close()

			// Get clean name for the file inside the zip (e.g. image_001.jpg)
			ext := filepath.Ext(file)
			zipFileName := fmt.Sprintf("image_%03d%s", i+1, ext)

			writer, err := archive.Create(zipFileName)
			if err != nil {
				return fmt.Errorf("failed to create zip entry for %s: %w", file, err)
			}

			if _, err := io.Copy(writer, f); err != nil {
				return fmt.Errorf("failed to write file %s to zip: %w", file, err)
			}
			return nil
		}()
		if err != nil {
			return err
		}
	}

	return nil
}
