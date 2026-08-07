package ocr

import "context"

// Result is the outcome of a successful text extraction.
type Result struct {
	// Text is the extracted plain text content.
	Text string
	// Pages is the number of pages processed, where applicable (PDFs). Zero
	// for single-image extractions.
	Pages int
	// Engine identifies which extraction path produced the result, e.g.
	// "pdf-textlayer", "tesseract", "pdftoppm+tesseract", "openai-vision".
	Engine string
}

// Client defines the interface for text extraction from images or documents.
type Client interface {
	ExtractText(ctx context.Context, data []byte, mimeType string) (Result, error)
}
