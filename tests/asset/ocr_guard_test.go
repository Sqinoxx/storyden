package asset_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/rs/xid"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/asset"
	"github.com/Southclaws/storyden/app/resources/asset/asset_querier"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/services/ocr"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/config"
	"github.com/Southclaws/storyden/internal/infrastructure/object"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

func uploadTestAsset(t *testing.T, ctx context.Context, cl *openapi.ClientWithResponses, session openapi.RequestEditorFn, contentType string, data []byte) openapi.Asset {
	t.Helper()

	assetResp, err := cl.AssetUploadWithBodyWithResponse(
		ctx,
		&openapi.AssetUploadParams{ContentLength: int64(len(data))},
		contentType,
		bytes.NewReader(data),
		session,
	)
	tests.Ok(t, err, assetResp)

	return *assetResp.JSON200
}

// TestOCR_SkipsUnsupportedSVG covers uploading a vector image (e.g. a
// GitHub favicon): Tesseract/Leptonica can't rasterise SVGs and previously
// aborted with "Leptonica Error in pixRead: image file not found" because
// the MIME whitelist accepted any image/* type. isSupportedMIME must reject
// image/svg+xml up front instead of handing it to the OCR engine.
func TestOCR_SkipsUnsupportedSVG(t *testing.T) {
	cfg := &config.Config{
		OCREnabled:         true,
		OCRProvider:        "textlayer",
		OCRBackfillEnabled: false,
	}

	integration.Test(t, cfg, e2e.Setup(), fx.Invoke(func(
		root context.Context,
		lc fx.Lifecycle,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
		aq *asset_querier.Querier,
		proc *ocr.Processor,
	) {
		lc.Append(fx.StartHook(func() {
			r := require.New(t)

			adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			adminSession := sh.WithSession(adminCtx)

			svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10"><circle cx="5" cy="5" r="4"/></svg>`)
			a := uploadTestAsset(t, root, cl, adminSession, "image/svg+xml", svg)

			assetID, err := xid.FromString(a.Id)
			r.NoError(err)

			r.NoError(proc.ProcessAsset(root, assetID))

			result, err := aq.GetByID(root, assetID)
			r.NoError(err)
			r.Equal("skipped", result.OCRStatus)
		}))
	}))
}

// TestOCR_SkipsMissingFile covers an asset row whose underlying object was
// removed from storage (manual cleanup, storage migration, etc.) while the
// database still lists it as pending. The pre-flight existence check must
// skip it rather than erroring out with "no such file or directory" and
// leaving the backfill loop stuck retrying it.
func TestOCR_SkipsMissingFile(t *testing.T) {
	cfg := &config.Config{
		OCREnabled:         true,
		OCRProvider:        "textlayer",
		OCRBackfillEnabled: false,
	}

	integration.Test(t, cfg, e2e.Setup(), fx.Invoke(func(
		root context.Context,
		lc fx.Lifecycle,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
		aq *asset_querier.Querier,
		objects object.Storer,
		proc *ocr.Processor,
	) {
		lc.Append(fx.StartHook(func() {
			r := require.New(t)

			adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			adminSession := sh.WithSession(adminCtx)

			png := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
			a := uploadTestAsset(t, root, cl, adminSession, "image/png", png)

			assetID, err := xid.FromString(a.Id)
			r.NoError(err)

			stored, err := aq.GetByID(root, assetID)
			r.NoError(err)

			// Upload also fires an async background OCR pass via the bus.
			// Retry the delete: on Windows it fails with "used by another
			// process" until that goroutine's file handle is released.
			path := asset.BuildAssetPath(stored.Name)
			r.Eventually(func() bool {
				return objects.Delete(root, path) == nil
			}, 2*time.Second, 20*time.Millisecond, "expected asset file to become deletable")

			r.NoError(proc.ProcessAsset(root, assetID))

			result, err := aq.GetByID(root, assetID)
			r.NoError(err)
			r.Equal("skipped", result.OCRStatus)
		}))
	}))
}
