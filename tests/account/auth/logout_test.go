package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_querier"
	"github.com/Southclaws/storyden/app/resources/account/token"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

func TestLogout(t *testing.T) {
	t.Parallel()

	integration.Test(t, nil, e2e.Setup(), fx.Invoke(func(
		lc fx.Lifecycle,
		root context.Context,
		ts *httptest.Server,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		accountQuery *account_querier.Querier,
		tokenRepo token.Repository,
	) {
		lc.Append(fx.StartHook(func() {
			t.Run("logout_revokes_session_token", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				handle := xid.New().String()
				password := "password"

				// Sign up with username + password
				signup, err := cl.AuthPasswordSignupWithResponse(root, nil, openapi.AuthPasswordSignupJSONRequestBody{
					Identifier: handle,
					Token:      password,
				})
				tests.Ok(t, err, signup)

				// Sign in with username + password to get a session
				signin, err := cl.AuthPasswordSigninWithResponse(root, openapi.AuthPasswordSigninJSONRequestBody{
					Identifier: handle,
					Token:      password,
				})
				tests.Ok(t, err, signin)
				a.NotEmpty(signin.HTTPResponse.Header.Get("Set-Cookie"))
				session := e2e.WithSessionFromHeader(t, root, signin.HTTPResponse.Header)

				// Verify we can access authenticated endpoints
				get1, err := cl.AccountGetWithResponse(root, session)
				tests.Ok(t, err, get1)
				a.Equal(handle, get1.JSON200.Handle)

				cookie, err := http.ParseSetCookie(signin.HTTPResponse.Header.Get("Set-Cookie"))
				r.NoError(err)
				sessionToken, err := token.FromString(cookie.Value)
				r.NoError(err)

				// Verify the token is valid before logout
				_, err = tokenRepo.Validate(root, sessionToken)
				r.NoError(err, "Token should be valid before logout")

				// Log out - invalidating the token.
				// NOTE: This endpoint performs a redirect to WEB_ADDRESS, but
				// I'm too lazy to set up no-redirect client behavior here.
				_, _ = cl.AuthProviderLogoutWithResponse(root, nil, session)

				// After logout, the token should be revoked in the database
				_, err = tokenRepo.Validate(root, sessionToken)
				r.Error(err, "Token should be revoked after logout")
				a.Contains(err.Error(), "token revoked", "Error should indicate token was revoked")

				// Attempting to use the old session should fail
				get2, err := cl.AccountGetWithResponse(root, session)
				r.NoError(err)
				r.NotNil(get2)
				a.Equal(http.StatusUnauthorized, get2.StatusCode())

				// Logging out again with the revoked session must still succeed.
				res := postLogout(t, ts, signin.HTTPResponse.Cookies()...)
				a.Equal(http.StatusFound, res.StatusCode)
			})

			t.Run("logout_without_session_redirects", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				res := postLogout(t, ts)

				r.Equal(http.StatusFound, res.StatusCode)
				a.Contains(res.Header.Get("Location"), "/logout")

				cookie, err := http.ParseSetCookie(res.Header.Get("Set-Cookie"))
				r.NoError(err)
				a.Empty(cookie.Value)
			})
		}))
	}))
}

func postLogout(t *testing.T, ts *httptest.Server, cookies ...*http.Cookie) *http.Response {
	t.Helper()

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/auth/logout?redirect=%2Flogout", nil)
	require.NoError(t, err)
	for _, c := range cookies {
		req.AddCookie(c)
	}

	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	res, err := client.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { res.Body.Close() })

	return res
}
