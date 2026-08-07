package ocr

import (
	"context"
	"log/slog"

	"github.com/Southclaws/storyden/internal/config"
)

// disabledClient always reports the extraction backend as unavailable. It is
// used when OCR is disabled entirely, so that an asset ends up `skipped`
// with a clear reason rather than `completed` with fabricated text.
type disabledClient struct{}

func (disabledClient) ExtractText(ctx context.Context, data []byte, mimeType string) (Result, error) {
	return Result{}, ErrEngineUnavailable
}

func New(cfg config.Config, logger *slog.Logger) Client {
	if !cfg.OCREnabled {
		return disabledClient{}
	}

	switch cfg.OCRProvider {
	case "mock":
		return NewMockClient()

	case "textlayer":
		// Pure-Go PDF text layer extraction only, no OCR binaries involved.
		return NewLayeredClient(logger, nil, nil, cfg)

	case "openai_vision":
		var imageOCR Client
		if cfg.OpenAIKey != "" {
			imageOCR = NewOpenAIVisionClient(cfg.OpenAIKey)
		} else {
			logger.Warn("OCRProvider set to openai_vision but OPENAI_API_KEY is empty, falling back to Tesseract OCR")
			imageOCR = NewTesseractClient(cfg.TesseractBinaryPath, cfg.OCRLanguages, logger)
		}
		raster := NewPopplerRasterizer(cfg.PDFToPPMBinaryPath)
		return NewLayeredClient(logger, imageOCR, raster, cfg)

	case "tesseract":
		fallthrough
	default:
		imageOCR := NewTesseractClient(cfg.TesseractBinaryPath, cfg.OCRLanguages, logger)
		raster := NewPopplerRasterizer(cfg.PDFToPPMBinaryPath)
		return NewLayeredClient(logger, imageOCR, raster, cfg)
	}
}
