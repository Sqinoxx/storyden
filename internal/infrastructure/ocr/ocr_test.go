package ocr_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Southclaws/storyden/internal/config"
	"github.com/Southclaws/storyden/internal/infrastructure/ocr"
)

func TestMockClient(t *testing.T) {
	client := ocr.NewMockClient()
	result, err := client.ExtractText(context.Background(), []byte("dummy image data"), "image/png")
	require.NoError(t, err)
	assert.Contains(t, result.Text, "Mock OCR text")
}

func TestOCRProviderFactory(t *testing.T) {
	logger := slog.Default()

	// Disabled OCR -> a client that always reports unavailable, never fabricated text.
	cfg := config.Config{OCREnabled: false}
	client := ocr.New(cfg, logger)
	require.NotNil(t, client)
	_, err := client.ExtractText(context.Background(), []byte("data"), "image/png")
	assert.ErrorIs(t, err, ocr.ErrEngineUnavailable)

	// Mock provider
	cfg = config.Config{OCREnabled: true, OCRProvider: "mock"}
	client = ocr.New(cfg, logger)
	assert.NotNil(t, client)

	// Pure-Go text-layer provider, no external binaries required.
	cfg = config.Config{OCREnabled: true, OCRProvider: "textlayer"}
	client = ocr.New(cfg, logger)
	assert.NotNil(t, client)

	// Tesseract provider (layered with PDF text-layer extraction)
	cfg = config.Config{OCREnabled: true, OCRProvider: "tesseract", OCRPDFRasterizeEnabled: true}
	client = ocr.New(cfg, logger)
	assert.NotNil(t, client)
}
