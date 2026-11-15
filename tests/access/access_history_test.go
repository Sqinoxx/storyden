package access_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Southclaws/opt"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

func TestAccessGrantHistory(t *testing.T) {
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
			memberCtx, member := e2e.WithAccount(root, aw, seed.Account_003_Baldur)
			_, other := e2e.WithAccount(root, aw, seed.Account_004_Loki)
			adminSession := sh.WithSession(adminCtx)
			memberSession := sh.WithSession(memberCtx)

			pass := tests.AssertRequest(cl.AccessPassCreateWithResponse(root, openapi.AccessPassInitialProps{
				Name: "Historie A",
			}, adminSession))(t, http.StatusOK).JSON200

			second := tests.AssertRequest(cl.AccessPassCreateWithResponse(root, openapi.AccessPassInitialProps{
				Name: "Historie B",
			}, adminSession))(t, http.StatusOK).JSON200

			firstExpiry := time.Now().Add(30 * 24 * time.Hour).Truncate(time.Second)
			secondExpiry := firstExpiry.Add(180 * 24 * time.Hour)

			tests.AssertRequest(cl.AccessGrantSetWithResponse(root, pass.Id, member.Handle, openapi.AccessGrantMutableProps{
				ExpiresAt: &firstExpiry,
				Note:      opt.New("10 € bar").Ptr(),
			}, adminSession))(t, http.StatusOK)

			tests.AssertRequest(cl.AccessGrantSetWithResponse(root, pass.Id, member.Handle, openapi.AccessGrantMutableProps{
				ExpiresAt: &secondExpiry,
			}, adminSession))(t, http.StatusOK)

			reason := "Doppelt bezahlt"
			tests.AssertRequest(cl.AccessGrantRemoveWithResponse(root, pass.Id, member.Handle, &openapi.AccessGrantRemoveParams{
				Action: openapi.Revoke,
				Reason: &reason,
			}, adminSession))(t, http.StatusOK)

			tests.AssertRequest(cl.AccessGrantSetWithResponse(root, second.Id, member.Handle, openapi.AccessGrantMutableProps{}, adminSession))(t, http.StatusOK)
			tests.AssertRequest(cl.AccessGrantSetWithResponse(root, pass.Id, other.Handle, openapi.AccessGrantMutableProps{}, adminSession))(t, http.StatusOK)

			t.Run("lists_decisions_for_member_and_pass_newest_first", func(t *testing.T) {
				r := require.New(t)

				res := tests.AssertRequest(cl.AccessGrantHistoryWithResponse(root, &openapi.AccessGrantHistoryParams{
					AccountHandle: member.Handle,
					PassId:        &pass.Id,
				}, adminSession))(t, http.StatusOK).JSON200

				r.Len(res.Entries, 3)

				revoked := res.Entries[0]
				r.Equal(openapi.AccessGrantStatusRevoked, revoked.Status)
				r.Equal(pass.Id, revoked.PassId)
				r.Equal("Historie A", revoked.PassName)
				r.NotNil(revoked.Reason)
				r.Equal(reason, *revoked.Reason)
				r.NotNil(revoked.DecidedBy)
				r.Equal(admin.Handle, revoked.DecidedBy.Handle)

				extended := res.Entries[1]
				r.Equal(openapi.AccessGrantStatusActive, extended.Status)
				r.NotNil(extended.ExpiresAt)
				r.True(extended.ExpiresAt.Equal(secondExpiry))

				granted := res.Entries[2]
				r.Equal(openapi.AccessGrantStatusActive, granted.Status)
				r.NotNil(granted.ExpiresAt)
				r.True(granted.ExpiresAt.Equal(firstExpiry))
				r.NotNil(granted.Note)
				r.Equal("10 € bar", *granted.Note)
			})

			t.Run("without_pass_filter_includes_all_passes_of_member_only", func(t *testing.T) {
				r := require.New(t)

				res := tests.AssertRequest(cl.AccessGrantHistoryWithResponse(root, &openapi.AccessGrantHistoryParams{
					AccountHandle: member.Handle,
				}, adminSession))(t, http.StatusOK).JSON200

				r.Len(res.Entries, 4)
				r.Equal(second.Id, res.Entries[0].PassId)
			})

			t.Run("unknown_member", func(t *testing.T) {
				tests.AssertRequest(cl.AccessGrantHistoryWithResponse(root, &openapi.AccessGrantHistoryParams{
					AccountHandle: "does-not-exist-anywhere",
				}, adminSession))(t, http.StatusNotFound)
			})

			t.Run("requires_manage_access", func(t *testing.T) {
				tests.AssertRequest(cl.AccessGrantHistoryWithResponse(root, &openapi.AccessGrantHistoryParams{
					AccountHandle: member.Handle,
				}, memberSession))(t, http.StatusForbidden)

				tests.AssertRequest(cl.AccessGrantHistoryWithResponse(root, &openapi.AccessGrantHistoryParams{
					AccountHandle: member.Handle,
				}))(t, http.StatusUnauthorized)
			})
		}))
	}))
}
