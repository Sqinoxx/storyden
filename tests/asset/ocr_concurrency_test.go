package asset_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/asset/asset_querier"
	"github.com/Southclaws/storyden/app/resources/asset/asset_writer"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/services/ocr"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/config"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
)

// The bulk import and the admin reindex both drive extraction through
// ProcessAllPending with OCR_CONCURRENCY above one. Uploads are already
// processed by the live bus handler, so the queue is refilled through the same
// reset the reindex endpoint uses and then drained in a single call.
func TestOCR_ConcurrentBatchDrainsEverything(t *testing.T) {
	const assetCount = 12

	cfg := &config.Config{
		OCREnabled:         true,
		OCRProvider:        "mock",
		OCRBackfillEnabled: false,
		OCRConcurrency:     4,
		OCRMaxFileSizeMB:   10,
		OCRMaxTextLength:   200000,
	}

	integration.Test(t, cfg, e2e.Setup(), fx.Invoke(func(
		root context.Context,
		lc fx.Lifecycle,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
		aq *asset_querier.Querier,
		writer *asset_writer.Writer,
		proc *ocr.Processor,
	) {
		lc.Append(fx.StartHook(func() {
			a := assert.New(t)
			r := require.New(t)

			adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			adminSession := sh.WithSession(adminCtx)

			for n := range assetCount {
				uploaded := uploadTestAsset(t, root, cl, adminSession, "image/png", onePixelPNG)
				_, err := xid.FromString(uploaded.Id)
				r.NoError(err, "asset %d", n)
			}

			reset, err := writer.ResetOCRForReindex(root, assetCount*2)
			r.NoError(err)
			r.NotEmpty(reset, "reindex should have queued the uploaded assets")

			// The bus handler may still be finishing an upload when the reset
			// runs, so the queue can be a little larger than the reset returned.
			// What must hold is that one call drains all of it.
			processed, err := proc.ProcessAllPending(root, assetCount*2)
			r.NoError(err)
			a.GreaterOrEqual(processed, len(reset), "the pool must drain every queued asset in one call")

			for n, id := range reset {
				stored, err := aq.GetByID(root, xid.ID(id))
				r.NoError(err, "asset %d", n)
				a.Equal("completed", stored.OCRStatus, fmt.Sprintf("asset %d left in %q", n, stored.OCRStatus))
				a.NotEmpty(stored.OCRText.OrZero())
			}

			remaining, err := proc.ProcessAllPending(root, assetCount*2)
			r.NoError(err)
			a.Zero(remaining, "nothing should be left pending")
		}))
	}))
}

// A default of one keeps small always-on deployments responsive, and a
// misconfigured zero or negative value must clamp rather than deadlock.
func TestOCR_ConcurrencyDefaultsAndClamps(t *testing.T) {
	for name, concurrency := range map[string]int{"unset": 0, "negative": -3, "single": 1} {
		t.Run(name, func(t *testing.T) {
			cfg := &config.Config{
				OCREnabled:         true,
				OCRProvider:        "mock",
				OCRBackfillEnabled: false,
				OCRConcurrency:     concurrency,
				OCRMaxFileSizeMB:   10,
				OCRMaxTextLength:   200000,
			}

			integration.Test(t, cfg, e2e.Setup(), fx.Invoke(func(
				root context.Context,
				lc fx.Lifecycle,
				cl *openapi.ClientWithResponses,
				sh *e2e.SessionHelper,
				aw *account_writer.Writer,
				writer *asset_writer.Writer,
				proc *ocr.Processor,
			) {
				lc.Append(fx.StartHook(func() {
					a := assert.New(t)
					r := require.New(t)

					adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
					adminSession := sh.WithSession(adminCtx)

					for range 3 {
						uploadTestAsset(t, root, cl, adminSession, "image/png", onePixelPNG)
					}

					reset, err := writer.ResetOCRForReindex(root, 10)
					r.NoError(err)
					r.NotEmpty(reset)

					processed, err := proc.ProcessAllPending(root, 10)
					r.NoError(err)
					a.GreaterOrEqual(processed, len(reset))

					remaining, err := proc.ProcessAllPending(root, 10)
					r.NoError(err)
					a.Zero(remaining, "a clamped pool must still finish the queue")
				}))
			}))
		})
	}
}
