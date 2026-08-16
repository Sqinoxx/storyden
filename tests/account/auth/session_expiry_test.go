package auth_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/token"
	"github.com/Southclaws/storyden/app/resources/settings"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

// sessionCookie extracts the session cookie from a response.
func sessionCookie(t *testing.T, resp *http.Response) *http.Cookie {
	t.Helper()

	for _, c := range resp.Cookies() {
		if c.Name == "storyden-session" {
			return c
		}
	}

	t.Fatalf("no session cookie in response")
	return nil
}

func TestSessionExpiry(t *testing.T) {
	t.Parallel()

	integration.Test(t, nil, e2e.Setup(), fx.Invoke(func(
		lc fx.Lifecycle,
		root context.Context,
		cl *openapi.ClientWithResponses,
		tokenRepo token.Repository,
	) {
		lc.Append(fx.StartHook(func() {
			signUp := func(t *testing.T, rememberMe *bool) *http.Response {
				t.Helper()

				resp, err := cl.AuthPasswordSignupWithResponse(root, nil, openapi.AuthPair{
					Identifier: xid.New().String(),
					Token:      "password",
					RememberMe: rememberMe,
				})
				tests.Ok(t, err, resp)

				return resp.HTTPResponse
			}

			t.Run("without_remember_me_the_cookie_dies_with_the_browser", func(t *testing.T) {
				a := assert.New(t)

				cookie := sessionCookie(t, signUp(t, nil))

				// No Expires and no Max-Age is what makes a browser-session
				// cookie, which is the whole point of the default.
				a.True(cookie.Expires.IsZero(), "expected no Expires, got %v", cookie.Expires)
				a.Zero(cookie.MaxAge, "expected no Max-Age, got %d", cookie.MaxAge)
				a.True(cookie.HttpOnly)
				a.True(cookie.Secure)
			})

			t.Run("without_remember_me_the_session_expires_after_the_idle_timeout", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				cookie := sessionCookie(t, signUp(t, nil))

				tok, err := token.FromString(cookie.Value)
				r.NoError(err)

				validated, err := tokenRepo.Validate(root, tok)
				r.NoError(err)

				a.WithinDuration(time.Now().Add(settings.DefaultSessionIdleTimeout), validated.ExpiresAt, time.Minute)
				a.Equal(settings.DefaultSessionIdleTimeout, validated.Lifetime)
				a.False(validated.Persistent)
			})

			t.Run("with_remember_me_the_cookie_persists", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				remember := true
				cookie := sessionCookie(t, signUp(t, &remember))

				a.False(cookie.Expires.IsZero(), "a remembered session needs an Expires")
				a.Positive(cookie.MaxAge, "a remembered session needs a Max-Age")

				tok, err := token.FromString(cookie.Value)
				r.NoError(err)

				validated, err := tokenRepo.Validate(root, tok)
				r.NoError(err)

				a.True(validated.Persistent)
				a.Equal(settings.DefaultRememberMeDuration, validated.Lifetime)
				a.WithinDuration(time.Now().Add(settings.DefaultRememberMeDuration), validated.ExpiresAt, time.Minute)
			})

			t.Run("refresh_slides_the_window_forward", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				cookie := sessionCookie(t, signUp(t, nil))

				tok, err := token.FromString(cookie.Value)
				r.NoError(err)

				before, err := tokenRepo.Validate(root, tok)
				r.NoError(err)

				refreshed, err := tokenRepo.Refresh(root, tok)
				r.NoError(err)

				a.False(refreshed.ExpiresAt.Before(before.ExpiresAt), "expiry moved backwards")
				a.True(refreshed.RefreshedAt.Ok())
				a.Equal(before.Lifetime, refreshed.Lifetime, "refreshing must not change the window size")
			})

			t.Run("refresh_is_throttled", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				cookie := sessionCookie(t, signUp(t, nil))

				tok, err := token.FromString(cookie.Value)
				r.NoError(err)

				issued, err := tokenRepo.Validate(root, tok)
				r.NoError(err)

				// Just issued, so the anchor is fresh and no write is due. This
				// is what keeps a busy client from writing once per request.
				a.False(token.Session(*issued).NeedsRefresh(time.Now(), token.RefreshInterval))

				// Past the interval it becomes due again.
				a.True(token.Session(*issued).NeedsRefresh(
					time.Now().Add(2*token.RefreshInterval), token.RefreshInterval))
			})

			t.Run("legacy_sessions_keep_their_expiry", func(t *testing.T) {
				a := assert.New(t)

				// Rows written before configurable lifetimes have none recorded.
				legacy := token.Session{
					Token:     token.Generate(),
					ExpiresAt: time.Now().Add(90 * 24 * time.Hour),
				}

				a.True(legacy.IsLegacy())
				a.False(legacy.NeedsRefresh(time.Now().Add(365*24*time.Hour), token.RefreshInterval),
					"deploying sliding sessions must not shorten sessions issued before it")
			})
		}))
	}))
}
