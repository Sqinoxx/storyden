package asset_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"net/http"
	"testing"
	"time"

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

// TestAsset_UploadSizeLimit pins the whole feature end to end: an admin
// lowering the configured max upload size takes effect on the already-running
// server (no restart), a file over that limit is rejected with a descriptive
// 413 rather than a generic 500, and a file under it still succeeds.
func TestAsset_UploadSizeLimit(t *testing.T) {
	integration.Test(t, &config.Config{MaxUploadSizeMB: 50, OCREnabled: false, OCRBackfillEnabled: false}, e2e.Setup(), fx.Invoke(func(
		root context.Context,
		lc fx.Lifecycle,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
	) {
		lc.Append(fx.StartHook(func() {
			r := require.New(t)
			a := assert.New(t)

			ctx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			session := sh.WithSession(ctx)
			adminSession := session

			// Lower the limit from the config default of 50MB to 1MB.
			tests.AssertRequest(cl.AdminSettingsUpdateWithResponse(root, openapi.AdminSettingsUpdateJSONRequestBody{
				Services: &openapi.AdminSettingsServiceProps{
					Assets: &openapi.AssetServiceSettings{
						MaxUploadSizeMb: opt.New(1).Ptr(),
					},
				},
			}, adminSession))(t, http.StatusOK)

			// The limiter middleware reconfigures asynchronously off the
			// settings-updated pubsub event.
			time.Sleep(100 * time.Millisecond)

			t.Run("oversized upload is rejected with a descriptive 413", func(t *testing.T) {
				oversized := randomBytes(t, 2*1024*1024)

				resp, err := cl.AssetUploadWithBodyWithResponse(
					root,
					&openapi.AssetUploadParams{
						ContentLength: int64(len(oversized)),
						Filename:      ptr("too-big.bin"),
					},
					"application/octet-stream",
					bytes.NewReader(oversized),
					session,
				)
				r.NoError(err)
				a.Equal(http.StatusRequestEntityTooLarge, resp.StatusCode())
				r.NotNil(resp.JSONDefault)
				r.NotNil(resp.JSONDefault.Detail)
				a.Contains(*resp.JSONDefault.Detail, "1 MB")
			})

			t.Run("upload within the lowered limit still succeeds", func(t *testing.T) {
				withinLimit := randomBytes(t, 256*1024)

				resp, err := cl.AssetUploadWithBodyWithResponse(
					root,
					&openapi.AssetUploadParams{
						ContentLength: int64(len(withinLimit)),
						Filename:      ptr("fits.bin"),
					},
					"application/octet-stream",
					bytes.NewReader(withinLimit),
					session,
				)
				tests.Ok(t, err, resp)
			})
		}))
	}))
}

func randomBytes(t *testing.T, n int) []byte {
	t.Helper()

	b := make([]byte, n)
	_, err := rand.Read(b)
	require.NoError(t, err)

	return b
}
