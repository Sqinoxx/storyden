package email_verification_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Southclaws/opt"
	"github.com/google/uuid"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/infrastructure/mailer"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

const (
	problemEmailNotVerified = "urn:storyden:problem:email-not-verified"
	problemPermissionDenied = "urn:storyden:problem:permission-denied"
)

func problemType(t *testing.T, body []byte) string {
	t.Helper()

	var problem struct {
		Type string `json:"type"`
	}
	require.NoError(t, json.Unmarshal(body, &problem), "response body was not a problem document: %s", body)

	return problem.Type
}

func TestEmailVerificationGate(t *testing.T) {
	if tests.IsSharedPostgresDatabase() {
		t.Skip("skipping email verification gate test on shared postgres database")
	}

	t.Parallel()

	integration.Test(t, nil, e2e.Setup(), fx.Invoke(func(
		lc fx.Lifecycle,
		root context.Context,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
		mail mailer.Sender,
	) {
		inbox := mail.(*mailer.Mock)

		lc.Append(fx.StartHook(func() {
			adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			adminSession := sh.WithSession(adminCtx)

			emailMode := openapi.Email
			_, err := cl.AdminSettingsUpdateWithResponse(adminCtx, openapi.AdminSettingsUpdateJSONRequestBody{
				AuthenticationMode: &emailMode,
			}, adminSession)
			require.NoError(t, err)

			cat := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
				Colour:      "#fe4efd",
				Description: "gate test",
				Name:        "Gate " + uuid.NewString(),
			}, adminSession))(t, http.StatusOK)

			signUpUnverified := func(t *testing.T) (string, openapi.RequestEditorFn) {
				t.Helper()

				address := xid.New().String() + "@storyden.org"
				signup, err := cl.AuthEmailSignupWithResponse(root, nil, openapi.AuthEmailSignupJSONRequestBody{Email: address})
				tests.Ok(t, err, signup)

				accountID := account.AccountID(openapi.GetAccountID(signup.JSON200.Id))

				return address, sh.WithSession(e2e.WithAccountID(root, accountID))
			}

			t.Run("unverified_403_carries_a_distinct_problem_type", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				_, session := signUpUnverified(t)

				resp, err := cl.ThreadCreateWithResponse(root, openapi.ThreadInitialProps{
					Body:       opt.New("<p>blocked</p>").Ptr(),
					Category:   opt.New(cat.JSON200.Id).Ptr(),
					Visibility: opt.New(openapi.VisibilityPublished).Ptr(),
					Title:      "Should be blocked",
				}, session)
				r.NoError(err)
				r.Equal(http.StatusForbidden, resp.StatusCode())

				a.Equal(problemEmailNotVerified, problemType(t, resp.Body),
					"the frontend keys off this to offer a resend instead of a raw error")
			})

			t.Run("genuine_permission_failures_are_not_mislabelled", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				// A verified, non-admin member lacking the permission must still
				// report a plain permission denial. The account has to be marked
				// verified first, otherwise the email gate legitimately applies
				// and this would not be testing what it claims to.
				memberCtx, member := e2e.WithAccount(root, aw, seed.Account_004_Loki)
				_, err := aw.Update(root, member.ID, account_writer.SetVerifiedStatus(account.VerifiedStatusManual))
				r.NoError(err)

				memberSession := sh.WithSession(memberCtx)

				resp, err := cl.AdminSettingsUpdateWithResponse(root, openapi.AdminSettingsUpdateJSONRequestBody{
					AuthenticationMode: &emailMode,
				}, memberSession)
				r.NoError(err)

				r.Equal(http.StatusForbidden, resp.StatusCode())

				a.Equal(problemPermissionDenied, problemType(t, resp.Body))
			})

			t.Run("resend_sends_a_new_code_for_a_known_address", func(t *testing.T) {
				a := assert.New(t)

				address, session := signUpUnverified(t)
				before := inbox.Count()

				tests.AssertRequest(cl.AuthEmailVerifyResendWithResponse(root, openapi.AuthEmailVerifyResendJSONRequestBody{
					Email: (*openapi.EmailAddress)(&address),
				}, session))(t, http.StatusOK)

				a.NotNil(tests.WaitForNextEmail(t, inbox, before))
			})

			t.Run("resend_without_a_body_address_uses_the_session_account", func(t *testing.T) {
				a := assert.New(t)

				_, session := signUpUnverified(t)
				before := inbox.Count()

				tests.AssertRequest(cl.AuthEmailVerifyResendWithResponse(root,
					openapi.AuthEmailVerifyResendJSONRequestBody{}, session))(t, http.StatusOK)

				a.NotNil(tests.WaitForNextEmail(t, inbox, before))
			})

			t.Run("resend_does_not_leak_whether_an_address_is_registered", func(t *testing.T) {
				unknown := openapi.EmailAddress(xid.New().String() + "@storyden.org")

				tests.AssertRequest(cl.AuthEmailVerifyResendWithResponse(root, openapi.AuthEmailVerifyResendJSONRequestBody{
					Email: &unknown,
				}))(t, http.StatusOK)
			})
		}))
	}))
}
