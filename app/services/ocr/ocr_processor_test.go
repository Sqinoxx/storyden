package ocr

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	infra_ocr "github.com/Southclaws/storyden/internal/infrastructure/ocr"
)

func TestAwaitExtraction_ReturnsResultWhenFast(t *testing.T) {
	res, err := awaitExtraction(time.Second, func() (infra_ocr.Result, error) {
		return infra_ocr.Result{Text: "hello", Engine: "test"}, nil
	})

	require.NoError(t, err)
	assert.Equal(t, "hello", res.Text)
}

func TestAwaitExtraction_PropagatesFastError(t *testing.T) {
	wantErr := errors.New("boom")

	_, err := awaitExtraction(time.Second, func() (infra_ocr.Result, error) {
		return infra_ocr.Result{}, wantErr
	})

	assert.ErrorIs(t, err, wantErr)
}

func TestAwaitExtraction_AbandonsCallThatOutlivesTimeout(t *testing.T) {
	release := make(chan struct{})
	defer close(release) // let the goroutine finish so it doesn't outlive the test binary

	started := time.Now()
	_, err := awaitExtraction(20*time.Millisecond, func() (infra_ocr.Result, error) {
		<-release
		return infra_ocr.Result{Text: "too late"}, nil
	})
	elapsed := time.Since(started)

	assert.ErrorIs(t, err, ErrExtractionAbandoned)
	assert.Less(t, elapsed, 500*time.Millisecond, "should return promptly on timeout rather than waiting for the slow call")
}
