package thread_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/Southclaws/opt"
	"github.com/google/uuid"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/services/link/link_backfill"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/ent"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

func TestThreadLinkFromEmbeddedCard(t *testing.T) {
	t.Parallel()

	integration.Test(t, nil, e2e.Setup(), fx.Invoke(func(
		lc fx.Lifecycle,
		root context.Context,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
		db *ent.Client,
		backfill *link_backfill.Backfill,
	) {
		lc.Append(fx.StartHook(func() {
			r := require.New(t)
			a := assert.New(t)

			ctx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			session := sh.WithSession(ctx)

			seedID := uuid.NewString()
			catRes := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
				Name:        "Link cards " + seedID,
				Colour:      "#123456",
				Description: "link cards",
			}, session))(t, http.StatusOK)
			cat := &catRes.JSON200.Id
			cardURL := "https://link-card-" + seedID + ".invalid/"
			cardBody := `<p>Altklausuren</p><div data-href="` + cardURL + `" data-display="card"></div>`

			listLink := func(id openapi.Identifier) *openapi.LinkReference {
				list := tests.AssertRequest(cl.ThreadListWithResponse(root, &openapi.ThreadListParams{
					Categories: &[]string{catRes.JSON200.Slug},
				}, session))(t, http.StatusOK)
				for _, thr := range list.JSON200.Threads {
					if thr.Id == id {
						return thr.Link
					}
				}
				r.FailNow("thread not in list")
				return nil
			}

			created := tests.AssertRequest(cl.ThreadCreateWithResponse(root, openapi.ThreadInitialProps{
				Title:      "Card " + seedID,
				Body:       opt.New(cardBody).Ptr(),
				Category:   cat,
				Visibility: opt.New(openapi.VisibilityPublished).Ptr(),
			}, session))(t, http.StatusOK)
			r.NotNil(created.JSON200.Link)
			a.Equal(cardURL, created.JSON200.Link.Url)

			link := listLink(created.JSON200.Id)
			r.NotNil(link)
			a.Equal(cardURL, link.Url)

			tests.AssertRequest(cl.ThreadUpdateWithResponse(root, created.JSON200.Slug, openapi.ThreadMutableProps{
				Body: opt.New("<p>No more card</p>").Ptr(),
			}, session))(t, http.StatusOK)
			a.Nil(listLink(created.JSON200.Id))

			tests.AssertRequest(cl.ThreadUpdateWithResponse(root, created.JSON200.Slug, openapi.ThreadMutableProps{
				Body: opt.New(cardBody).Ptr(),
			}, session))(t, http.StatusOK)
			link = listLink(created.JSON200.Id)
			r.NotNil(link)
			a.Equal(cardURL, link.Url)

			plain := tests.AssertRequest(cl.ThreadCreateWithResponse(root, openapi.ThreadInitialProps{
				Title:      "Legacy " + seedID,
				Body:       opt.New("<p>legacy</p>").Ptr(),
				Category:   cat,
				Visibility: opt.New(openapi.VisibilityPublished).Ptr(),
			}, session))(t, http.StatusOK)
			legacyID, err := xid.FromString(plain.JSON200.Id)
			r.NoError(err)
			r.NoError(db.Post.UpdateOneID(legacyID).SetBody(`<body>` + cardBody + `</body>`).Exec(root))
			a.Nil(listLink(plain.JSON200.Id))

			a.GreaterOrEqual(backfill.Run(root), 1)
			link = listLink(plain.JSON200.Id)
			r.NotNil(link)
			a.Equal(cardURL, link.Url)
		}))
	}))
}
