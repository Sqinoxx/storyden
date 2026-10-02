package asset_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/Southclaws/opt"
	"github.com/rs/xid"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/services/ocr"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/config"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

func TestOCR_AdminMaxFileSizeAndAssetList(t *testing.T) {
	cfg := &config.Config{
		OCREnabled:         true,
		OCRProvider:        "textlayer",
		OCRBackfillEnabled: false,
		OCRMaxFileSizeMB:   10,
	}

	integration.Test(t, cfg, e2e.Setup(), fx.Invoke(func(
		root context.Context,
		lc fx.Lifecycle,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
		proc *ocr.Processor,
	) {
		lc.Append(fx.StartHook(func() {
			r := require.New(t)

			adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			adminSession := sh.WithSession(adminCtx)
			memberCtx, _ := e2e.WithAccount(root, aw, seed.Account_003_Baldur)
			memberSession := sh.WithSession(memberCtx)

			get := tests.AssertRequest(cl.AdminSettingsGetWithResponse(root, adminSession))(t, http.StatusOK)
			r.Equal(10, *get.JSON200.Services.Assets.OcrMaxFileSizeMb)

			update := tests.AssertRequest(cl.AdminSettingsUpdateWithResponse(root, openapi.AdminSettingsUpdateJSONRequestBody{
				Services: &openapi.AdminSettingsServiceProps{
					Assets: &openapi.AssetServiceSettings{OcrMaxFileSizeMb: opt.New(1).Ptr()},
				},
			}, adminSession))(t, http.StatusOK)
			r.Equal(1, *update.JSON200.Services.Assets.OcrMaxFileSizeMb)
			r.Equal(get.JSON200.Services.Assets.MaxUploadSizeMb, update.JSON200.Services.Assets.MaxUploadSizeMb, "updating the OCR limit must not reset the upload limit")

			big := make([]byte, 2*1024*1024)
			copy(big, []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
			a := uploadTestAsset(t, root, cl, adminSession, "image/png", big)

			assetID, err := xid.FromString(a.Id)
			r.NoError(err)
			r.NoError(proc.ProcessAsset(root, assetID))

			skipped := openapi.AdminOCRAssetListParamsStatus("skipped")
			list := tests.AssertRequest(cl.AdminOCRAssetListWithResponse(root, &openapi.AdminOCRAssetListParams{Status: &skipped}, adminSession))(t, http.StatusOK)

			var found bool
			for _, item := range list.JSON200.Assets {
				if item.Id == a.Id {
					found = true
					r.Equal("skipped", item.Status)
					r.NotNil(item.Error)
					r.Equal("file exceeds max size of 1 MB", *item.Error)
					r.Equal(len(big), item.Size)
				}
			}
			r.True(found, "skipped asset should be listed")

			completed := openapi.AdminOCRAssetListParamsStatus("completed")
			list = tests.AssertRequest(cl.AdminOCRAssetListWithResponse(root, &openapi.AdminOCRAssetListParams{Status: &completed}, adminSession))(t, http.StatusOK)
			for _, item := range list.JSON200.Assets {
				r.NotEqual(a.Id, item.Id)
			}

			tests.AssertRequest(cl.AdminOCRAssetListWithResponse(root, nil, memberSession))(t, http.StatusForbidden)
		}))
	}))
}
