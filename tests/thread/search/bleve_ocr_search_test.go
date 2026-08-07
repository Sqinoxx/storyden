package search_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/Southclaws/opt"
	"github.com/google/uuid"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/config"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

func TestThreadGetIncludesAttachedAssets(t *testing.T) {
	cfg := &config.Config{}

	integration.Test(t, cfg, e2e.Setup(), fx.Invoke(func(
		root context.Context,
		lc fx.Lifecycle,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
	) {
		lc.Append(fx.StartHook(func() {
			adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			adminSession := sh.WithSession(adminCtx)

			catResp, err := cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
				Name:   "thread-assets-" + uuid.NewString(),
				Colour: "#123456",
			}, adminSession)
			tests.Ok(t, err, catResp)

			pngData := []byte{
				0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
				0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
				0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
				0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
				0x89, 0x00, 0x00, 0x00, 0x0a, 0x49, 0x44, 0x41,
				0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
				0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00,
				0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
				0x42, 0x60, 0x82,
			}

			assetResp, err := cl.AssetUploadWithBodyWithResponse(
				root,
				&openapi.AssetUploadParams{ContentLength: int64(len(pngData))},
				"image/png",
				bytes.NewReader(pngData),
				adminSession,
			)
			tests.Ok(t, err, assetResp)

			assetIDs := openapi.AssetIDs{assetResp.JSON200.Id}
			threadResp, err := cl.ThreadCreateWithResponse(root, openapi.ThreadInitialProps{
				Title:      "Thread with asset " + uuid.NewString(),
				Body:       opt.New("<p>This thread verifies attached asset hydration.</p>").Ptr(),
				Category:   opt.New(catResp.JSON200.Id).Ptr(),
				Visibility: opt.New(openapi.VisibilityPublished).Ptr(),
				AssetIds:   &assetIDs,
			}, adminSession)
			tests.Ok(t, err, threadResp)

			threadGet, err := cl.ThreadGetWithResponse(root, threadResp.JSON200.Slug, nil, adminSession)
			tests.Ok(t, err, threadGet)
			if t.Failed() {
				return
			}

			tests.Ok(t, err, threadGet)
			if threadGet.JSON200 == nil {
				t.Fatalf("expected thread response")
			}
			if len(threadGet.JSON200.Assets) == 0 {
				t.Fatalf("expected thread assets to be hydrated")
			}
			if threadGet.JSON200.Assets[0].Id != assetResp.JSON200.Id {
				t.Fatalf("expected attached asset %s, got %s", assetResp.JSON200.Id, threadGet.JSON200.Assets[0].Id)
			}
		}))
	}))
}