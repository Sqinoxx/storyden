package ocr

import (
	"context"
	"io"
)

// Client defines the interface for OCR text extraction from images or documents.
type Client interface {
	ExtractText(ctx context.Context, r io.Reader, mimeType string) (string, error)
}
