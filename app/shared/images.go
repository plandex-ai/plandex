package shared

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"io"
	"log"
	"math"
	"path/filepath"
	"strings"

	"github.com/sashabaranov/go-openai"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/webp"
)

func GetImageTokens(base64Image string, detail openai.ImageURLDetail) (int, error) {
	imageData, err := base64.StdEncoding.DecodeString(base64Image)
	if err != nil {
		log.Println("failed to decode base64 image data:", err)
		return 0, fmt.Errorf("failed to decode base64 image data: %w", err)
	}

	return GetImageTokensFromHeader(bytes.NewReader(imageData), detail, int64(len(imageData)))
}

func GetImageTokensFromHeader(reader io.Reader, detail openai.ImageURLDetail, maxBytes int64) (int, error) {
	reader = io.LimitReader(reader, maxBytes)
	img, _, err := image.DecodeConfig(reader)
	if err != nil {
		log.Println("failed to decode image config:", err)
		return 0, fmt.Errorf("failed to decode image config: %w", err)
	}

	width, height := img.Width, img.Height

	anthropicTokens := getAnthropicImageTokens(width, height)
	googleTokens := getGoogleImageTokens(width, height)
	openaiTokens := getOpenAIImageTokens(width, height, detail)

	// log.Printf("GetImageTokens - width: %d, height: %d\n", width, height)
	// log.Printf("GetImageTokens - anthropicTokens: %d\n", anthropicTokens)
	// log.Printf("GetImageTokens - googleTokens: %d\n", googleTokens)
	// log.Printf("GetImageTokens - openaiTokens: %d\n", openaiTokens)

	// get max of the three
	return int(math.Max(
		float64(anthropicTokens),
		math.Max(
			float64(googleTokens),
			float64(openaiTokens),
		),
	)), nil
}

func GetImageTokensEstimateFromBytes(l int64) int {
	return int(l) / 750
}

func getAnthropicImageTokens(width, height int) int {
	// Anthropic uses a simple area-based calculation (1 token per ~750 px²)
	area := width * height
	return int(math.Ceil(float64(area) / 750.0))
}

func getGoogleImageTokens(width, height int) int {
	// Google Gemini uses 768px tiles at 258 tokens per tile
	const tileSize = 768
	const tokensPerTile = 258

	horizontalTiles := int(math.Ceil(float64(width) / float64(tileSize)))
	verticalTiles := int(math.Ceil(float64(height) / float64(tileSize)))

	numTiles := horizontalTiles * verticalTiles
	return numTiles * tokensPerTile
}

func getOpenAIImageTokens(width, height int, detail openai.ImageURLDetail) int {
	const (
		lowDetailTokens  = 85
		highDetailBase   = 85
		highDetailFactor = 170
	)

	if detail == openai.ImageURLDetailLow {
		return lowDetailTokens
	}

	// Scale to fit within 2048px square
	if width > 2048 || height > 2048 {
		scaleFactor := math.Min(2048.0/float64(width), 2048.0/float64(height))
		width = int(float64(width) * scaleFactor)
		height = int(float64(height) * scaleFactor)
	}

	// Scale shortest side to 768px
	if width < height {
		scaleFactor := 768.0 / float64(width)
		width = 768
		height = int(float64(height) * scaleFactor)
	} else {
		scaleFactor := 768.0 / float64(height)
		height = 768
		width = int(float64(width) * scaleFactor)
	}

	// Calculate 512px tiles
	horizontalTiles := int(math.Ceil(float64(width) / 512.0))
	verticalTiles := int(math.Ceil(float64(height) / 512.0))

	numTiles := horizontalTiles * verticalTiles
	return highDetailBase + numTiles*highDetailFactor
}

func GetImageDataURI(base64Image, path string) string {
	mimeType := ImageMimeType(path)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, base64Image)
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
	return "application/octet-stream"
}
