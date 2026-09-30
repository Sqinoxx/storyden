package daily_library_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Southclaws/opt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/library"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/services/library/daily_library"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

func TestDailyLibrary(t *testing.T) {
	t.Parallel()

	integration.Test(t, nil, e2e.Setup(), fx.Invoke(func(
		lc fx.Lifecycle,
		root context.Context,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
		gen *daily_library.Generator,
	) {
		lc.Append(fx.StartHook(func() {
			adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			memberCtx, _ := e2e.WithAccount(root, aw, seed.Account_003_Baldur)
			adminSession := sh.WithSession(adminCtx)
			memberSession := sh.WithSession(memberCtx)

			category := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
				Colour:      "#fe4efd",
				Description: "daily library",
				Name:        "Daily library " + uuid.NewString(),
			}, adminSession))(t, http.StatusOK)

			createThread := func(t *testing.T, title string, vis openapi.Visibility) *openapi.Thread {
				t.Helper()
				resp := tests.AssertRequest(cl.ThreadCreateWithResponse(root, openapi.ThreadInitialProps{
					Title:      title,
					Body:       opt.New("<p>" + title + "</p>").Ptr(),
					Category:   opt.New(category.JSON200.Id).Ptr(),
					Visibility: opt.New(vis).Ptr(),
				}, memberSession))(t, http.StatusOK)
				return resp.JSON200
			}

			daySlug := daily_library.DaySlug(gen.Day(time.Now()))

			publishedTitle := "Published " + uuid.NewString()
			draftTitle := "Draft " + uuid.NewString()
			published := createThread(t, publishedTitle, openapi.VisibilityPublished)
			createThread(t, draftTitle, openapi.VisibilityDraft)

			require.NoError(t, gen.Sweep(root, time.Now()))

			t.Run("creates_a_review_page_listing_the_days_threads", func(t *testing.T) {
				a := assert.New(t)
				r := require.New(t)

				day := tests.AssertRequest(cl.NodeGetWithResponse(root, daySlug, nil, adminSession))(t, http.StatusOK)
				a.Equal(openapi.VisibilityReview, day.JSON200.Visibility)
				r.NotNil(day.JSON200.Parent)
				a.Equal(daily_library.ArchiveSlug, day.JSON200.Parent.Slug)

				content := opt.NewPtr(day.JSON200.Content).OrZero()
				a.Contains(content, publishedTitle)
				a.Contains(content, `href="/t/`+published.Slug+`"`)
				a.NotContains(content, draftTitle)

				meta, ok := day.JSON200.Meta[daily_library.MetadataKey].(map[string]any)
				r.True(ok)
				a.GreaterOrEqual(meta["thread_count"], float64(1))

				archive := tests.AssertRequest(cl.NodeGetWithResponse(root, daily_library.ArchiveSlug, nil, adminSession))(t, http.StatusOK)
				a.Equal(openapi.VisibilityReview, archive.JSON200.Visibility)
			})

			t.Run("pages_are_hidden_until_published", func(t *testing.T) {
				resp, err := cl.NodeGetWithResponse(root, daySlug, nil, memberSession)
				require.NoError(t, err)
				assert.NotEqual(t, http.StatusOK, resp.StatusCode())

				resp, err = cl.NodeGetWithResponse(root, daySlug, nil)
				require.NoError(t, err)
				assert.NotEqual(t, http.StatusOK, resp.StatusCode())
			})

			t.Run("refreshes_while_in_review", func(t *testing.T) {
				laterTitle := "Later " + uuid.NewString()
				createThread(t, laterTitle, openapi.VisibilityPublished)

				require.NoError(t, gen.Sweep(root, time.Now()))

				day := tests.AssertRequest(cl.NodeGetWithResponse(root, daySlug, nil, adminSession))(t, http.StatusOK)
				assert.Contains(t, opt.NewPtr(day.JSON200.Content).OrZero(), laterTitle)
			})

			t.Run("published_pages_are_left_alone", func(t *testing.T) {
				a := assert.New(t)

				tests.AssertRequest(cl.NodeUpdateVisibilityWithResponse(root, daySlug, openapi.VisibilityMutationProps{
					Visibility: openapi.VisibilityPublished,
				}, adminSession))(t, http.StatusOK)

				ignoredTitle := "Ignored " + uuid.NewString()
				createThread(t, ignoredTitle, openapi.VisibilityPublished)

				require.NoError(t, gen.Sweep(root, time.Now()))

				day := tests.AssertRequest(cl.NodeGetWithResponse(root, daySlug, nil, adminSession))(t, http.StatusOK)
				a.Equal(openapi.VisibilityPublished, day.JSON200.Visibility)
				a.NotContains(opt.NewPtr(day.JSON200.Content).OrZero(), ignoredTitle)
			})

			t.Run("days_without_threads_get_no_page", func(t *testing.T) {
				archive := tests.AssertRequest(cl.NodeGetWithResponse(root, daily_library.ArchiveSlug, nil, adminSession))(t, http.StatusOK)
				archiveID, err := library.NodeIDFromString(archive.JSON200.Id)
				require.NoError(t, err)

				empty := time.Date(1990, 1, 1, 12, 0, 0, 0, time.UTC)
				created, err := gen.GenerateDay(root, account.AccountID(seed.Account_001_Odin.ID), archiveID, empty)
				require.NoError(t, err)
				assert.False(t, created)

				resp, err := cl.NodeGetWithResponse(root, daily_library.DaySlug(gen.Day(empty)), nil, adminSession)
				require.NoError(t, err)
				assert.Equal(t, http.StatusNotFound, resp.StatusCode())
			})
		}))
	}))
}
