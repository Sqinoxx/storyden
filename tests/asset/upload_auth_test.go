package asset_test

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
)

// TestBinaryUpload_Authorisation pins the authorisation of every route that
// bypasses the OpenAPI request validator so its body can be streamed. Bypassing
// the validator also bypasses its AuthenticationFunc, so each handler applies
// the check itself — if that were ever dropped, these routes would silently
// become world-writable.
func TestBinaryUpload_Authorisation(t *testing.T) {
	integration.Test(t, noOCR(), e2e.Setup(), fx.Invoke(func(
		root context.Context,
		lc fx.Lifecycle,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
	) {
		lc.Append(fx.StartHook(func() {
			adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			adminSession := sh.WithSession(adminCtx)

			// Baldur holds no admin flag, so this exercises the permission
			// check rather than just the session check.
			memberCtx, _ := e2e.WithAccount(root, aw, seed.Account_003_Baldur)
			memberSession := sh.WithSession(memberCtx)

			t.Run("icon upload rejects guests", func(t *testing.T) {
				r := require.New(t)

				resp, err := cl.IconUploadWithBodyWithResponse(root, "image/png", bytes.NewReader(onePixelPNG))
				r.NoError(err)
				r.Equal(http.StatusUnauthorized, resp.StatusCode())
			})

			t.Run("icon upload rejects non-admin members", func(t *testing.T) {
				r := require.New(t)

				resp, err := cl.IconUploadWithBodyWithResponse(root, "image/png", bytes.NewReader(onePixelPNG), memberSession)
				r.NoError(err)
				r.Equal(http.StatusForbidden, resp.StatusCode())
			})

			t.Run("banner upload rejects guests", func(t *testing.T) {
				r := require.New(t)

				resp, err := cl.BannerUploadWithBodyWithResponse(root, "image/png", bytes.NewReader(onePixelPNG))
				r.NoError(err)
				r.Equal(http.StatusUnauthorized, resp.StatusCode())
			})

			t.Run("set avatar rejects guests", func(t *testing.T) {
				r := require.New(t)

				resp, err := cl.AccountSetAvatarWithBodyWithResponse(root,
					&openapi.AccountSetAvatarParams{ContentLength: int64(len(onePixelPNG))},
					"image/png", bytes.NewReader(onePixelPNG))
				r.NoError(err)
				r.Equal(http.StatusUnauthorized, resp.StatusCode())
			})

			t.Run("set avatar accepts the account owner", func(t *testing.T) {
				r := require.New(t)

				resp, err := cl.AccountSetAvatarWithBodyWithResponse(root,
					&openapi.AccountSetAvatarParams{ContentLength: int64(len(onePixelPNG))},
					"image/png", bytes.NewReader(onePixelPNG), memberSession)
				r.NoError(err)
				r.Equal(http.StatusOK, resp.StatusCode())
			})

			t.Run("icon upload accepts admins", func(t *testing.T) {
				r := require.New(t)

				resp, err := cl.IconUploadWithBodyWithResponse(root, "image/png", bytes.NewReader(onePixelPNG), adminSession)
				r.NoError(err)
				r.Equal(http.StatusOK, resp.StatusCode())
			})
		}))
	}))
}
