package username_password_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

func TestLoginLockoutSettings(t *testing.T) {
	t.Parallel()

	integration.Test(t, nil, e2e.Setup(), fx.Invoke(func(
		lc fx.Lifecycle,
		root context.Context,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
	) {
		lc.Append(fx.StartHook(func() {
			adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			adminSession := sh.WithSession(adminCtx)

			failLogins := func(t *testing.T, handle string, n int) {
				for i := 0; i < n; i++ {
					resp, err := cl.AuthPasswordSigninWithResponse(root, openapi.AuthPair{Identifier: handle, Token: "wrongpassword"})
					require.NoError(t, err)
					require.Equal(t, http.StatusUnauthorized, resp.StatusCode())
				}
			}

			signIn := func(t *testing.T, handle string) int {
				resp, err := cl.AuthPasswordSigninWithResponse(root, openapi.AuthPair{Identifier: handle, Token: "correctpassword"})
				require.NoError(t, err)
				return resp.StatusCode()
			}

			signUp := func(t *testing.T) string {
				handle := xid.New().String()
				resp, err := cl.AuthPasswordSignupWithResponse(root, nil, openapi.AuthPair{Identifier: handle, Token: "correctpassword"})
				tests.Ok(t, err, resp)
				return handle
			}

			t.Run("defaults_are_exposed", func(t *testing.T) {
				r := require.New(t)

				resp, err := cl.AdminSettingsGetWithResponse(adminCtx, adminSession)
				tests.Ok(t, err, resp)

				rl := resp.JSON200.Services.RateLimiting
				r.NotNil(rl)
				r.Equal(5, *rl.LoginMaxAttempts)
				r.Equal(900, *rl.LoginLockoutDuration)
			})

			t.Run("rejects_zero_attempts", func(t *testing.T) {
				r := require.New(t)

				zero := 0
				resp, err := cl.AdminSettingsUpdateWithResponse(adminCtx, openapi.AdminSettingsUpdateJSONRequestBody{
					Services: &openapi.AdminSettingsServiceProps{
						RateLimiting: &openapi.RateLimitServiceSettings{LoginMaxAttempts: &zero},
					},
				}, adminSession)
				r.NoError(err)
				r.Equal(http.StatusBadRequest, resp.StatusCode())
			})

			t.Run("lower_threshold_locks_out_sooner", func(t *testing.T) {
				r := require.New(t)

				maxAttempts := 2
				lockout := 600
				update, err := cl.AdminSettingsUpdateWithResponse(adminCtx, openapi.AdminSettingsUpdateJSONRequestBody{
					Services: &openapi.AdminSettingsServiceProps{
						RateLimiting: &openapi.RateLimitServiceSettings{
							LoginMaxAttempts:     &maxAttempts,
							LoginLockoutDuration: &lockout,
						},
					},
				}, adminSession)
				tests.Ok(t, err, update)

				rl := update.JSON200.Services.RateLimiting
				r.Equal(2, *rl.LoginMaxAttempts)
				r.Equal(600, *rl.LoginLockoutDuration)

				belowLimit := signUp(t)
				failLogins(t, belowLimit, 1)
				r.Equal(http.StatusOK, signIn(t, belowLimit))

				locked := signUp(t)
				failLogins(t, locked, 2)
				r.Equal(http.StatusTooManyRequests, signIn(t, locked))
			})

			t.Run("higher_threshold_allows_more_attempts", func(t *testing.T) {
				r := require.New(t)

				maxAttempts := 8
				update, err := cl.AdminSettingsUpdateWithResponse(adminCtx, openapi.AdminSettingsUpdateJSONRequestBody{
					Services: &openapi.AdminSettingsServiceProps{
						RateLimiting: &openapi.RateLimitServiceSettings{LoginMaxAttempts: &maxAttempts},
					},
				}, adminSession)
				tests.Ok(t, err, update)
				r.Equal(600, *update.JSON200.Services.RateLimiting.LoginLockoutDuration)

				handle := signUp(t)
				failLogins(t, handle, 7)
				r.Equal(http.StatusOK, signIn(t, handle))

				failLogins(t, handle, 8)
				r.Equal(http.StatusTooManyRequests, signIn(t, handle))
			})
		}))
	}))
}
