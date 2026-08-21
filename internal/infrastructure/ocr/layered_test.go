package ocr

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
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
func (unavailableRasterizer) Rasterize(ctx context.Context, pdfPath, outDir string, firstPage, lastPage, dpi int) ([]string, error) {
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

// fakeRasterizer renders a document of a fixed page count, recording which
// ranges it was asked for and where it wrote, so batching can be asserted.
type fakeRasterizer struct {
	totalPages int
	ranges     [][2]int
	batchDirs  []string
}

func (f *fakeRasterizer) Available() bool { return true }

func (f *fakeRasterizer) Rasterize(ctx context.Context, pdfPath, outDir string, firstPage, lastPage, dpi int) ([]string, error) {
	f.ranges = append(f.ranges, [2]int{firstPage, lastPage})
	f.batchDirs = append(f.batchDirs, outDir)

	if firstPage > f.totalPages {
		return nil, errors.New("wrong page range given: the first page is greater than the last page")
	}

	paths := []string{}
	for page := firstPage; page <= lastPage && page <= f.totalPages; page++ {
		path := filepath.Join(outDir, fmt.Sprintf("page-%03d.png", page))
		if err := os.WriteFile(path, []byte("png"), 0o600); err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}

	return paths, nil
}

// A long scan must be rendered in bounded windows rather than all at once:
// rendering every page up front fills the temp directory and then leaves the
// disks idle for the whole OCR stretch.
func TestLayeredClient_RasterisesInBatches(t *testing.T) {
	img := &recordingImageClient{}
	raster := &fakeRasterizer{totalPages: 20}

	c := NewLayeredClient(slog.Default(), img, raster, config.Config{
		OCRPDFRasterizeEnabled: true,
		OCRPDFMaxPages:         200,
		OCRPDFDPI:              200,
	})

	result, err := c.ExtractText(context.Background(), makeEmptyPDF(t), "application/pdf")
	require.NoError(t, err)

	assert.Equal(t, "pdftoppm+ocr", result.Engine)
	assert.Equal(t, 20, result.Pages)
	assert.Equal(t, 20, img.calls, "every page of the document should reach the OCR engine")

	assert.Equal(t, [][2]int{{1, 8}, {9, 16}, {17, 24}}, raster.ranges,
		"pages must be requested in windows, stopping once a short batch shows the document ended")

	for _, dir := range raster.batchDirs {
		_, err := os.Stat(dir)
		assert.True(t, os.IsNotExist(err), "batch directory %s must be cleaned up, not left to accumulate", dir)
	}
}

// The page cap has to bound the work even when the document is longer.
func TestLayeredClient_RasterisationStopsAtMaxPages(t *testing.T) {
	img := &recordingImageClient{}
	raster := &fakeRasterizer{totalPages: 500}

	c := NewLayeredClient(slog.Default(), img, raster, config.Config{
		OCRPDFRasterizeEnabled: true,
		OCRPDFMaxPages:         10,
		OCRPDFDPI:              200,
	})

	result, err := c.ExtractText(context.Background(), makeEmptyPDF(t), "application/pdf")
	require.NoError(t, err)

	assert.Equal(t, 10, result.Pages)
	assert.Equal(t, 10, img.calls)
	assert.Equal(t, [][2]int{{1, 8}, {9, 10}}, raster.ranges)
}

// Running past the end of a document is how the loop detects the end; it must
// not surface as a failure once pages have already been extracted.
func TestLayeredClient_RangePastEndOfDocumentEndsCleanly(t *testing.T) {
	img := &recordingImageClient{}
	raster := &fakeRasterizer{totalPages: 8}

	c := NewLayeredClient(slog.Default(), img, raster, config.Config{
		OCRPDFRasterizeEnabled: true,
		OCRPDFMaxPages:         200,
		OCRPDFDPI:              200,
	})

	result, err := c.ExtractText(context.Background(), makeEmptyPDF(t), "application/pdf")
	require.NoError(t, err)

	assert.Equal(t, 8, result.Pages)
	assert.Equal(t, [][2]int{{1, 8}, {9, 16}}, raster.ranges)
}
