package ocr

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"

	"github.com/Southclaws/storyden/internal/config"
)

// rasterBatchPages is how many PDF pages are rendered to disk at a time
// before being OCR'd and deleted again.
const rasterBatchPages = 8

// LayeredClient extracts text from images directly via imageOCR, and from
// PDFs by first trying the pure-Go embedded text layer, falling back to
// rasterising pages and running them through imageOCR when the PDF has no
// usable text layer (i.e. it's a scan) and a rasteriser is available.
type LayeredClient struct {
	logger           *slog.Logger
	imageOCR         Client
	raster           Rasterizer
	rasterizeEnabled bool
	maxPages         int
	dpi              int
}

func NewLayeredClient(logger *slog.Logger, imageOCR Client, raster Rasterizer, cfg config.Config) *LayeredClient {
	return &LayeredClient{
		logger:           logger,
		imageOCR:         imageOCR,
		raster:           raster,
		rasterizeEnabled: cfg.OCRPDFRasterizeEnabled,
		maxPages:         cfg.OCRPDFMaxPages,
		dpi:              cfg.OCRPDFDPI,
	}
}

func (c *LayeredClient) ExtractText(ctx context.Context, data []byte, mimeType string) (Result, error) {
	mt := strings.ToLower(mimeType)

	switch {
	case strings.HasPrefix(mt, "image/"):
		if c.imageOCR == nil {
			return Result{}, fault.Wrap(ErrEngineUnavailable, fctx.With(ctx))
		}
		return c.imageOCR.ExtractText(ctx, data, mimeType)

	case strings.Contains(mt, "pdf"):
		return c.extractPDF(ctx, data)

	default:
		return Result{}, fault.Wrap(ErrUnsupportedMIME, fctx.With(ctx))
	}
}

func (c *LayeredClient) extractPDF(ctx context.Context, data []byte) (Result, error) {
	text, pages, err := extractPDFTextLayer(data)
	if err != nil {
		c.logger.Warn("pdf text layer extraction failed, will try rasterisation fallback", slog.String("error", err.Error()))
	}

	if hasUsableText(text) {
		return Result{Text: strings.TrimSpace(text), Pages: pages, Engine: "pdf-textlayer"}, nil
	}

	if !c.rasterizeEnabled || c.raster == nil || !c.raster.Available() || c.imageOCR == nil {
		if strings.TrimSpace(text) != "" {
			// Weak but non-empty text layer and no rasterisation available: better
			// than nothing.
			return Result{Text: strings.TrimSpace(text), Pages: pages, Engine: "pdf-textlayer-weak"}, nil
		}
		return Result{}, fault.Wrap(ErrNoTextLayer, fctx.With(ctx))
	}

	ocrText, ocrPages, err := c.rasterizeAndOCR(ctx, data)
	if err != nil {
		if strings.TrimSpace(text) != "" {
			return Result{Text: strings.TrimSpace(text), Pages: pages, Engine: "pdf-textlayer-weak"}, nil
		}
		return Result{}, err
	}

	return Result{Text: ocrText, Pages: ocrPages, Engine: "pdftoppm+ocr"}, nil
}

func (c *LayeredClient) rasterizeAndOCR(ctx context.Context, data []byte) (string, int, error) {
	tmpDir, err := os.MkdirTemp("", "storyden_ocr_pdf_*")
	if err != nil {
		return "", 0, fault.Wrap(err, fctx.With(ctx))
	}
	defer os.RemoveAll(tmpDir)

	pdfPath := tmpDir + string(os.PathSeparator) + "input.pdf"
	if err := os.WriteFile(pdfPath, data, 0o600); err != nil {
		return "", 0, fault.Wrap(err, fctx.With(ctx))
	}

	var sb strings.Builder
	done := 0

	// Rasterise in small windows rather than the whole document at once. A
	// 200-page scan rendered up front puts gigabytes into the temp directory
	// and is then followed by a long CPU-only OCR stretch with no disk
	// activity at all — which on at least one Windows setup was long enough
	// for the drives to hit their idle power-down timeout, and the next
	// access bugchecked the machine. Batching bounds temp usage and keeps
	// I/O trickling throughout.
	for first := 1; first <= c.maxPages; first += rasterBatchPages {
		last := min(first+rasterBatchPages-1, c.maxPages)

		batchDir := filepath.Join(tmpDir, "batch"+strconv.Itoa(first))
		if err := os.MkdirAll(batchDir, 0o700); err != nil {
			return "", 0, fault.Wrap(err, fctx.With(ctx))
		}

		pages, err := c.raster.Rasterize(ctx, pdfPath, batchDir, first, last, c.dpi)
		if err != nil {
			os.RemoveAll(batchDir)

			// Asking for a range past the last page is how the loop discovers
			// the document has ended, so it is only a real failure if nothing
			// has been rendered yet.
			if done > 0 {
				break
			}
			return "", 0, fault.Wrap(err, fctx.With(ctx))
		}

		if len(pages) == 0 {
			os.RemoveAll(batchDir)
			break
		}

		for _, pagePath := range pages {
			img, err := os.ReadFile(pagePath)
			if err != nil {
				continue
			}

			res, err := c.imageOCR.ExtractText(ctx, img, "image/png")
			if err != nil {
				if errors.Is(err, ErrEngineUnavailable) {
					os.RemoveAll(batchDir)
					return "", 0, err
				}
				continue
			}

			if sb.Len() > 0 {
				sb.WriteString("\n\n")
			}
			sb.WriteString(res.Text)
		}

		done += len(pages)
		os.RemoveAll(batchDir)

		if len(pages) < last-first+1 {
			break
		}
	}

	if done == 0 {
		return "", 0, fault.Wrap(ErrNoTextLayer, fctx.With(ctx))
	}

	return strings.TrimSpace(sb.String()), done, nil
}
