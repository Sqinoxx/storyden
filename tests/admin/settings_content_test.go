package admin_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/Southclaws/opt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/resources/settings"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

func TestContentSettings(t *testing.T) {
	t.Parallel()

	integration.Test(t, nil, e2e.Setup(), fx.Invoke(func(
		lc fx.Lifecycle,
		root context.Context,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
	) {
		lc.Append(fx.StartHook(func() {
			adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			adminSession := sh.WithSession(adminCtx)

			t.Run("defaults", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				get := tests.AssertRequest(cl.AdminSettingsGetWithResponse(adminCtx, adminSession))(t, http.StatusOK)
				r.NotNil(get.JSON200)
				r.NotNil(get.JSON200.Services)
				r.NotNil(get.JSON200.Services.Content)

				r.NotNil(get.JSON200.Services.Content.RepliesPerPage)
				a.Equal(settings.DefaultRepliesPerPage, *get.JSON200.Services.Content.RepliesPerPage)

				r.NotNil(get.JSON200.Services.Content.ThreadsPerPage)
				a.Equal(settings.DefaultThreadsPerPage, *get.JSON200.Services.Content.ThreadsPerPage)
			})

			t.Run("round_trip", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				update := tests.AssertRequest(cl.AdminSettingsUpdateWithResponse(adminCtx, openapi.AdminSettingsUpdateJSONRequestBody{
					Services: &openapi.AdminSettingsServiceProps{
						Content: &openapi.ContentServiceSettings{
							RepliesPerPage: opt.New(25).Ptr(),
							ThreadsPerPage: opt.New(30).Ptr(),
						},
					},
				}, adminSession))(t, http.StatusOK)
				r.NotNil(update.JSON200)
				r.NotNil(update.JSON200.Services)
				r.NotNil(update.JSON200.Services.Content)
				a.Equal(25, *update.JSON200.Services.Content.RepliesPerPage)
				a.Equal(30, *update.JSON200.Services.Content.ThreadsPerPage)

				get := tests.AssertRequest(cl.AdminSettingsGetWithResponse(adminCtx, adminSession))(t, http.StatusOK)
				r.NotNil(get.JSON200)
				a.Equal(25, *get.JSON200.Services.Content.RepliesPerPage)
				a.Equal(30, *get.JSON200.Services.Content.ThreadsPerPage)
			})

			t.Run("configured_page_size_applies_to_threads", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				tests.AssertRequest(cl.AdminSettingsUpdateWithResponse(adminCtx, openapi.AdminSettingsUpdateJSONRequestBody{
					Services: &openapi.AdminSettingsServiceProps{
						Content: &openapi.ContentServiceSettings{
							RepliesPerPage: opt.New(3).Ptr(),
						},
					},
				}, adminSession))(t, http.StatusOK)

				cat := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
					Name:        "PageSize " + uuid.NewString(),
					Colour:      "#123456",
					Description: "page size",
				}, adminSession))(t, http.StatusOK)

				thr := tests.AssertRequest(cl.ThreadCreateWithResponse(root, openapi.ThreadInitialProps{
					Title:      "Page size thread " + uuid.NewString(),
					Body:       opt.New("<p>body</p>").Ptr(),
					Category:   opt.New(cat.JSON200.Id).Ptr(),
					Visibility: opt.New(openapi.VisibilityPublished).Ptr(),
				}, adminSession))(t, http.StatusOK)

				for range 5 {
					tests.AssertRequest(cl.ReplyCreateWithResponse(root, thr.JSON200.Slug, openapi.ReplyInitialProps{
						Body: "reply",
					}, adminSession))(t, http.StatusOK)
				}

				get := tests.AssertRequest(cl.ThreadGetWithResponse(root, thr.JSON200.Slug, &openapi.ThreadGetParams{}, adminSession))(t, http.StatusOK)
				r.NotNil(get.JSON200)

				a.Equal(3, get.JSON200.Replies.PageSize)
				a.Len(get.JSON200.Replies.Replies, 3)
				a.Equal(2, get.JSON200.Replies.TotalPages)
			})
		}))
	}))
}
