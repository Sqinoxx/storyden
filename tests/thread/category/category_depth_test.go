package category_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/oapi-codegen/nullable"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/post/category"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

func TestCategoryNestingDepth(t *testing.T) {
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

			create := func(t *testing.T, parent *openapi.Identifier) *openapi.Category {
				t.Helper()

				result := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
					Colour:      "#abc123",
					Description: "nesting test",
					Name:        "Category " + uuid.NewString(),
					Parent:      parent,
				}, adminSession))(t, http.StatusOK)
				require.NotNil(t, result.JSON200)

				return (*openapi.Category)(result.JSON200)
			}

			t.Run("deeply_nested_tree_is_returned_in_full", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				// Four levels, one more than the two the API used to hydrate.
				l1 := create(t, nil)
				l2 := create(t, &l1.Id)
				l3 := create(t, &l2.Id)
				l4 := create(t, &l3.Id)

				get := tests.AssertRequest(cl.CategoryGetWithResponse(root, l1.Slug, adminSession))(t, http.StatusOK)
				r.NotNil(get.JSON200)

				r.Len(get.JSON200.Children, 1)
				a.Equal(l2.Id, get.JSON200.Children[0].Id)

				r.Len(get.JSON200.Children[0].Children, 1)
				a.Equal(l3.Id, get.JSON200.Children[0].Children[0].Id)

				r.Len(get.JSON200.Children[0].Children[0].Children, 1)
				a.Equal(l4.Id, get.JSON200.Children[0].Children[0].Children[0].Id)

				a.Empty(get.JSON200.Children[0].Children[0].Children[0].Children)
			})

			t.Run("list_returns_every_level_flat", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				l1 := create(t, nil)
				l2 := create(t, &l1.Id)
				l3 := create(t, &l2.Id)

				list := tests.AssertRequest(cl.CategoryListWithResponse(root, adminSession))(t, http.StatusOK)
				r.NotNil(list.JSON200)

				for _, want := range []*openapi.Category{l1, l2, l3} {
					found, ok := lo.Find(list.JSON200.Categories, func(c openapi.Category) bool { return c.Id == want.Id })
					r.True(ok, "category %s missing from flat list", want.Name)
					a.Equal(want.Slug, found.Slug)
				}

				// The flat list stays flat, the client assembles the tree from
				// the parent pointers.
				deepest, ok := lo.Find(list.JSON200.Categories, func(c openapi.Category) bool { return c.Id == l3.Id })
				r.True(ok)
				r.NotNil(deepest.Parent)
				a.Equal(l2.Id, *deepest.Parent)
			})

			t.Run("rejects_nesting_past_the_depth_limit", func(t *testing.T) {
				current := create(t, nil)
				for range category.MaxCategoryDepth - 1 {
					current = create(t, &current.Id)
				}

				// The chain now occupies every allowed level, one more must fail.
				tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
					Colour:      "#abc123",
					Description: "one level too deep",
					Name:        "Category " + uuid.NewString(),
					Parent:      &current.Id,
				}, adminSession))(t, http.StatusBadRequest)
			})

			t.Run("rejects_moving_a_category_into_its_own_descendant", func(t *testing.T) {
				parent := create(t, nil)
				child := create(t, &parent.Id)

				tests.AssertRequest(cl.CategoryUpdatePositionWithResponse(root, parent.Slug, openapi.CategoryUpdatePositionJSONRequestBody{
					Parent: nullable.NewNullableWithValue(openapi.NullableIdentifier(child.Id)),
				}, adminSession))(t, http.StatusBadRequest)
			})
		}))
	}))
}
