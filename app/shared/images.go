package shared

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"math"
	"path/filepath"
	"strings"

	"github.com/sashabaranov/go-openai"

	"image"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/webp"
)

func GetImageDataURI(base64Image, path string) string {
	mimeType := ImageMimeType(path)

	dataURI := fmt.Sprintf("data:%s;base64,%s", mimeType, base64Image)

	return dataURI
}

func GetImageTokens(base64Image string, detail openai.ImageURLDetail) (int, error) {
	imageData, err := base64.StdEncoding.DecodeString(base64Image)
	if err != nil {
		return 0, fmt.Errorf("failed to decode base64 image data: %w", err)
	}

	img, _, err := image.DecodeConfig(bytes.NewReader(imageData))
	if err != nil {
		return 0, fmt.Errorf("failed to decode image: %s", err)
	}

	return GetImageTokensForDims(img.Width, img.Height, detail), nil
}

func GetImageTokensForDims(width, height int, detail openai.ImageURLDetail) int {
	const (
		lowDetailTokens  = 85
		highDetailBase   = 85
		highDetailFactor = 170
	)

	if detail == "low" {
		return lowDetailTokens
	}

	// Scale the image to fit within a 2048 x 2048 square
	if width > 2048 || height > 2048 {
		scaleFactor := math.Min(2048.0/float64(width), 2048.0/float64(height))
		width = int(float64(width) * scaleFactor)
		height = int(float64(height) * scaleFactor)
	}

	// Scale the shortest side to 768px
	if width < height {
		scaleFactor := 768.0 / float64(width)
		width = 768
		height = int(float64(height) * scaleFactor)
	} else {
		scaleFactor := 768.0 / float64(height)
		height = 768
		width = int(float64(width) * scaleFactor)
	}

	// Calculate the number of 512px tiles
	numTiles := int(math.Ceil(float64(width)/512.0) * math.Ceil(float64(height)/512.0))

	return highDetailBase + numTiles*highDetailFactor
}

func IsImageFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".webp" || ext == ".gif"
}

func ImageMimeType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	}
	return ""
}
