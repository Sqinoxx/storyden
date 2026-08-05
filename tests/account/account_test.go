package account_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/internal/utils"
)

func TestAccountAdmin(t *testing.T) {
	t.Parallel()

	integration.Test(t, nil, e2e.Setup(), fx.Invoke(func(
		lc fx.Lifecycle,
		root context.Context,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		accountWrite *account_writer.Writer,
	) {
		lc.Append(fx.StartHook(func() {
			r := require.New(t)

			adminHandle := "admin-" + xid.New().String()
			victimHandle := "victim-" + xid.New().String()
			randomHandle := "random-" + xid.New().String()

			// Sign up for a new account with a password

			admin, err := cl.AuthPasswordSignupWithResponse(root, nil, openapi.AuthPair{Identifier: adminHandle, Token: "password"})
			r.NoError(err)
			r.Equal(http.StatusOK, admin.StatusCode())
			adminID := account.AccountID(utils.Must(xid.FromString(admin.JSON200.Id)))
			adminSession := sh.WithSession(e2e.WithAccountID(root, adminID))

			accountWrite.Update(root, adminID, account_writer.SetAdmin(true))

			victim, err := cl.AuthPasswordSignupWithResponse(root, nil, openapi.AuthPair{Identifier: victimHandle, Token: "password"})
			r.NoError(err)
			r.Equal(http.StatusOK, victim.StatusCode())
			victimID := account.AccountID(utils.Must(xid.FromString(victim.JSON200.Id)))
			victimSession := sh.WithSession(e2e.WithAccountID(root, victimID))

			random, err := cl.AuthPasswordSignupWithResponse(root, nil, openapi.AuthPair{Identifier: randomHandle, Token: "password"})
			r.NoError(err)
			r.Equal(http.StatusOK, random.StatusCode())
			randomID := account.AccountID(utils.Must(xid.FromString(random.JSON200.Id)))
			randomSession := sh.WithSession(e2e.WithAccountID(root, randomID))

			// Try to suspend the account without being logged in - fails

			suspend1, err := cl.AdminAccountBanCreateWithResponse(root, victim.JSON200.Id)
			r.NoError(err)
			r.NotNil(suspend1)
			r.Equal(http.StatusUnauthorized, suspend1.StatusCode())

			// Try to suspend the account as a non-admin - fails

			suspend2, err := cl.AdminAccountBanCreateWithResponse(root, victim.JSON200.Id, randomSession)
			r.NoError(err)
			r.NotNil(suspend2)
			r.Equal(http.StatusForbidden, suspend2.StatusCode())

			// Try to suspend the account as an admin - succeeds

			suspend3, err := cl.AdminAccountBanCreateWithResponse(root, victim.JSON200.Id, adminSession)
			r.NoError(err)
			r.NotNil(suspend3)
			r.Equal(http.StatusOK, suspend3.StatusCode())

			victimsigni1, err := cl.AuthPasswordSigninWithResponse(root, openapi.AuthPair{
				Identifier: victimHandle,
				Token:      "password",
			}, victimSession)
			r.NoError(err)
			r.NotNil(victimsigni1)
			r.Equal(http.StatusForbidden, victimsigni1.StatusCode())

			// Try to reinstate the account without being logged in - fails

			reinstate1, err := cl.AdminAccountBanRemoveWithResponse(root, victim.JSON200.Id)
			r.NoError(err)
			r.NotNil(reinstate1)
			r.Equal(http.StatusUnauthorized, reinstate1.StatusCode())

			// Try to reinstate the account as a non-admin - fails

			reinstate2, err := cl.AdminAccountBanRemoveWithResponse(root, victim.JSON200.Id, randomSession)
			r.NoError(err)
			r.NotNil(reinstate2)
			r.Equal(http.StatusForbidden, reinstate2.StatusCode())

			// Try to reinstate the account as an admin - succeeds

			reinstate3, err := cl.AdminAccountBanRemoveWithResponse(root, victim.JSON200.Id, adminSession)
			r.NoError(err)
			r.NotNil(reinstate3)
			r.Equal(http.StatusOK, reinstate3.StatusCode())

			victimsignin2, err := cl.AuthPasswordSigninWithResponse(root, openapi.AuthPair{
				Identifier: victimHandle,
				Token:      "password",
			}, victimSession)
			r.NoError(err)
			r.NotNil(victimsignin2)
			r.Equal(http.StatusOK, victimsignin2.StatusCode())
		}))
	}))
}

func TestAccountDelete(t *testing.T) {
	t.Parallel()

	integration.Test(t, nil, e2e.Setup(), fx.Invoke(func(
		lc fx.Lifecycle,
		root context.Context,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		accountWrite *account_writer.Writer,
	) {
		lc.Append(fx.StartHook(func() {
			r := require.New(t)

			adminHandle := "ad-" + xid.New().String()
			targetHandle := "tg-" + xid.New().String()
			userHandle := "usr-" + xid.New().String()

			adminRes, err := cl.AuthPasswordSignupWithResponse(root, nil, openapi.AuthPair{Identifier: adminHandle, Token: "password"})
			r.NoError(err)
			r.Equal(http.StatusOK, adminRes.StatusCode())
			adminID := account.AccountID(utils.Must(xid.FromString(adminRes.JSON200.Id)))
			adminSession := sh.WithSession(e2e.WithAccountID(root, adminID))
			accountWrite.Update(root, adminID, account_writer.SetAdmin(true))

			targetRes, err := cl.AuthPasswordSignupWithResponse(root, nil, openapi.AuthPair{Identifier: targetHandle, Token: "password"})
			r.NoError(err)
			r.Equal(http.StatusOK, targetRes.StatusCode())
			targetID := account.AccountID(utils.Must(xid.FromString(targetRes.JSON200.Id)))
			_ = targetID

			userRes, err := cl.AuthPasswordSignupWithResponse(root, nil, openapi.AuthPair{Identifier: userHandle, Token: "password"})
			r.NoError(err)
			r.Equal(http.StatusOK, userRes.StatusCode())
			userID := account.AccountID(utils.Must(xid.FromString(userRes.JSON200.Id)))
			userSession := sh.WithSession(e2e.WithAccountID(root, userID))

			// 1. Try deleting target account while active (not suspended) - fails (400 Bad Request)
			delActive, err := cl.AdminAccountDeleteWithResponse(root, targetHandle, adminSession)
			r.NoError(err)
			r.Equal(http.StatusBadRequest, delActive.StatusCode())

			// 2. Suspend target account
			suspendRes, err := cl.AdminAccountBanCreateWithResponse(root, targetHandle, adminSession)
			r.NoError(err)
			r.Equal(http.StatusOK, suspendRes.StatusCode())

			// 3. Try deleting suspended account without auth - fails (401)
			delUnauth, err := cl.AdminAccountDeleteWithResponse(root, targetHandle)
			r.NoError(err)
			r.Equal(http.StatusUnauthorized, delUnauth.StatusCode())

			// 4. Try deleting suspended account as normal user - fails (403)
			delForbidden, err := cl.AdminAccountDeleteWithResponse(root, targetHandle, userSession)
			r.NoError(err)
			r.Equal(http.StatusForbidden, delForbidden.StatusCode())

			// 5. Try deleting self as admin - fails (400)
			delSelf, err := cl.AdminAccountDeleteWithResponse(root, adminHandle, adminSession)
			r.NoError(err)
			r.Equal(http.StatusBadRequest, delSelf.StatusCode())

			// 6. Delete suspended account as admin - succeeds (204)
			delSuccess, err := cl.AdminAccountDeleteWithResponse(root, targetHandle, adminSession)
			r.NoError(err)
			r.Equal(http.StatusNoContent, delSuccess.StatusCode())

			// 7. Verify target profile can no longer be retrieved (404)
			getProfile, err := cl.ProfileGetWithResponse(root, targetHandle)
			r.NoError(err)
			r.Equal(http.StatusNotFound, getProfile.StatusCode())
		}))
	}))
}

