package ocr

import "github.com/Southclaws/fault"

var (
	// ErrUnsupportedMIME is returned when the given MIME type cannot be
	// processed by any extraction stage.
	ErrUnsupportedMIME = fault.New("ocr: unsupported mime type")
	// ErrEngineUnavailable is returned when a required external binary
	// (tesseract, pdftoppm) is not installed on the host. This is distinct
	// from a successful extraction that happens to find no text.
	ErrEngineUnavailable = fault.New("ocr: engine unavailable")
	// ErrNoTextLayer is returned when a PDF has no usable embedded text
	// layer and no rasterisation fallback is available to OCR it.
	ErrNoTextLayer = fault.New("ocr: pdf has no usable text layer and no rasteriser available")
)
