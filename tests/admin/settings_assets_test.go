package admin_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/Southclaws/opt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/config"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

// TestAssetSettings covers the admin-configurable max upload size: it should
// default to the deploy-time MAX_UPLOAD_SIZE_MB config value and be editable
// (and readable back) via the admin settings API, mirroring the existing
// content/rate-limit settings coverage.
func TestAssetSettings(t *testing.T) {
	t.Parallel()

	integration.Test(t, &config.Config{MaxUploadSizeMB: 50}, e2e.Setup(), fx.Invoke(func(
		lc fx.Lifecycle,
		root context.Context,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
	) {
		lc.Append(fx.StartHook(func() {
			adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			adminSession := sh.WithSession(adminCtx)

			t.Run("defaults_from_config", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				get := tests.AssertRequest(cl.AdminSettingsGetWithResponse(adminCtx, adminSession))(t, http.StatusOK)
				r.NotNil(get.JSON200)
				r.NotNil(get.JSON200.Services)
				r.NotNil(get.JSON200.Services.Assets)
				r.NotNil(get.JSON200.Services.Assets.MaxUploadSizeMb)
				a.Equal(50, *get.JSON200.Services.Assets.MaxUploadSizeMb)
			})

			t.Run("public_info_reflects_default", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				info := tests.AssertRequest(cl.GetInfoWithResponse(root))(t, http.StatusOK)
				r.NotNil(info.JSON200)
				r.NotNil(info.JSON200.MaxUploadSizeMb)
				a.Equal(50, *info.JSON200.MaxUploadSizeMb)
			})

			t.Run("round_trip", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				update := tests.AssertRequest(cl.AdminSettingsUpdateWithResponse(adminCtx, openapi.AdminSettingsUpdateJSONRequestBody{
					Services: &openapi.AdminSettingsServiceProps{
						Assets: &openapi.AssetServiceSettings{
							MaxUploadSizeMb: opt.New(10).Ptr(),
						},
					},
				}, adminSession))(t, http.StatusOK)
				r.NotNil(update.JSON200)
				r.NotNil(update.JSON200.Services)
				r.NotNil(update.JSON200.Services.Assets)
				a.Equal(10, *update.JSON200.Services.Assets.MaxUploadSizeMb)

				get := tests.AssertRequest(cl.AdminSettingsGetWithResponse(adminCtx, adminSession))(t, http.StatusOK)
				r.NotNil(get.JSON200)
				a.Equal(10, *get.JSON200.Services.Assets.MaxUploadSizeMb)

				info := tests.AssertRequest(cl.GetInfoWithResponse(root))(t, http.StatusOK)
				r.NotNil(info.JSON200)
				a.Equal(10, *info.JSON200.MaxUploadSizeMb, "the public info endpoint must reflect admin-configured limits too, since every member relies on it for client-side validation")
			})
		}))
	}))
}
