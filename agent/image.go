package agent

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
)

func Base64EncodeImage(filename string) (string, error) {
	ext := filepath.Ext(filename)

	mimeType := ""
	// Convert extension to proper MIME type
	switch ext {
	case ".jpg", ".jpeg":
		mimeType = "image/jpeg"
	case ".png":
		mimeType = "image/png"
	case ".gif":
		mimeType = "image/gif"
	case ".bmp":
		mimeType = "image/bmp"
	case ".webp":
		mimeType = "image/webp"
	default:
		return "", fmt.Errorf("unsupported image format: %s", ext)
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to open image file: %w", err)
	}

	encodedImage := base64.StdEncoding.EncodeToString(data)
	return "data:" + mimeType + ";base64," + encodedImage, nil
}
