package ocr

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/fmsg"
)

// TesseractClient extracts text from images using the local Tesseract OCR
// CLI binary. It only ever handles image MIME types — PDF handling belongs
// to the layered client, which rasterises pages before handing them here.
type TesseractClient struct {
	binaryPath string
	languages  []string
	logger     *slog.Logger
	available  bool
}

func NewTesseractClient(binaryPath string, languages []string, logger *slog.Logger) *TesseractClient {
	if binaryPath == "" {
		binaryPath = "tesseract"
	}

	_, err := exec.LookPath(binaryPath)

	return &TesseractClient{
		binaryPath: binaryPath,
		languages:  languages,
		logger:     logger,
		available:  err == nil,
	}
}

func (t *TesseractClient) Available() bool { return t.available }

func (t *TesseractClient) ExtractText(ctx context.Context, data []byte, mimeType string) (Result, error) {
	if !strings.HasPrefix(strings.ToLower(mimeType), "image/") {
		return Result{}, fault.Wrap(ErrUnsupportedMIME, fctx.With(ctx), fmsg.WithDesc("unsupported mime type", mimeType))
	}

	if !t.available {
		return Result{}, fault.Wrap(ErrEngineUnavailable, fctx.With(ctx), fmsg.With("tesseract binary not found on system path"))
	}

	text, err := t.extractImage(ctx, data)
	if err != nil {
		return Result{}, err
	}

	return Result{Text: text, Engine: "tesseract"}, nil
}

func (t *TesseractClient) extractImage(ctx context.Context, data []byte) (string, error) {
	ext := ".img"
	switch {
	case bytes.HasPrefix(data, []byte("\x89PNG")):
		ext = ".png"
	case bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}):
		ext = ".jpg"
	case bytes.HasPrefix(data, []byte("RIFF")):
		ext = ".webp"
	}

	tmpFile, err := os.CreateTemp("", "storyden_ocr_*"+ext)
	if err != nil {
		return "", fault.Wrap(err, fctx.With(ctx))
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return "", fault.Wrap(err, fctx.With(ctx))
	}
	tmpFile.Close()

	return t.runOnFile(ctx, tmpFile.Name())
}

func (t *TesseractClient) runOnFile(ctx context.Context, path string) (string, error) {
	langFlag := strings.Join(t.languages, "+")
	if langFlag == "" {
		langFlag = "eng"
	}

	// Tesseract CLI syntax: tesseract <input> stdout -l <lang>
	cmd := exec.CommandContext(ctx, t.binaryPath, path, "stdout", "-l", langFlag)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if execErr, ok := err.(*exec.Error); ok && execErr.Err == exec.ErrNotFound {
			return "", fault.Wrap(ErrEngineUnavailable, fctx.With(ctx))
		}
		t.logger.Error("tesseract OCR execution failed", slog.String("stderr", stderr.String()), slog.String("error", err.Error()))
		return "", fault.Wrap(fault.Newf("tesseract execution failed: %s: %s", err.Error(), stderr.String()), fctx.With(ctx))
	}

	return strings.TrimSpace(stdout.String()), nil
}
