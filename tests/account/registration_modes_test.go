package account_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

func TestRestrictedRegistrationModes(t *testing.T) {
	if tests.IsSharedPostgresDatabase() {
		t.Skip("skipping registration mode test on shared postgres database")
	}

	t.Parallel()

	integration.Test(t, nil, e2e.Setup(), fx.Invoke(func(
		lc fx.Lifecycle,
		root context.Context,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
	) {
		lc.Append(fx.StartHook(func() {
			adminCtx, admin := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			adminSession := sh.WithSession(adminCtx)

			t.Run("public_allows_registration_without_invitation", func(t *testing.T) {
				mode := openapi.RegistrationModePublic
				tests.AssertRequest(
					cl.AdminSettingsUpdateWithResponse(root, openapi.AdminSettingsUpdateJSONRequestBody{
						RegistrationMode: &mode,
					}, adminSession),
				)(t, http.StatusOK)

				signup := tests.AssertRequest(
					cl.AuthPasswordSignupWithResponse(root, nil, openapi.AuthPair{
						Identifier: "public-" + xid.New().String(),
						Token:      "password",
					}),
				)(t, http.StatusOK)
				require.NotNil(t, signup.JSON200)
				assert.NotEmpty(t, signup.HTTPResponse.Header.Get("Set-Cookie"))
			})

			t.Run("invitation_requires_valid_invitation", func(t *testing.T) {
				mode := openapi.RegistrationModeInvitation
				tests.AssertRequest(
					cl.AdminSettingsUpdateWithResponse(root, openapi.AdminSettingsUpdateJSONRequestBody{
						RegistrationMode: &mode,
					}, adminSession),
				)(t, http.StatusOK)

				missingInvite, err := cl.AuthPasswordSignupWithResponse(root, nil, openapi.AuthPair{
					Identifier: "missing-invite-" + xid.New().String(),
					Token:      "password",
				})
				tests.Status(t, err, missingInvite, http.StatusForbidden)

				unknownInvite := openapi.Identifier(xid.New().String())
				unknownInviteResponse, err := cl.AuthPasswordSignupWithResponse(root, &openapi.AuthPasswordSignupParams{
					InvitationId: &unknownInvite,
				}, openapi.AuthPair{
					Identifier: "unknown-invite-" + xid.New().String(),
					Token:      "password",
				})
				tests.Status(t, err, unknownInviteResponse, http.StatusForbidden)

				message := "Join me!"
				deletedInvite := tests.AssertRequest(
					cl.InvitationCreateWithResponse(root, openapi.InvitationInitialProps{
						Message: &message,
					}, adminSession),
				)(t, http.StatusOK)
				tests.AssertRequest(
					cl.InvitationDeleteWithResponse(root, deletedInvite.JSON200.Id, adminSession),
				)(t, http.StatusOK)

				deletedInviteResponse, err := cl.AuthPasswordSignupWithResponse(root, &openapi.AuthPasswordSignupParams{
					InvitationId: &deletedInvite.JSON200.Id,
				}, openapi.AuthPair{
					Identifier: "deleted-invite-" + xid.New().String(),
					Token:      "password",
				})
				tests.Status(t, err, deletedInviteResponse, http.StatusForbidden)

				invite := tests.AssertRequest(
					cl.InvitationCreateWithResponse(root, openapi.InvitationInitialProps{
						Message: &message,
					}, adminSession),
				)(t, http.StatusOK)

				handle := "invited-" + xid.New().String()
				signup := tests.AssertRequest(
					cl.AuthPasswordSignupWithResponse(root, &openapi.AuthPasswordSignupParams{
						InvitationId: &invite.JSON200.Id,
					}, openapi.AuthPair{
						Identifier: handle,
						Token:      "password",
					}),
				)(t, http.StatusOK)
				require.NotNil(t, signup.JSON200)
				assert.NotEmpty(t, signup.HTTPResponse.Header.Get("Set-Cookie"))

				gotAccount := tests.AssertRequest(
					cl.AccountGetWithResponse(root, sh.WithSession(e2e.WithAccountID(root, account.AccountID(openapi.GetAccountID(signup.JSON200.Id))))),
				)(t, http.StatusOK)
				require.NotNil(t, gotAccount.JSON200.InvitedBy)
				assert.Equal(t, admin.ID.String(), gotAccount.JSON200.InvitedBy.Id)
			})

			t.Run("disabled_rejects_public_and_invited_registration", func(t *testing.T) {
				mode := openapi.RegistrationModeDisabled
				tests.AssertRequest(
					cl.AdminSettingsUpdateWithResponse(root, openapi.AdminSettingsUpdateJSONRequestBody{
						RegistrationMode: &mode,
					}, adminSession),
				)(t, http.StatusOK)

				publicSignup, err := cl.AuthPasswordSignupWithResponse(root, nil, openapi.AuthPair{
					Identifier: "disabled-public-" + xid.New().String(),
					Token:      "password",
				})
				tests.Status(t, err, publicSignup, http.StatusForbidden)

				message := "Join me anyway?"
				invite := tests.AssertRequest(
					cl.InvitationCreateWithResponse(root, openapi.InvitationInitialProps{
						Message: &message,
					}, adminSession),
				)(t, http.StatusOK)

				invitedSignup, err := cl.AuthPasswordSignupWithResponse(root, &openapi.AuthPasswordSignupParams{
					InvitationId: &invite.JSON200.Id,
				}, openapi.AuthPair{
					Identifier: "disabled-invited-" + xid.New().String(),
					Token:      "password",
				})
				tests.Status(t, err, invitedSignup, http.StatusForbidden)
			})
		}))
	}))
}
