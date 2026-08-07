package ocr

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Southclaws/storyden/internal/config"
)

type recordingImageClient struct {
	calls int
}

func (c *recordingImageClient) ExtractText(ctx context.Context, data []byte, mimeType string) (Result, error) {
	c.calls++
	return Result{Text: "image ocr text", Engine: "test-image"}, nil
}

type unavailableRasterizer struct{}

func (unavailableRasterizer) Available() bool { return false }
func (unavailableRasterizer) Rasterize(ctx context.Context, pdfPath, outDir string, maxPages, dpi int) ([]string, error) {
	return nil, errors.New("not available")
}

func TestLayeredClient_PDFWithTextLayerSkipsImageOCR(t *testing.T) {
	img := &recordingImageClient{}
	c := NewLayeredClient(slog.Default(), img, unavailableRasterizer{}, config.Config{OCRPDFRasterizeEnabled: true})

	data := makeTextPDF(t, "Spanisch fuer Mediziner im vierten Semester ist Pflicht")

	result, err := c.ExtractText(context.Background(), data, "application/pdf")
	require.NoError(t, err)
	assert.Contains(t, result.Text, "Spanisch")
	assert.Equal(t, "pdf-textlayer", result.Engine)
	assert.Equal(t, 0, img.calls, "image OCR must not run when the text layer is already usable")
}

func TestLayeredClient_PDFWithoutTextLayerAndNoRasterizerFails(t *testing.T) {
	c := NewLayeredClient(slog.Default(), nil, unavailableRasterizer{}, config.Config{OCRPDFRasterizeEnabled: true})

	data := makeEmptyPDF(t)

	_, err := c.ExtractText(context.Background(), data, "application/pdf")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoTextLayer)
}

func TestLayeredClient_UnsupportedMIME(t *testing.T) {
	c := NewLayeredClient(slog.Default(), nil, nil, config.Config{})

	_, err := c.ExtractText(context.Background(), []byte("data"), "text/plain")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnsupportedMIME)
}

func TestLayeredClient_ImageRoutesDirectlyToImageOCR(t *testing.T) {
	img := &recordingImageClient{}
	c := NewLayeredClient(slog.Default(), img, nil, config.Config{})

	result, err := c.ExtractText(context.Background(), []byte("fakeimage"), "image/png")
	require.NoError(t, err)
	assert.Equal(t, "image ocr text", result.Text)
	assert.Equal(t, 1, img.calls)
}
