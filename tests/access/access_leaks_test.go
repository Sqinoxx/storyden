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

	"github.com/Southclaws/storyden/app/resources/access"
	"github.com/Southclaws/storyden/app/resources/access/access_repo"
	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/account/notification"
	"github.com/Southclaws/storyden/app/resources/account/notification/notify_writer"
	"github.com/Southclaws/storyden/app/resources/datagraph"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/ent"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

// TestAccessSideChannels covers read paths outside thread get and list that
// carry thread content, so a member without the pass cannot reach it there.
func TestAccessSideChannels(t *testing.T) {
	t.Parallel()

	integration.Test(t, nil, e2e.Setup(), fx.Invoke(func(
		lc fx.Lifecycle,
		root context.Context,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
		repo *access_repo.Repository,
		nw *notify_writer.Writer,
		db *ent.Client,
	) {
		lc.Append(fx.StartHook(func() {
			r := require.New(t)

			adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			memberCtx, member := e2e.WithAccount(root, aw, seed.Account_003_Baldur)
			paidCtx, paid := e2e.WithAccount(root, aw, seed.Account_004_Loki)
			admin := sh.WithSession(adminCtx)
			memberSession := sh.WithSession(memberCtx)
			paidSession := sh.WithSession(paidCtx)

			locked := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
				Colour: "#fe4efd", Description: "Z1", Name: "Z1 " + uuid.NewString(),
			}, admin))(t, http.StatusOK).JSON200
			open := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
				Colour: "#fe4efd", Description: "Offen", Name: "Offen " + uuid.NewString(),
			}, admin))(t, http.StatusOK).JSON200

			pass := tests.AssertRequest(cl.AccessPassCreateWithResponse(root, openapi.AccessPassInitialProps{
				Name:        "Z1-Protokolle",
				CategoryIds: &[]openapi.Identifier{locked.Id},
			}, admin))(t, http.StatusOK).JSON200

			_, err := repo.SetGrant(root, paid.ID, access.PassID(openapi.ParseID(pass.Id)), access.GrantStatusActive, access_repo.GrantOptions{})
			r.NoError(err)

			tag := "protokoll" + uuid.NewString()[:8]

			protocol := tests.AssertRequest(cl.ThreadCreateWithResponse(root, openapi.ThreadInitialProps{
				Title:      "Protokoll Anatomie",
				Body:       opt.New("<p>" + secret + "</p>").Ptr(),
				Category:   &locked.Id,
				Visibility: opt.New(openapi.VisibilityPublished).Ptr(),
				Tags:       &openapi.TagNameList{tag},
			}, admin))(t, http.StatusOK).JSON200

			openThread := tests.AssertRequest(cl.ThreadCreateWithResponse(root, openapi.ThreadInitialProps{
				Title:      "Offener Thread",
				Body:       opt.New("<p>frei lesbar</p>").Ptr(),
				Category:   &open.Id,
				Visibility: opt.New(openapi.VisibilityPublished).Ptr(),
			}, admin))(t, http.StatusOK).JSON200

			t.Run("tag_page", func(t *testing.T) {
				r := require.New(t)

				guest := tests.AssertRequest(cl.TagGetWithResponse(root, tag))(t, http.StatusOK)
				r.NotContains(string(guest.Body), secret)
				r.Contains(string(guest.Body), protocol.Title, "the title stays visible as a preview")

				holder := tests.AssertRequest(cl.TagGetWithResponse(root, tag, paidSession))(t, http.StatusOK)
				r.Contains(string(holder.Body), secret)
			})

			t.Run("reports", func(t *testing.T) {
				r := require.New(t)

				created := tests.AssertRequest(cl.ReportCreateWithResponse(root, openapi.ReportInitialProps{
					TargetId:   protocol.Id,
					TargetKind: openapi.DatagraphItemKindPost,
				}, memberSession))(t, http.StatusOK)
				r.NotContains(string(created.Body), secret)

				list := tests.AssertRequest(cl.ReportListWithResponse(root, nil, memberSession))(t, http.StatusOK)
				r.NotContains(string(list.Body), secret)

				staff := tests.AssertRequest(cl.ReportListWithResponse(root, nil, admin))(t, http.StatusOK)
				r.Contains(string(staff.Body), secret)
			})

			t.Run("collection_item_writes", func(t *testing.T) {
				r := require.New(t)

				col := tests.AssertRequest(cl.CollectionCreateWithResponse(root, openapi.CollectionInitialProps{
					Name: "Meine Sammlung",
				}, memberSession))(t, http.StatusOK).JSON200

				added := tests.AssertRequest(cl.CollectionAddPostWithResponse(root, col.Id, protocol.Id, memberSession))(t, http.StatusOK)
				r.NotContains(string(added.Body), secret)

				tests.AssertRequest(cl.CollectionAddPostWithResponse(root, col.Id, openThread.Id, memberSession))(t, http.StatusOK)

				foreign, err := cl.CollectionRemovePostWithResponse(root, col.Id, openThread.Id, paidSession)
				r.NoError(err)
				r.Equal(http.StatusForbidden, foreign.StatusCode(), "only the owner may remove items")

				removed := tests.AssertRequest(cl.CollectionRemovePostWithResponse(root, col.Id, openThread.Id, memberSession))(t, http.StatusOK)
				r.NotContains(string(removed.Body), secret)
			})

			t.Run("reply_to_another_thread", func(t *testing.T) {
				r := require.New(t)

				resp, err := cl.ReplyCreateWithResponse(root, openThread.Slug, openapi.ReplyInitialProps{
					Body:    "<p>siehe dort</p>",
					ReplyTo: &protocol.Id,
				}, memberSession)
				r.NoError(err)
				r.Equal(http.StatusBadRequest, resp.StatusCode())
				r.NotContains(string(resp.Body), secret)

				holder, err := cl.ReplyCreateWithResponse(root, openThread.Slug, openapi.ReplyInitialProps{
					Body:    "<p>zitat</p>",
					ReplyTo: &protocol.Id,
				}, paidSession)
				r.NoError(err)
				r.Equal(http.StatusBadRequest, holder.StatusCode(), "pass holders cannot carry locked text into open threads either")

				own := tests.AssertRequest(cl.ReplyCreateWithResponse(root, openThread.Slug, openapi.ReplyInitialProps{
					Body: "<p>erste antwort</p>",
				}, admin))(t, http.StatusOK).JSON200

				tests.AssertRequest(cl.ReplyCreateWithResponse(root, openThread.Slug, openapi.ReplyInitialProps{
					Body:    "<p>antwort auf antwort</p>",
					ReplyTo: &own.Id,
				}, memberSession))(t, http.StatusOK)
			})

			t.Run("notifications", func(t *testing.T) {
				r := require.New(t)

				_, err := nw.Notification(root, member.ID, notification.EventThreadReply,
					opt.New[datagraph.ItemRef](&datagraph.Ref{ID: openapi.ParseID(protocol.Id), Kind: datagraph.KindPost}),
					opt.NewEmpty[account.AccountID]())
				r.NoError(err)
				_, err = nw.Notification(root, paid.ID, notification.EventThreadReply,
					opt.New[datagraph.ItemRef](&datagraph.Ref{ID: openapi.ParseID(protocol.Id), Kind: datagraph.KindPost}),
					opt.NewEmpty[account.AccountID]())
				r.NoError(err)

				list := tests.AssertRequest(cl.NotificationListWithResponse(root, nil, memberSession))(t, http.StatusOK)
				r.Len(list.JSON200.Notifications, 1, "the notification itself is kept")
				r.NotNil(list.JSON200.Notifications[0].Item)
				r.NotContains(string(list.Body), secret)

				holder := tests.AssertRequest(cl.NotificationListWithResponse(root, nil, paidSession))(t, http.StatusOK)
				r.Contains(string(holder.Body), secret)
			})

			t.Run("event_thread", func(t *testing.T) {
				r := require.New(t)

				start := time.Now().Add(24 * time.Hour)
				evt := tests.AssertRequest(cl.EventCreateWithResponse(root, openapi.EventInitialProps{
					Name:                "Klausurbesprechung",
					Content:             "<p>" + secret + "</p>",
					ParticipationPolicy: openapi.EventParticipationPolicyOpen,
					ThreadCategoryId:    locked.Id,
					TimeRange:           openapi.EventTimeRange{Start: start, End: start.Add(time.Hour)},
					Visibility:          openapi.VisibilityPublished,
				}, admin))(t, http.StatusOK).JSON200

				guest := tests.AssertRequest(cl.EventGetWithResponse(root, evt.Slug))(t, http.StatusOK)
				r.NotContains(string(guest.Body), secret)
				r.NotNil(guest.JSON200.Thread.Access)

				holder := tests.AssertRequest(cl.EventGetWithResponse(root, evt.Slug, paidSession))(t, http.StatusOK)
				r.Contains(string(holder.Body), secret)

				denied, err := cl.EventCreateWithResponse(root, openapi.EventInitialProps{
					Name:                "Nicht erlaubt",
					Content:             "<p>x</p>",
					ParticipationPolicy: openapi.EventParticipationPolicyOpen,
					ThreadCategoryId:    locked.Id,
					TimeRange:           openapi.EventTimeRange{Start: start, End: start.Add(time.Hour)},
					Visibility:          openapi.VisibilityPublished,
				}, memberSession)
				r.NoError(err)
				r.NotEqual(http.StatusOK, denied.StatusCode())
			})

			t.Run("links", func(t *testing.T) {
				r := require.New(t)

				hiddenURL := "https://drive.example.com/altklausuren-" + uuid.NewString()
				sharedURL := "https://example.com/shared-" + uuid.NewString()

				hidden, err := db.Link.Create().
					SetURL(hiddenURL).SetSlug("hidden-" + uuid.NewString()).SetDomain("drive.example.com").
					SetTitle("Altklausuren").SetDescription("").
					AddPostContentReferenceIDs(openapi.ParseID(protocol.Id)).
					Save(root)
				r.NoError(err)

				shared, err := db.Link.Create().
					SetURL(sharedURL).SetSlug("shared-"+uuid.NewString()).SetDomain("example.com").
					SetTitle("Geteilt").SetDescription("").
					AddPostContentReferenceIDs(openapi.ParseID(protocol.Id), openapi.ParseID(openThread.Id)).
					AddPostIDs(openapi.ParseID(protocol.Id)).
					Save(root)
				r.NoError(err)

				list := tests.AssertRequest(cl.LinkListWithResponse(root, nil, memberSession))(t, http.StatusOK)
				r.NotContains(string(list.Body), hiddenURL)
				r.Contains(string(list.Body), sharedURL)

				holderList := tests.AssertRequest(cl.LinkListWithResponse(root, nil, paidSession))(t, http.StatusOK)
				r.Contains(string(holderList.Body), hiddenURL)

				gone, err := cl.LinkGetWithResponse(root, hidden.Slug, memberSession)
				r.NoError(err)
				r.Equal(http.StatusNotFound, gone.StatusCode())

				tests.AssertRequest(cl.LinkGetWithResponse(root, hidden.Slug, paidSession))(t, http.StatusOK)

				visible := tests.AssertRequest(cl.LinkGetWithResponse(root, shared.Slug, memberSession))(t, http.StatusOK)
				for _, p := range visible.JSON200.Posts {
					r.NotEqual(protocol.Id, p.Id, "locked posts are not listed as places a link appears")
				}
			})
		}))
	}))
}
