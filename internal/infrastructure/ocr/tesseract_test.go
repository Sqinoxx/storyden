package ocr

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTesseractClient_MissingBinaryReturnsTypedError(t *testing.T) {
	client := NewTesseractClient("storyden-definitely-not-a-real-binary", []string{"eng"}, slog.Default())

	require.False(t, client.Available())

	_, err := client.ExtractText(context.Background(), []byte("fake image bytes"), "image/png")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEngineUnavailable), "missing binary must be a typed error, not a silent empty success")
}

func TestTesseractClient_RejectsNonImageMIME(t *testing.T) {
	client := NewTesseractClient("storyden-definitely-not-a-real-binary", []string{"eng"}, slog.Default())

	_, err := client.ExtractText(context.Background(), []byte("%PDF-1.4"), "application/pdf")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnsupportedMIME))
}
