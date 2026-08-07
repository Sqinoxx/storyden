package account_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/opt"
	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/app/resources/account/account_querier"
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
		accountQuery *account_querier.Querier,
	) {
		lc.Append(fx.StartHook(func() {
			r := require.New(t)

			adminHandle := "ad-" + xid.New().String()
			targetHandle := "tg-" + xid.New().String()
			target2Handle := "tg2-" + xid.New().String()
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
			targetSession := sh.WithSession(e2e.WithAccountID(root, targetID))

			target2Res, err := cl.AuthPasswordSignupWithResponse(root, nil, openapi.AuthPair{Identifier: target2Handle, Token: "password"})
			r.NoError(err)
			r.Equal(http.StatusOK, target2Res.StatusCode())
			target2ID := account.AccountID(utils.Must(xid.FromString(target2Res.JSON200.Id)))
			target2Session := sh.WithSession(e2e.WithAccountID(root, target2ID))

			// Admin creates a category
			catRes, err := cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
				Name: "General " + xid.New().String(),
			}, adminSession)
			r.NoError(err)
			r.Equal(http.StatusOK, catRes.StatusCode())
			catID := catRes.JSON200.Id

			// Target user creates a thread post in category
			bodyText := "<p>This content should remain after account deletion.</p>"
			threadRes, err := cl.ThreadCreateWithResponse(root, openapi.ThreadInitialProps{
				Title: "Preserved Thread",
				Body: &bodyText,
				Category: &catID,
				Visibility: opt.New(openapi.VisibilityPublished).Ptr(),
			}, targetSession)
			r.NoError(err)
			r.Equal(http.StatusOK, threadRes.StatusCode())
			threadSlug := threadRes.JSON200.Slug

			// Second target user also creates a thread, to verify multiple
			// deleted accounts' content shares the same placeholder author.
			body2Text := "<p>This content should also remain after account deletion.</p>"
			thread2Res, err := cl.ThreadCreateWithResponse(root, openapi.ThreadInitialProps{
				Title: "Also Preserved Thread",
				Body: &body2Text,
				Category: &catID,
				Visibility: opt.New(openapi.VisibilityPublished).Ptr(),
			}, target2Session)
			r.NoError(err)
			r.Equal(http.StatusOK, thread2Res.StatusCode())
			thread2Slug := thread2Res.JSON200.Slug

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

			// 7. Verify the thread post still exists and author is "Deleted User"
			getThread, err := cl.ThreadGetWithResponse(root, threadSlug, nil)
			r.NoError(err)
			r.Equal(http.StatusOK, getThread.StatusCode())
			r.Equal("Deleted User", getThread.JSON200.Author.Name)

			// 8. Verify target profile under old handle can no longer be retrieved (404)
			getProfile, err := cl.ProfileGetWithResponse(root, targetHandle)
			r.NoError(err)
			r.Equal(http.StatusNotFound, getProfile.StatusCode())

			// 9. Verify the account row itself is gone, not just anonymised in place.
			_, err = accountQuery.GetByID(root, targetID)
			r.Error(err)

			// 10. Suspend and delete the second account too, to prove the
			// shared placeholder is reused across multiple hard deletions.
			suspend2Res, err := cl.AdminAccountBanCreateWithResponse(root, target2Handle, adminSession)
			r.NoError(err)
			r.Equal(http.StatusOK, suspend2Res.StatusCode())

			delSuccess2, err := cl.AdminAccountDeleteWithResponse(root, target2Handle, adminSession)
			r.NoError(err)
			r.Equal(http.StatusNoContent, delSuccess2.StatusCode())

			getThread2, err := cl.ThreadGetWithResponse(root, thread2Slug, nil)
			r.NoError(err)
			r.Equal(http.StatusOK, getThread2.StatusCode())
			r.Equal("Deleted User", getThread2.JSON200.Author.Name)

			// Both threads' authored-by placeholder must be the exact same account.
			r.Equal(getThread.JSON200.Author.Id, getThread2.JSON200.Author.Id)

			_, err = accountQuery.GetByID(root, target2ID)
			r.Error(err)
		}))
	}))
}

