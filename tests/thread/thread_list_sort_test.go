package thread_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/Southclaws/opt"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

func TestThreadListSorting(t *testing.T) {
	t.Parallel()

	integration.Test(t, nil, e2e.Setup(), fx.Invoke(func(
		lc fx.Lifecycle,
		root context.Context,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
	) {
		lc.Append(fx.StartHook(func() {
			r := require.New(t)
			a := assert.New(t)

			ctx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			session := sh.WithSession(ctx)

			cat := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
				Name:        "Sorting " + uuid.NewString(),
				Colour:      "#123456",
				Description: "sort ordering",
			}, session))(t, http.StatusOK)
			r.NotNil(cat.JSON200)

			createThread := func(title string) *openapi.Thread {
				result := tests.AssertRequest(cl.ThreadCreateWithResponse(root, openapi.ThreadInitialProps{
					Title:      title,
					Body:       opt.New("<p>" + title + "</p>").Ptr(),
					Category:   opt.New(cat.JSON200.Id).Ptr(),
					Visibility: opt.New(openapi.VisibilityPublished).Ptr(),
				}, session))(t, http.StatusOK)
				r.NotNil(result.JSON200)
				return result.JSON200
			}

			// Created oldest to newest. The oldest gets the only reply, so its
			// last_reply_at is the most recent of the three and activity
			// ordering must disagree with creation ordering.
			oldest := createThread("Sort oldest " + uuid.NewString())
			middle := createThread("Sort middle " + uuid.NewString())
			newest := createThread("Sort newest " + uuid.NewString())

			tests.AssertRequest(cl.ReplyCreateWithResponse(root, oldest.Slug, openapi.ReplyInitialProps{
				Body: "bumping the oldest thread",
			}, session))(t, http.StatusOK)

			categories := &[]string{cat.JSON200.Slug}

			listOrder := func(sort *openapi.ThreadListParamsSort) []openapi.Identifier {
				list := tests.AssertRequest(cl.ThreadListWithResponse(root, &openapi.ThreadListParams{
					Categories: categories,
					Sort:       sort,
				}, session))(t, http.StatusOK)
				r.NotNil(list.JSON200)

				return lo.Map(list.JSON200.Threads, func(thr openapi.ThreadReference, _ int) openapi.Identifier {
					return thr.Id
				})
			}

			t.Run("defaults_to_newest_first", func(t *testing.T) {
				a.Equal([]openapi.Identifier{newest.Id, middle.Id, oldest.Id}, listOrder(nil))
			})

			t.Run("newest_first", func(t *testing.T) {
				sort := openapi.ThreadListParamsSort("newest")
				a.Equal([]openapi.Identifier{newest.Id, middle.Id, oldest.Id}, listOrder(&sort))
			})

			t.Run("oldest_first", func(t *testing.T) {
				sort := openapi.ThreadListParamsSort("oldest")
				a.Equal([]openapi.Identifier{oldest.Id, middle.Id, newest.Id}, listOrder(&sort))
			})

			t.Run("activity_bumps_replied_threads", func(t *testing.T) {
				sort := openapi.ThreadListParamsSort("activity")
				a.Equal([]openapi.Identifier{oldest.Id, newest.Id, middle.Id}, listOrder(&sort))
			})
		}))
	}))
}
