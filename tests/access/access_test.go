package access_test

import (
	"bytes"
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Southclaws/opt"
	"github.com/google/uuid"
	"github.com/rs/xid"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/access"
	"github.com/Southclaws/storyden/app/resources/access/access_repo"
	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

const secret = "geheimerprotokollinhalt"

func TestAccessPasses(t *testing.T) {
	t.Parallel()

	integration.Test(t, nil, e2e.Setup(), fx.Invoke(func(
		lc fx.Lifecycle,
		root context.Context,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
		repo *access_repo.Repository,
	) {
		lc.Append(fx.StartHook(func() {
			r := require.New(t)

			adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			lockedCtx, locked := e2e.WithAccount(root, aw, seed.Account_003_Baldur)
			paidCtx, paid := e2e.WithAccount(root, aw, seed.Account_004_Loki)
			expiredCtx, expired := e2e.WithAccount(root, aw, seed.Account_006_Freyja)
			admin := sh.WithSession(adminCtx)
			lockedSession := sh.WithSession(lockedCtx)
			paidSession := sh.WithSession(paidCtx)
			expiredSession := sh.WithSession(expiredCtx)

			z1 := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
				Colour: "#fe4efd", Description: "Z1", Name: "Z1 " + uuid.NewString(),
			}, admin))(t, http.StatusOK).JSON200
			anatomy := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
				Colour: "#fe4efd", Description: "Anatomie", Name: "Anatomie " + uuid.NewString(), Parent: &z1.Id,
			}, admin))(t, http.StatusOK).JSON200
			open := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
				Colour: "#fe4efd", Description: "Offen", Name: "Offen " + uuid.NewString(),
			}, admin))(t, http.StatusOK).JSON200

			passName := "Z1-Protokolle"

			t.Run("members_cannot_manage_passes", func(t *testing.T) {
				tests.AssertRequest(cl.AccessPassCreateWithResponse(root, openapi.AccessPassInitialProps{
					Name: "nope",
				}, lockedSession))(t, http.StatusForbidden)
			})

			pass := tests.AssertRequest(cl.AccessPassCreateWithResponse(root, openapi.AccessPassInitialProps{
				Name:        passName,
				PriceLabel:  opt.New("10 €").Ptr(),
				PaymentInfo: opt.New("Bar im Fachschaftsraum").Ptr(),
				CategoryIds: &[]openapi.Identifier{z1.Id},
			}, admin))(t, http.StatusOK).JSON200
			r.Equal(passName, pass.Name)
			r.Equal("10 €", pass.PriceLabel)
			r.Equal([]openapi.Identifier{z1.Id}, pass.CategoryIds)
			r.True(pass.Requestable)
			passID := access.PassID(openapi.ParseID(pass.Id))

			list := tests.AssertRequest(cl.AccessPassListWithResponse(root))(t, http.StatusOK).JSON200
			r.Len(list.Passes, 1)

			pdf := []byte("%PDF-1.4\n%%EOF")
			upload := tests.AssertRequest(cl.AssetUploadWithBodyWithResponse(root,
				&openapi.AssetUploadParams{ContentLength: int64(len(pdf)), Filename: opt.New("protokoll.pdf").Ptr()},
				"application/octet-stream", bytes.NewReader(pdf), admin,
			))(t, http.StatusOK).JSON200

			protocol := tests.AssertRequest(cl.ThreadCreateWithResponse(root, openapi.ThreadInitialProps{
				Title:      "Protokoll Anatomie WS24",
				Body:       opt.New("<p>" + secret + "</p>").Ptr(),
				Category:   &anatomy.Id,
				Visibility: opt.New(openapi.VisibilityPublished).Ptr(),
				AssetIds:   &[]string{upload.Id},
			}, admin))(t, http.StatusOK).JSON200
			r.Len(protocol.Assets, 1)

			openThread := tests.AssertRequest(cl.ThreadCreateWithResponse(root, openapi.ThreadInitialProps{
				Title:      "Offener Thread",
				Body:       opt.New("<p>frei lesbar</p>").Ptr(),
				Category:   &open.Id,
				Visibility: opt.New(openapi.VisibilityPublished).Ptr(),
			}, admin))(t, http.StatusOK).JSON200

			_, err := repo.SetGrant(root, paid.ID, passID, access.GrantStatusActive, access_repo.GrantOptions{})
			r.NoError(err)
			_, err = repo.SetGrant(root, expired.ID, passID, access.GrantStatusActive, access_repo.GrantOptions{
				ExpiresAt: opt.New(time.Now().Add(-time.Hour)),
			})
			r.NoError(err)

			assertLocked := func(t *testing.T, th *openapi.Thread) {
				r := require.New(t)
				r.NotNil(th.Access, "thread should be locked")
				r.True(th.Access.Locked)
				r.Len(th.Access.Passes, 1)
				r.Equal(passName, th.Access.Passes[0].Name)
				r.Equal("10 €", th.Access.Passes[0].PriceLabel)
				r.Equal(protocol.Title, th.Title)
				r.NotContains(th.Body, secret)
				r.Empty(th.Assets)
				r.Empty(th.Replies.Replies)
			}

			assertOpen := func(t *testing.T, th *openapi.Thread) {
				r := require.New(t)
				r.Nil(th.Access, "thread should not be locked")
				r.Contains(th.Body, secret)
				r.Len(th.Assets, 1)
			}

			t.Run("get_locked_without_grant", func(t *testing.T) {
				th := tests.AssertRequest(cl.ThreadGetWithResponse(root, protocol.Slug, nil, lockedSession))(t, http.StatusOK)
				assertLocked(t, th.JSON200)
			})

			t.Run("get_locked_for_guests", func(t *testing.T) {
				th := tests.AssertRequest(cl.ThreadGetWithResponse(root, protocol.Slug, nil))(t, http.StatusOK)
				assertLocked(t, th.JSON200)
			})

			t.Run("get_locked_with_expired_grant", func(t *testing.T) {
				th := tests.AssertRequest(cl.ThreadGetWithResponse(root, protocol.Slug, nil, expiredSession))(t, http.StatusOK)
				assertLocked(t, th.JSON200)
			})

			t.Run("parent_grant_unlocks_subcategory", func(t *testing.T) {
				th := tests.AssertRequest(cl.ThreadGetWithResponse(root, protocol.Slug, nil, paidSession))(t, http.StatusOK)
				assertOpen(t, th.JSON200)
			})

			t.Run("admin_bypasses", func(t *testing.T) {
				th := tests.AssertRequest(cl.ThreadGetWithResponse(root, protocol.Slug, nil, admin))(t, http.StatusOK)
				assertOpen(t, th.JSON200)
			})

			t.Run("open_categories_unaffected", func(t *testing.T) {
				th := tests.AssertRequest(cl.ThreadGetWithResponse(root, openThread.Slug, nil, lockedSession))(t, http.StatusOK)
				r := require.New(t)
				r.Nil(th.JSON200.Access)
				r.Contains(th.JSON200.Body, "frei lesbar")
			})

			t.Run("list_shows_preview", func(t *testing.T) {
				r := require.New(t)
				res := tests.AssertRequest(cl.ThreadListWithResponse(root, &openapi.ThreadListParams{
					Categories: &openapi.CategorySlugListQuery{anatomy.Slug},
				}, lockedSession))(t, http.StatusOK)
				r.Len(res.JSON200.Threads, 1)
				item := res.JSON200.Threads[0]
				r.Equal(protocol.Title, item.Title)
				r.NotNil(item.Access)
				r.True(item.Access.Locked)
				r.Empty(opt.NewPtr(item.Description).Or(""))
				r.NotContains(item.Body, secret)
				r.NotContains(string(res.Body), secret)

				res = tests.AssertRequest(cl.ThreadListWithResponse(root, &openapi.ThreadListParams{
					Categories: &openapi.CategorySlugListQuery{anatomy.Slug},
				}, paidSession))(t, http.StatusOK)
				r.Len(res.JSON200.Threads, 1)
				r.Nil(res.JSON200.Threads[0].Access)
				r.Contains(res.JSON200.Threads[0].Body, secret)
			})

			t.Run("attachments_hidden", func(t *testing.T) {
				r := require.New(t)
				get, err := cl.AssetGetWithResponse(root, upload.Filename, lockedSession)
				r.NoError(err)
				r.Equal(http.StatusNotFound, get.StatusCode())

				get, err = cl.AssetGetWithResponse(root, upload.Filename, paidSession)
				r.NoError(err)
				r.Equal(http.StatusOK, get.StatusCode())
			})

			t.Run("search_shows_title_only", func(t *testing.T) {
				r := require.New(t)
				kinds := []openapi.DatagraphItemKind{openapi.DatagraphItemKindThread}
				res := tests.AssertRequest(cl.DatagraphSearchWithResponse(root, &openapi.DatagraphSearchParams{
					Q:    opt.New("Anatomie").Ptr(),
					Kind: &kinds,
				}, lockedSession))(t, http.StatusOK)
				r.NotContains(string(res.Body), secret)

				found := false
				for _, it := range res.JSON200.Items {
					th, err := it.AsDatagraphItemThread()
					if err != nil || th.Ref.Id != protocol.Id {
						continue
					}
					found = true
					r.NotNil(th.Ref.Access)
				}
				r.True(found, "locked thread should still be findable by title")
			})

			t.Run("replies_and_reactions_blocked", func(t *testing.T) {
				tests.AssertRequest(cl.ReplyCreateWithResponse(root, protocol.Slug, openapi.ReplyInitialProps{
					Body: "<p>darf ich?</p>",
				}, lockedSession))(t, http.StatusForbidden)

				tests.AssertRequest(cl.PostReactAddWithResponse(root, protocol.Id, openapi.PostReactAddJSONRequestBody{
					Emoji: "👍",
				}, lockedSession))(t, http.StatusForbidden)

				tests.AssertRequest(cl.ReplyCreateWithResponse(root, protocol.Slug, openapi.ReplyInitialProps{
					Body: "<p>danke!</p>",
				}, paidSession))(t, http.StatusOK)
			})

			t.Run("posting_requires_access", func(t *testing.T) {
				tests.AssertRequest(cl.ThreadCreateWithResponse(root, openapi.ThreadInitialProps{
					Title:      "Mein Protokoll",
					Body:       opt.New("<p>x</p>").Ptr(),
					Category:   &anatomy.Id,
					Visibility: opt.New(openapi.VisibilityPublished).Ptr(),
				}, lockedSession))(t, http.StatusForbidden)
			})

			t.Run("author_keeps_own_thread_after_revocation", func(t *testing.T) {
				r := require.New(t)
				own := tests.AssertRequest(cl.ThreadCreateWithResponse(root, openapi.ThreadInitialProps{
					Title:      "Mitschrift",
					Body:       opt.New("<p>" + secret + "</p>").Ptr(),
					Category:   &anatomy.Id,
					Visibility: opt.New(openapi.VisibilityPublished).Ptr(),
				}, paidSession))(t, http.StatusOK).JSON200

				_, err := repo.SetGrant(root, paid.ID, passID, access.GrantStatusRevoked, access_repo.GrantOptions{})
				r.NoError(err)

				th := tests.AssertRequest(cl.ThreadGetWithResponse(root, own.Slug, nil, paidSession))(t, http.StatusOK)
				r.Nil(th.JSON200.Access)
				r.Contains(th.JSON200.Body, secret)

				other := tests.AssertRequest(cl.ThreadGetWithResponse(root, protocol.Slug, nil, paidSession))(t, http.StatusOK)
				r.NotNil(other.JSON200.Access)
			})

			t.Run("grant_takes_effect_despite_cached_etag", func(t *testing.T) {
				r := require.New(t)
				before := tests.AssertRequest(cl.ThreadGetWithResponse(root, protocol.Slug, nil, lockedSession))(t, http.StatusOK)
				assertLocked(t, before.JSON200)
				etag := before.HTTPResponse.Header.Get("ETag")
				r.NotEmpty(etag)

				g, err := repo.SetGrant(root, locked.ID, passID, access.GrantStatusActive, access_repo.GrantOptions{})
				r.NoError(err)
				r.Regexp(`^Z1-[A-Z2-9]{4}$`, g.ReferenceCode)

				after := tests.AssertRequest(cl.ThreadGetWithResponse(root, protocol.Slug, &openapi.ThreadGetParams{}, lockedSession, func(ctx context.Context, req *http.Request) error {
					req.Header.Set("If-None-Match", etag)
					return nil
				}))(t, http.StatusOK)
				assertOpen(t, after.JSON200)
			})

			t.Run("deleting_pass_unlocks", func(t *testing.T) {
				r := require.New(t)
				tests.AssertRequest(cl.AccessPassDeleteWithResponse(root, pass.Id, admin))(t, http.StatusOK)

				th := tests.AssertRequest(cl.ThreadGetWithResponse(root, protocol.Slug, nil, expiredSession))(t, http.StatusOK)
				r.Nil(th.JSON200.Access)

				_, err := repo.GetGrant(root, expired.ID, passID)
				r.Error(err, "grants are removed with their pass")
			})

			t.Run("update_unknown_pass", func(t *testing.T) {
				tests.AssertRequest(cl.AccessPassUpdateWithResponse(root, xid.New().String(), openapi.AccessPassMutableProps{
					Name: opt.New("x").Ptr(),
				}, admin))(t, http.StatusNotFound)
			})
		}))
	}))
}
