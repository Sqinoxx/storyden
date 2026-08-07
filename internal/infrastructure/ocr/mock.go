package ocr

import "context"

type MockClient struct{}

func NewMockClient() *MockClient {
	return &MockClient{}
}

func (m *MockClient) ExtractText(ctx context.Context, data []byte, mimeType string) (Result, error) {
	return Result{Text: "Mock OCR text extracted from image", Engine: "mock"}, nil
}
