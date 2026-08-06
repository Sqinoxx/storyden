package ocr

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/Southclaws/fault"
)

type TesseractClient struct {
	binaryPath string
	languages  []string
	logger     *slog.Logger
}

func NewTesseractClient(binaryPath string, languages []string, logger *slog.Logger) *TesseractClient {
	if binaryPath == "" {
		binaryPath = "tesseract"
	}
	return &TesseractClient{
		binaryPath: binaryPath,
		languages:  languages,
		logger:     logger,
	}
}

func (t *TesseractClient) ExtractText(ctx context.Context, r io.Reader, mimeType string) (string, error) {
	// Create temporary input file
	ext := ".img"
	if strings.Contains(mimeType, "png") {
		ext = ".png"
	} else if strings.Contains(mimeType, "jpeg") || strings.Contains(mimeType, "jpg") {
		ext = ".jpg"
	} else if strings.Contains(mimeType, "webp") {
		ext = ".webp"
	} else if strings.Contains(mimeType, "pdf") {
		ext = ".pdf"
	}

	tmpFile, err := os.CreateTemp("", "storyden_ocr_*"+ext)
	if err != nil {
		return "", fault.Wrap(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := io.Copy(tmpFile, r); err != nil {
		tmpFile.Close()
		return "", fault.Wrap(err)
	}
	tmpFile.Close()

	langFlag := strings.Join(t.languages, "+")
	if langFlag == "" {
		langFlag = "eng"
	}

	// Tesseract CLI syntax: tesseract <input> stdout -l <lang>
	cmd := exec.CommandContext(ctx, t.binaryPath, tmpFile.Name(), "stdout", "-l", langFlag)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		// Check if tesseract binary is not installed
		if execErr, ok := err.(*exec.Error); ok && execErr.Err == exec.ErrNotFound {
			t.logger.Warn("Tesseract binary not found on system path, OCR extraction skipped", slog.String("path", t.binaryPath))
			return "", nil
		}
		t.logger.Error("Tesseract OCR execution failed", slog.String("stderr", stderr.String()), slog.String("error", err.Error()))
		return "", fault.Wrap(fmt.Errorf("tesseract execution failed: %w: %s", err, stderr.String()))
	}

	extractedText := strings.TrimSpace(stdout.String())
	return extractedText, nil
}
