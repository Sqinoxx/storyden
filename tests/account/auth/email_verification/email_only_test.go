package email_verification_test

import (
	"context"
	"net/http"
	"regexp"
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/app/resources/account/account_querier"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/infrastructure/mailer"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

func TestEmailOnlyAuth(t *testing.T) {
	t.Parallel()

	integration.Test(t, nil, e2e.Setup(), fx.Invoke(func(
		lc fx.Lifecycle,
		root context.Context,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		accountQuery *account_querier.Querier,
		mail mailer.Sender,
	) {
		inbox := mail.(*mailer.Mock)

		lc.Append(fx.StartHook(func() {
			t.Run("verify_success", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				address := xid.New().String() + "@storyden.org"
				emailCount := inbox.Count()

				// Sign up with email
				signup, err := cl.AuthEmailSignupWithResponse(root, nil, openapi.AuthEmailSignupJSONRequestBody{Email: address})
				tests.Ok(t, err, signup)

				accountID := account.AccountID(openapi.GetAccountID(signup.JSON200.Id))
				ctx1 := e2e.WithAccountID(root, accountID)
				session := sh.WithSession(ctx1)

				// Get own account, currently unverified
				unverified, err := cl.AccountGetWithResponse(root, session)
				tests.Ok(t, err, unverified)
				r.Equal(openapi.AccountVerifiedStatusNone, unverified.JSON200.VerifiedStatus)
				r.Len(unverified.JSON200.EmailAddresses, 1)
				a.Equal(address, (unverified.JSON200.EmailAddresses)[0].EmailAddress)
				a.False(unverified.JSON200.EmailAddresses[0].Verified)

				// Get code from email, verify account
				verification := tests.WaitForNextEmail(t, inbox, emailCount)
				a.Equal(unverified.JSON200.Name, verification.Name)
				a.Equal(address, verification.Address.Address)
				code := regexp.MustCompile(`verify your account: ([0-9]{6})`).FindStringSubmatch(verification.Plain)[1]
				verify, err := cl.AuthEmailVerifyWithResponse(root, openapi.AuthEmailVerifyJSONRequestBody{Email: address, Code: code}, session)
				tests.Ok(t, err, verify)
				a.Equal(accountID.String(), verify.JSON200.Id)

				// Get own account, now verified
				verified, err := cl.AccountGetWithResponse(root, session)
				tests.Ok(t, err, verified)
				a.Equal(openapi.AccountVerifiedStatusVerifiedEmail, verified.JSON200.VerifiedStatus)
				r.NotNil(verified.JSON200.EmailAddresses)
				a.Equal(address, verified.JSON200.EmailAddresses[0].EmailAddress)
				a.True(verified.JSON200.EmailAddresses[0].Verified)
			})

			t.Run("verify_resend", func(t *testing.T) {
				// r := require.New(t)
				a := assert.New(t)

				address := xid.New().String() + "@storyden.org"

				// Sign up with email
				signup, err := cl.AuthEmailSignupWithResponse(root, nil, openapi.AuthEmailSignupJSONRequestBody{Email: address})
				tests.Ok(t, err, signup)

				// Sign up with email, again, resulting in a 202 Accepted and no cookie session
				signup2, err := cl.AuthEmailSignupWithResponse(root, nil, openapi.AuthEmailSignupJSONRequestBody{Email: address})
				tests.Status(t, err, signup2, http.StatusUnprocessableEntity)

				a.Empty(signup2.HTTPResponse.Header.Get("Set-Cookie"))
			})

			t.Run("verify_wrong_code", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				address := xid.New().String() + "@storyden.org"

				// Sign up with email
				signup, err := cl.AuthEmailSignupWithResponse(root, nil, openapi.AuthEmailSignupJSONRequestBody{Email: address})
				tests.Ok(t, err, signup)

				accountID := account.AccountID(openapi.GetAccountID(signup.JSON200.Id))
				ctx1 := e2e.WithAccountID(root, accountID)
				session := sh.WithSession(ctx1)

				// Get own account, currently unverified
				unverified, err := cl.AccountGetWithResponse(root, session)
				tests.Ok(t, err, unverified)
				r.Equal(openapi.AccountVerifiedStatusNone, unverified.JSON200.VerifiedStatus)

				incorrectCode := "999999" // one day, this test will fail...
				verify, err := cl.AuthEmailVerifyWithResponse(root, openapi.AuthEmailVerifyJSONRequestBody{Email: address, Code: incorrectCode}, session)
				tests.Status(t, err, verify, http.StatusUnauthorized)

				// Get own account, still not verified
				verified, err := cl.AccountGetWithResponse(root, session)
				tests.Ok(t, err, verified)
				a.Equal(openapi.AccountVerifiedStatusNone, verified.JSON200.VerifiedStatus)
			})

			t.Run("verify_code_cannot_be_reused", func(t *testing.T) {
				r := require.New(t)

				address := xid.New().String() + "@storyden.org"
				emailCount := inbox.Count()

				signup, err := cl.AuthEmailSignupWithResponse(root, nil, openapi.AuthEmailSignupJSONRequestBody{Email: address})
				tests.Ok(t, err, signup)

				accountID := account.AccountID(openapi.GetAccountID(signup.JSON200.Id))
				session := sh.WithSession(e2e.WithAccountID(root, accountID))

				verification := tests.WaitForNextEmail(t, inbox, emailCount)
				code := regexp.MustCompile(`verify your account: ([0-9]{6})`).FindStringSubmatch(verification.Plain)[1]

				first, err := cl.AuthEmailVerifyWithResponse(root, openapi.AuthEmailVerifyJSONRequestBody{Email: address, Code: code}, session)
				tests.Ok(t, err, first)

				// B5: the same code must not verify a second time.
				second, err := cl.AuthEmailVerifyWithResponse(root, openapi.AuthEmailVerifyJSONRequestBody{Email: address, Code: code}, session)
				r.NoError(err)
				r.Equal(http.StatusUnauthorized, second.StatusCode())
			})

			t.Run("verify_locks_out_after_too_many_wrong_attempts", func(t *testing.T) {
				r := require.New(t)

				address := xid.New().String() + "@storyden.org"
				emailCount := inbox.Count()

				signup, err := cl.AuthEmailSignupWithResponse(root, nil, openapi.AuthEmailSignupJSONRequestBody{Email: address})
				tests.Ok(t, err, signup)

				accountID := account.AccountID(openapi.GetAccountID(signup.JSON200.Id))
				session := sh.WithSession(e2e.WithAccountID(root, accountID))

				verification := tests.WaitForNextEmail(t, inbox, emailCount)
				code := regexp.MustCompile(`verify your account: ([0-9]{6})`).FindStringSubmatch(verification.Plain)[1]

				for i := 0; i < 5; i++ {
					resp, err := cl.AuthEmailVerifyWithResponse(root, openapi.AuthEmailVerifyJSONRequestBody{Email: address, Code: "000000"}, session)
					r.NoError(err)
					r.Equal(http.StatusUnauthorized, resp.StatusCode())
				}

				// B5: even the correct code is rejected once the attempt
				// budget for it is spent - a new one must be requested.
				resp, err := cl.AuthEmailVerifyWithResponse(root, openapi.AuthEmailVerifyJSONRequestBody{Email: address, Code: code}, session)
				r.NoError(err)
				r.Equal(http.StatusUnauthorized, resp.StatusCode())
			})
		}))
	}))
}
