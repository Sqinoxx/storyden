package category_test

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
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

// TestCategoryCacheInvalidatedByChildCreation guards against a stale ETag on
// a parent category page after a subcategory is created underneath it: a
// client holding the parent's ETag from before the child existed must not
// get served a 304 that omits the new child.
func TestCategoryCacheInvalidatedByChildCreation(t *testing.T) {
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

			adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			adminSession := sh.WithSession(adminCtx)

			parent := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
				Colour:      "#abc123",
				Description: "parent cache test",
				Name:        "Parent " + uuid.NewString(),
			}, adminSession))(t, http.StatusOK)

			parentGet1 := tests.AssertRequest(cl.CategoryGetWithResponse(root, parent.JSON200.Slug, adminSession))(t, http.StatusOK)
			a.Len(parentGet1.JSON200.Children, 0, "parent should have no children yet")

			etag1 := `W/"stale"`

			child := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
				Colour:      "#def456",
				Description: "child cache test",
				Name:        "Child " + uuid.NewString(),
				Parent:      &parent.JSON200.Id,
			}, adminSession))(t, http.StatusOK)

			parentGet2 := tests.AssertRequest(cl.CategoryGetWithResponse(root, parent.JSON200.Slug, adminSession, func(ctx context.Context, req *http.Request) error {
				req.Header.Set("If-None-Match", etag1)
				return nil
			}))(t, http.StatusOK)
			r.NotNil(parentGet2.JSON200, "should return 200 with body after cache invalidation from child creation")
			r.Len(parentGet2.JSON200.Children, 1, "parent should now have the new child in its response")
			a.Equal(child.JSON200.Id, parentGet2.JSON200.Children[0].Id)
		}))
	}))
}

func TestCategoryCacheInvalidatedByChildDeletion(t *testing.T) {
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

			adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			adminSession := sh.WithSession(adminCtx)

			parent := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
				Colour:      "#abc123",
				Description: "parent cache test",
				Name:        "Parent " + uuid.NewString(),
			}, adminSession))(t, http.StatusOK)

			child := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
				Colour:      "#def456",
				Description: "child cache test",
				Name:        "Child " + uuid.NewString(),
				Parent:      &parent.JSON200.Id,
			}, adminSession))(t, http.StatusOK)

			parentGet1 := tests.AssertRequest(cl.CategoryGetWithResponse(root, parent.JSON200.Slug, adminSession))(t, http.StatusOK)
			r.Len(parentGet1.JSON200.Children, 1)

			etag1 := `W/"stale"`

			tests.AssertRequest(cl.CategoryDeleteWithResponse(root, child.JSON200.Slug, openapi.CategoryDeleteJSONRequestBody{
				MoveTo: parent.JSON200.Id,
			}, adminSession))(t, http.StatusOK)

			parentGet2 := tests.AssertRequest(cl.CategoryGetWithResponse(root, parent.JSON200.Slug, adminSession, func(ctx context.Context, req *http.Request) error {
				req.Header.Set("If-None-Match", etag1)
				return nil
			}))(t, http.StatusOK)
			r.NotNil(parentGet2.JSON200, "should return 200 with body after cache invalidation from child deletion")
			r.Len(parentGet2.JSON200.Children, 0, "deleted child must not be served from a stale cache")
		}))
	}))
}

func TestCategoryGetReflectsNewThreadCount(t *testing.T) {
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

			adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			adminSession := sh.WithSession(adminCtx)

			parent := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
				Colour:      "#abc123",
				Description: "parent count test",
				Name:        "Parent " + uuid.NewString(),
			}, adminSession))(t, http.StatusOK)

			child := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
				Colour:      "#def456",
				Description: "child count test",
				Name:        "Child " + uuid.NewString(),
				Parent:      &parent.JSON200.Id,
			}, adminSession))(t, http.StatusOK)

			parentGet1 := tests.AssertRequest(cl.CategoryGetWithResponse(root, parent.JSON200.Slug, adminSession))(t, http.StatusOK)
			r.Len(parentGet1.JSON200.Children, 1)
			r.Equal(0, parentGet1.JSON200.Children[0].PostCount)
			r.Equal("no-store", parentGet1.HTTPResponse.Header.Get("Cache-Control"))

			tests.AssertRequest(cl.ThreadCreateWithResponse(root, openapi.ThreadInitialProps{
				Body:       opt.New("<p>count me</p>").Ptr(),
				Category:   opt.New(child.JSON200.Id).Ptr(),
				Visibility: opt.New(openapi.VisibilityPublished).Ptr(),
				Title:      "Count thread",
			}, adminSession))(t, http.StatusOK)

			parentGet2 := tests.AssertRequest(cl.CategoryGetWithResponse(root, parent.JSON200.Slug, adminSession, func(ctx context.Context, req *http.Request) error {
				req.Header.Set("If-None-Match", parentGet1.HTTPResponse.Header.Get("ETag"))
				req.Header.Set("If-Modified-Since", parentGet1.HTTPResponse.Header.Get("Last-Modified"))
				return nil
			}))(t, http.StatusOK)
			r.Len(parentGet2.JSON200.Children, 1)
			r.Equal(1, parentGet2.JSON200.Children[0].PostCount)
		}))
	}))
}
