package ocr_test

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Southclaws/storyden/internal/config"
	"github.com/Southclaws/storyden/internal/infrastructure/ocr"
)

func TestMockClient(t *testing.T) {
	client := ocr.NewMockClient()
	text, err := client.ExtractText(context.Background(), strings.NewReader("dummy image data"), "image/png")
	require.NoError(t, err)
	assert.Contains(t, text, "Mock OCR text")
}

func TestOCRProviderFactory(t *testing.T) {
	logger := slog.Default()

	// Disabled OCR -> Mock client
	cfg := config.Config{OCREnabled: false}
	client := ocr.New(cfg, logger)
	assert.NotNil(t, client)

	// Mock provider
	cfg = config.Config{OCREnabled: true, OCRProvider: "mock"}
	client = ocr.New(cfg, logger)
	assert.NotNil(t, client)

	// Tesseract provider
	cfg = config.Config{OCREnabled: true, OCRProvider: "tesseract"}
	client = ocr.New(cfg, logger)
	assert.NotNil(t, client)
}
