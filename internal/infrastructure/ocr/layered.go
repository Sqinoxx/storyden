package ocr

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"

	"github.com/Southclaws/storyden/internal/config"
)

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

	pages, err := c.raster.Rasterize(ctx, pdfPath, tmpDir, c.maxPages, c.dpi)
	if err != nil {
		return "", 0, fault.Wrap(err, fctx.With(ctx))
	}

	if len(pages) == 0 {
		return "", 0, fault.Wrap(ErrNoTextLayer, fctx.With(ctx))
	}

	var sb strings.Builder
	for _, pagePath := range pages {
		img, err := os.ReadFile(pagePath)
		if err != nil {
			continue
		}

		res, err := c.imageOCR.ExtractText(ctx, img, "image/png")
		if err != nil {
			if errors.Is(err, ErrEngineUnavailable) {
				return "", 0, err
			}
			continue
		}

		if sb.Len() > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(res.Text)
	}

	return strings.TrimSpace(sb.String()), len(pages), nil
}
