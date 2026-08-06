package ocr

import (
	"log/slog"

	"github.com/Southclaws/storyden/internal/config"
)

func New(cfg config.Config, logger *slog.Logger) Client {
	if !cfg.OCREnabled {
		return NewMockClient()
	}

	switch cfg.OCRProvider {
	case "openai_vision":
		if cfg.OpenAIKey != "" {
			return NewOpenAIVisionClient(cfg.OpenAIKey)
		}
		logger.Warn("OCRProvider set to openai_vision but OPENAI_API_KEY is empty, falling back to Tesseract OCR")
		return NewTesseractClient(cfg.TesseractBinaryPath, cfg.OCRLanguages, logger)
	case "mock":
		return NewMockClient()
	case "tesseract":
		fallthrough
	default:
		return NewTesseractClient(cfg.TesseractBinaryPath, cfg.OCRLanguages, logger)
	}
}
