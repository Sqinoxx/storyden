package ocr

import (
	"context"
	"io"
)

type MockClient struct{}

func NewMockClient() *MockClient {
	return &MockClient{}
}

func (m *MockClient) ExtractText(ctx context.Context, r io.Reader, mimeType string) (string, error) {
	// Return a dummy mock OCR result for testing or development
	return "Mock OCR text extracted from image", nil
}
