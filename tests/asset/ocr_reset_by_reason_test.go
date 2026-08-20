package asset_test

import (
	"context"
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/asset/asset_querier"
	"github.com/Southclaws/storyden/app/resources/asset/asset_writer"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/config"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
)

// This is exactly the failure mode that motivated ResetOCRByErrorPrefix: a
// run with the wrong environment (no OCR binary on PATH, a size cap too low)
// marks assets "skipped" for a reason that has nothing to do with the file
// itself, and "skipped" is otherwise a terminal status the normal queue never
// revisits. Genuinely unsupported files (an .ico, a .zip) must stay skipped.
func TestOCR_ResetByErrorPrefixRequeuesOnlyEnvironmentReasons(t *testing.T) {
	cfg := &config.Config{OCREnabled: false, OCRBackfillEnabled: false}

	integration.Test(t, cfg, e2e.Setup(), fx.Invoke(func(
		root context.Context,
		lc fx.Lifecycle,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
		aq *asset_querier.Querier,
		writer *asset_writer.Writer,
	) {
		lc.Append(fx.StartHook(func() {
			a := assert.New(t)
			r := require.New(t)

			adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			adminSession := sh.WithSession(adminCtx)

			engineUnavailable := uploadTestAsset(t, root, cl, adminSession, "image/png", onePixelPNG)
			tooLarge := uploadTestAsset(t, root, cl, adminSession, "image/png", encodePNG(4, 4))
			unsupportedMIME := uploadTestAsset(t, root, cl, adminSession, "application/zip", []byte("PK\x03\x04not a real zip but sniffable"))

			engineID, err := xid.FromString(engineUnavailable.Id)
			r.NoError(err)
			largeID, err := xid.FromString(tooLarge.Id)
			r.NoError(err)
			mimeID, err := xid.FromString(unsupportedMIME.Id)
			r.NoError(err)

			_, err = writer.UpdateOCRSkipped(root, engineID, "ocr engine unavailable")
			r.NoError(err)
			_, err = writer.UpdateOCRSkipped(root, largeID, "file exceeds max size of 10 MB")
			r.NoError(err)
			_, err = writer.UpdateOCRSkipped(root, mimeID, "unsupported mime type: application/zip")
			r.NoError(err)

			reset, err := writer.ResetOCRByErrorPrefix(root, []string{"ocr engine unavailable", "file exceeds max size of"})
			r.NoError(err)
			a.Len(reset, 2)

			engineAfter, err := aq.GetByID(root, engineID)
			r.NoError(err)
			a.Equal("pending", engineAfter.OCRStatus)
			a.False(engineAfter.OCRError.Ok(), "the stale error must be cleared so it doesn't linger after requeue")

			largeAfter, err := aq.GetByID(root, largeID)
			r.NoError(err)
			a.Equal("pending", largeAfter.OCRStatus)

			mimeAfter, err := aq.GetByID(root, mimeID)
			r.NoError(err)
			a.Equal("skipped", mimeAfter.OCRStatus, "a genuinely unsupported file type must not be requeued")
			a.Equal("unsupported mime type: application/zip", mimeAfter.OCRError.OrZero())
		}))
	}))
}

func TestOCR_ResetByErrorPrefixIsANoOpWhenNothingMatches(t *testing.T) {
	cfg := &config.Config{OCREnabled: false, OCRBackfillEnabled: false}

	integration.Test(t, cfg, e2e.Setup(), fx.Invoke(func(
		root context.Context,
		lc fx.Lifecycle,
		writer *asset_writer.Writer,
	) {
		lc.Append(fx.StartHook(func() {
			r := require.New(t)

			reset, err := writer.ResetOCRByErrorPrefix(root, []string{"ocr engine unavailable", "file exceeds max size of"})
			r.NoError(err)
			r.Empty(reset)
		}))
	}))
}
