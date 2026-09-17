package ocr

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Southclaws/storyden/internal/config"
	infra_ocr "github.com/Southclaws/storyden/internal/infrastructure/ocr"
)

// blockingOCRClient never returns on its own, simulating a stuck engine
// invocation (e.g. a pathological file that makes Tesseract/rasterisation
// hang). It only unblocks when the context passed to it is cancelled.
type blockingOCRClient struct{}

func (blockingOCRClient) ExtractText(ctx context.Context, data []byte, mimeType string) (infra_ocr.Result, error) {
	<-ctx.Done()
	return infra_ocr.Result{}, ctx.Err()
}

// TestExtractTextTimesOutOnStuckEngine covers B16: a single slow extraction
// used to be able to hold the processor's one concurrency slot forever,
// starving every other pending asset. extractText must bound the call and
// report a distinct, non-retryable timeout error.
func TestExtractTextTimesOutOnStuckEngine(t *testing.T) {
	t.Parallel()

	p := &Processor{
		cfg:       config.Config{OCRTimeout: 20 * time.Millisecond},
		ocrClient: blockingOCRClient{},
	}

	start := time.Now()
	_, err := p.extractText(context.Background(), nil, "image/png")
	elapsed := time.Since(start)

	require.ErrorIs(t, err, errOCRTimeout)
	assert.Less(t, elapsed, time.Second, "extractText must not block past the configured timeout")
}

// TestExtractTextPropagatesParentCancellation ensures a caller-cancelled
// context (e.g. shutdown) is still reported as a timeout-shaped bound, not
// left hanging - the outer context expiring is functionally the same
// situation as the per-call deadline expiring.
func TestExtractTextPropagatesParentCancellation(t *testing.T) {
	t.Parallel()

	p := &Processor{
		cfg:       config.Config{OCRTimeout: time.Minute},
		ocrClient: blockingOCRClient{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := p.extractText(ctx, nil, "image/png")
	require.Error(t, err)
}
