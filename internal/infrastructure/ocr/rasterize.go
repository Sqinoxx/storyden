package ocr

import (
	"context"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/fmsg"
)

// Rasterizer renders PDF pages to image files so they can be passed through
// an image OCR engine, for PDFs that have no usable embedded text layer
// (i.e. scanned documents).
type Rasterizer interface {
	Available() bool
	// Rasterize renders pages firstPage..lastPage (1-based, inclusive) of the
	// PDF at pdfPath into PNG files inside outDir, and returns their paths in
	// page order. Rendering a range rather than the whole document lets the
	// caller bound how much temporary disk one PDF occupies at a time.
	Rasterize(ctx context.Context, pdfPath, outDir string, firstPage, lastPage, dpi int) ([]string, error)
}

// PopplerRasterizer shells out to `pdftoppm` (part of poppler-utils).
type PopplerRasterizer struct {
	binaryPath string
	available  bool
}

func NewPopplerRasterizer(binaryPath string) *PopplerRasterizer {
	if binaryPath == "" {
		binaryPath = "pdftoppm"
	}

	_, err := exec.LookPath(binaryPath)

	return &PopplerRasterizer{
		binaryPath: binaryPath,
		available:  err == nil,
	}
}

func (p *PopplerRasterizer) Available() bool { return p.available }

func (p *PopplerRasterizer) Rasterize(ctx context.Context, pdfPath, outDir string, firstPage, lastPage, dpi int) ([]string, error) {
	if !p.available {
		return nil, fault.Wrap(ErrEngineUnavailable, fctx.With(ctx), fmsg.With("pdftoppm binary not found"))
	}

	outputPrefix := filepath.Join(outDir, "page")

	args := []string{
		"-png",
		"-r", strconv.Itoa(dpi),
		"-f", strconv.Itoa(firstPage),
		"-l", strconv.Itoa(lastPage),
		pdfPath,
		outputPrefix,
	}

	cmd := exec.CommandContext(ctx, p.binaryPath, args...)
	if err := cmd.Run(); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx), fmsg.With("pdftoppm execution failed"))
	}

	matches, err := filepath.Glob(outputPrefix + "*.png")
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	sort.Strings(matches)

	return matches, nil
}
