package access_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Southclaws/opt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/services/access/access_expiry"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

func findPass(passes []openapi.AccessPass, id openapi.Identifier) *openapi.AccessPass {
	for i := range passes {
		if passes[i].Id == id {
			return &passes[i]
		}
	}
	return nil
}

func TestAccessRequestFlow(t *testing.T) {
	t.Parallel()

	integration.Test(t, nil, e2e.Setup(), fx.Invoke(func(
		lc fx.Lifecycle,
		root context.Context,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
		expiry *access_expiry.Expiry,
	) {
		lc.Append(fx.StartHook(func() {
			adminCtx, admin := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			memberCtx, member := e2e.WithAccount(root, aw, seed.Account_003_Baldur)
			otherCtx, other := e2e.WithAccount(root, aw, seed.Account_004_Loki)
			adminSession := sh.WithSession(adminCtx)
			memberSession := sh.WithSession(memberCtx)
			otherSession := sh.WithSession(otherCtx)

			z1 := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
				Colour: "#fe4efd", Description: "Z1", Name: "Z1 " + uuid.NewString(),
			}, adminSession))(t, http.StatusOK).JSON200

			protocol := tests.AssertRequest(cl.ThreadCreateWithResponse(root, openapi.ThreadInitialProps{
				Title:      "Protokoll",
				Body:       opt.New("<p>" + secret + "</p>").Ptr(),
				Category:   &z1.Id,
				Visibility: opt.New(openapi.VisibilityPublished).Ptr(),
			}, adminSession))(t, http.StatusOK).JSON200

			pass := tests.AssertRequest(cl.AccessPassCreateWithResponse(root, openapi.AccessPassInitialProps{
				Name:        "Z1-Protokolle",
				PriceLabel:  opt.New("10 €").Ptr(),
				CategoryIds: &[]openapi.Identifier{z1.Id},
			}, adminSession))(t, http.StatusOK).JSON200

			closed := tests.AssertRequest(cl.AccessPassCreateWithResponse(root, openapi.AccessPassInitialProps{
				Name:        "Nur persönlich",
				Requestable: opt.New(false).Ptr(),
			}, adminSession))(t, http.StatusOK).JSON200

			var referenceCode string

			t.Run("guests_see_passes_without_membership", func(t *testing.T) {
				r := require.New(t)
				list := tests.AssertRequest(cl.AccessPassListWithResponse(root))(t, http.StatusOK).JSON200
				p := findPass(list.Passes, pass.Id)
				r.NotNil(p)
				r.Nil(p.Membership)
			})

			t.Run("request", func(t *testing.T) {
				r := require.New(t)
				res := tests.AssertRequest(cl.AccessPassRequestWithResponse(root, pass.Id, memberSession))(t, http.StatusOK).JSON200
				r.NotNil(res.Membership)
				r.Equal(openapi.AccessGrantStatusRequested, res.Membership.Status)
				r.Regexp(`^Z1-[A-Z2-9]{4}$`, res.Membership.ReferenceCode)
				r.NotNil(res.Membership.RequestedAt)
				referenceCode = res.Membership.ReferenceCode

				again := tests.AssertRequest(cl.AccessPassRequestWithResponse(root, pass.Id, memberSession))(t, http.StatusOK).JSON200
				r.Equal(referenceCode, again.Membership.ReferenceCode, "requesting twice keeps the same code")

				list := tests.AssertRequest(cl.AccessPassListWithResponse(root, memberSession))(t, http.StatusOK).JSON200
				mine := findPass(list.Passes, pass.Id)
				r.NotNil(mine.Membership)
				r.Equal(openapi.AccessGrantStatusRequested, mine.Membership.Status)

				r.Eventually(func() bool {
					n, err := cl.NotificationListWithResponse(root, &openapi.NotificationListParams{}, adminSession)
					if err != nil || n.JSON200 == nil {
						return false
					}
					for _, it := range n.JSON200.Notifications {
						if it.Event == openapi.AccessRequested && it.Source != nil && it.Source.Id == member.ID.String() {
							return true
						}
					}
					return false
				}, 5*time.Second, 100*time.Millisecond, "managers are told about requests")
			})

			t.Run("request_not_requestable", func(t *testing.T) {
				tests.AssertRequest(cl.AccessPassRequestWithResponse(root, closed.Id, memberSession))(t, http.StatusBadRequest)
			})

			t.Run("withdraw_and_rerequest", func(t *testing.T) {
				r := require.New(t)
				res := tests.AssertRequest(cl.AccessPassRequestWithdrawWithResponse(root, pass.Id, otherSession))(t, http.StatusBadRequest)
				r.Nil(res.JSON200)

				tests.AssertRequest(cl.AccessPassRequestWithResponse(root, pass.Id, otherSession))(t, http.StatusOK)
				withdrawn := tests.AssertRequest(cl.AccessPassRequestWithdrawWithResponse(root, pass.Id, otherSession))(t, http.StatusOK).JSON200
				r.Nil(withdrawn.Membership)

				tests.AssertRequest(cl.AccessPassRequestWithResponse(root, pass.Id, otherSession))(t, http.StatusOK)
			})

			t.Run("members_cannot_manage_grants", func(t *testing.T) {
				tests.AssertRequest(cl.AccessGrantListWithResponse(root, &openapi.AccessGrantListParams{}, memberSession))(t, http.StatusForbidden)
				tests.AssertRequest(cl.AccessGrantSetWithResponse(root, pass.Id, member.Handle, openapi.AccessGrantMutableProps{}, memberSession))(t, http.StatusForbidden)
			})

			t.Run("list_open_requests", func(t *testing.T) {
				r := require.New(t)
				status := openapi.AccessGrantStatusRequested
				list := tests.AssertRequest(cl.AccessGrantListWithResponse(root, &openapi.AccessGrantListParams{Status: &status}, adminSession))(t, http.StatusOK).JSON200
				r.Len(list.Grants, 2)
				handles := []string{list.Grants[0].Account.Handle, list.Grants[1].Account.Handle}
				r.Equal([]string{member.Handle, other.Handle}, handles, "oldest request first")
				for _, g := range list.Grants {
					r.Equal("Z1-Protokolle", g.Pass.Name)
				}
			})

			t.Run("grant_after_payment", func(t *testing.T) {
				r := require.New(t)

				tests.AssertRequest(cl.AccessGrantSetWithResponse(root, pass.Id, member.Handle, openapi.AccessGrantMutableProps{
					ExpiresAt: opt.New(time.Now().Add(-time.Hour)).Ptr(),
				}, adminSession))(t, http.StatusBadRequest)

				expires := time.Now().Add(3 * 24 * time.Hour).Truncate(time.Second)
				g := tests.AssertRequest(cl.AccessGrantSetWithResponse(root, pass.Id, member.Handle, openapi.AccessGrantMutableProps{
					ExpiresAt: &expires,
					Note:      opt.New("10 € bar").Ptr(),
				}, adminSession))(t, http.StatusOK).JSON200
				r.Equal(openapi.AccessGrantStatusActive, g.Status)
				r.Equal(referenceCode, g.ReferenceCode)
				r.Equal("10 € bar", g.Note)
				r.NotNil(g.DecidedBy)
				r.Equal(admin.ID.String(), g.DecidedBy.Id)
				r.NotNil(g.ExpiresAt)
				r.True(expires.Equal(*g.ExpiresAt))

				th := tests.AssertRequest(cl.ThreadGetWithResponse(root, protocol.Slug, nil, memberSession))(t, http.StatusOK).JSON200
				r.Nil(th.Access)
				r.Contains(th.Body, secret)

				list := tests.AssertRequest(cl.AccessPassListWithResponse(root, memberSession))(t, http.StatusOK).JSON200
				mine := findPass(list.Passes, pass.Id)
				r.Equal(openapi.AccessGrantStatusActive, mine.Membership.Status)
				r.NotNil(mine.Membership.ExpiresAt)

				r.Eventually(func() bool {
					n, err := cl.NotificationListWithResponse(root, &openapi.NotificationListParams{}, memberSession)
					if err != nil || n.JSON200 == nil {
						return false
					}
					for _, it := range n.JSON200.Notifications {
						if it.Event == openapi.AccessGranted {
							return true
						}
					}
					return false
				}, 5*time.Second, 100*time.Millisecond, "members are told when they are unlocked")
			})

			t.Run("expiry_reminder", func(t *testing.T) {
				r := require.New(t)
				r.Equal(1, expiry.Sweep(root, time.Now()))
				r.Equal(0, expiry.Sweep(root, time.Now()), "each grant is reminded once")

				r.Eventually(func() bool {
					n, err := cl.NotificationListWithResponse(root, &openapi.NotificationListParams{}, memberSession)
					if err != nil || n.JSON200 == nil {
						return false
					}
					for _, it := range n.JSON200.Notifications {
						if it.Event == openapi.AccessExpiring {
							return true
						}
					}
					return false
				}, 5*time.Second, 100*time.Millisecond)
			})

			t.Run("expired_after_lapse", func(t *testing.T) {
				r := require.New(t)
				later := time.Now().Add(4 * 24 * time.Hour)
				r.Equal(0, expiry.Sweep(root, later))

				expired := openapi.AccessGrantStatusExpired
				list := tests.AssertRequest(cl.AccessGrantListWithResponse(root, &openapi.AccessGrantListParams{Status: &expired}, adminSession))(t, http.StatusOK).JSON200
				r.Empty(list.Grants, "not expired yet in real time")

				active := openapi.AccessGrantStatusActive
				list = tests.AssertRequest(cl.AccessGrantListWithResponse(root, &openapi.AccessGrantListParams{
					Status:        &active,
					AccountHandle: &member.Handle,
				}, adminSession))(t, http.StatusOK).JSON200
				r.Len(list.Grants, 1)
			})

			t.Run("reject", func(t *testing.T) {
				r := require.New(t)
				tests.AssertRequest(cl.AccessGrantRemoveWithResponse(root, pass.Id, member.Handle, &openapi.AccessGrantRemoveParams{
					Action: openapi.Reject,
				}, adminSession))(t, http.StatusBadRequest)

				reason := "Keine Zahlung eingegangen"
				g := tests.AssertRequest(cl.AccessGrantRemoveWithResponse(root, pass.Id, other.Handle, &openapi.AccessGrantRemoveParams{
					Action: openapi.Reject,
					Reason: &reason,
				}, adminSession))(t, http.StatusOK).JSON200
				r.Equal(openapi.AccessGrantStatusRejected, g.Status)
				r.NotNil(g.Reason)
				r.Equal(reason, *g.Reason)

				list := tests.AssertRequest(cl.AccessPassListWithResponse(root, otherSession))(t, http.StatusOK).JSON200
				mine := findPass(list.Passes, pass.Id).Membership
				r.Equal(openapi.AccessGrantStatusRejected, mine.Status)
				r.NotNil(mine.Reason, "the member sees why")
				r.Equal(reason, *mine.Reason)

				again := tests.AssertRequest(cl.AccessPassRequestWithResponse(root, pass.Id, otherSession))(t, http.StatusOK).JSON200
				r.Equal(openapi.AccessGrantStatusRequested, again.Membership.Status, "rejected members may ask again")
				r.Nil(again.Membership.Reason, "a new request clears the old reason")
			})

			t.Run("revoke", func(t *testing.T) {
				r := require.New(t)
				reason := "Doppelt freigeschaltet"
				g := tests.AssertRequest(cl.AccessGrantRemoveWithResponse(root, pass.Id, member.Handle, &openapi.AccessGrantRemoveParams{
					Action: openapi.Revoke,
					Reason: &reason,
				}, adminSession))(t, http.StatusOK).JSON200
				r.Equal(openapi.AccessGrantStatusRevoked, g.Status)
				r.Equal(reason, *g.Reason)

				r.Eventually(func() bool {
					n, err := cl.NotificationListWithResponse(root, &openapi.NotificationListParams{}, memberSession)
					if err != nil || n.JSON200 == nil {
						return false
					}
					for _, it := range n.JSON200.Notifications {
						if it.Event == openapi.AccessRevoked {
							return true
						}
					}
					return false
				}, 5*time.Second, 100*time.Millisecond, "members are told when access is revoked")

				th := tests.AssertRequest(cl.ThreadGetWithResponse(root, protocol.Slug, nil, memberSession))(t, http.StatusOK).JSON200
				r.NotNil(th.Access)

				tests.AssertRequest(cl.AccessGrantRemoveWithResponse(root, pass.Id, member.Handle, &openapi.AccessGrantRemoveParams{
					Action: openapi.Revoke,
				}, adminSession))(t, http.StatusBadRequest)
			})

			t.Run("grant_without_request", func(t *testing.T) {
				r := require.New(t)
				_, fresh := e2e.WithAccount(root, aw, seed.Account_006_Freyja)
				g := tests.AssertRequest(cl.AccessGrantSetWithResponse(root, pass.Id, fresh.Handle, openapi.AccessGrantMutableProps{}, adminSession))(t, http.StatusOK).JSON200
				r.Equal(openapi.AccessGrantStatusActive, g.Status)
				r.Nil(g.ExpiresAt)
				r.Nil(g.RequestedAt)

				tests.AssertRequest(cl.AccessGrantSetWithResponse(root, pass.Id, "nobody-"+uuid.NewString(), openapi.AccessGrantMutableProps{}, adminSession))(t, http.StatusNotFound)
			})

			t.Run("audit_log", func(t *testing.T) {
				r := require.New(t)
				types := openapi.AuditEventTypeFilterQuery{openapi.AccessGrantUpdated}
				res := tests.AssertRequest(cl.AuditEventListWithResponse(root, &openapi.AuditEventListParams{Types: &types}, adminSession))(t, http.StatusOK).JSON200
				r.NotNil(res.Events)

				statuses := map[openapi.AccessGrantStatus]int{}
				for _, ev := range *res.Events {
					d, err := ev.AsAuditEventAccessGrantUpdated()
					r.NoError(err)
					r.Equal("Z1-Protokolle", d.PassName)
					statuses[d.Status]++
					if d.AccountId == openapi.Identifier(member.ID.String()) && d.Status == openapi.AccessGrantStatusActive {
						r.NotNil(d.Note)
						r.Equal("10 € bar", *d.Note)
						r.NotNil(d.ExpiresAt)
					}
				}
				r.Equal(2, statuses[openapi.AccessGrantStatusActive])
				r.Equal(1, statuses[openapi.AccessGrantStatusRejected])
				r.Equal(1, statuses[openapi.AccessGrantStatusRevoked])
			})

			t.Run("pass_audit_log", func(t *testing.T) {
				r := require.New(t)

				temp := tests.AssertRequest(cl.AccessPassCreateWithResponse(root, openapi.AccessPassInitialProps{
					Name: "Vorübergehend",
				}, adminSession))(t, http.StatusOK).JSON200
				tests.AssertRequest(cl.AccessPassUpdateWithResponse(root, temp.Id, openapi.AccessPassMutableProps{
					Name: opt.New("Umbenannt").Ptr(),
				}, adminSession))(t, http.StatusOK)
				tests.AssertRequest(cl.AccessPassDeleteWithResponse(root, temp.Id, adminSession))(t, http.StatusOK)

				types := openapi.AuditEventTypeFilterQuery{openapi.AccessPassUpdated}
				res := tests.AssertRequest(cl.AuditEventListWithResponse(root, &openapi.AuditEventListParams{Types: &types}, adminSession))(t, http.StatusOK).JSON200

				actions := map[openapi.AuditEventAccessPassUpdatedAction]string{}
				for _, ev := range *res.Events {
					d, err := ev.AsAuditEventAccessPassUpdated()
					r.NoError(err)
					if d.PassId == temp.Id {
						actions[d.Action] = d.PassName
					}
				}
				r.Equal(map[openapi.AuditEventAccessPassUpdatedAction]string{
					openapi.Created: "Vorübergehend",
					openapi.Updated: "Umbenannt",
					openapi.Deleted: "Umbenannt",
				}, actions)
			})
		}))
	}))
}
