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
	"github.com/Southclaws/storyden/app/resources/mark"
	"github.com/Southclaws/storyden/app/resources/post/category"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

func TestCategoryCRUD(t *testing.T) {
	t.Parallel()

	integration.Test(t, nil, e2e.Setup(), fx.Invoke(func(
		lc fx.Lifecycle,
		root context.Context,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
		category_repo *category.Repository,
	) {
		lc.Append(fx.StartHook(func() {
			adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			adminSession := sh.WithSession(adminCtx)

			t.Run("create_and_list", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				name := "Category " + uuid.NewString()
				create := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
					Colour:      "#abc123",
					Description: "category testing",
					Name:        name,
				}, adminSession))(t, http.StatusOK)
				r.NotNil(create.JSON200)
				a.Equal(name, create.JSON200.Name)
				a.Equal(mark.Slugify(name), create.JSON200.Slug)

				get := tests.AssertRequest(cl.CategoryGetWithResponse(root, create.JSON200.Slug, adminSession))(t, http.StatusOK)
				r.NotNil(get.JSON200)
				a.Equal(create.JSON200.Id, get.JSON200.Id)
				a.Equal(create.JSON200.Slug, get.JSON200.Slug)

				list := tests.AssertRequest(cl.CategoryListWithResponse(root, adminSession))(t, http.StatusOK)
				r.NotNil(list.JSON200)
				found, ok := lo.Find(list.JSON200.Categories, func(c openapi.Category) bool { return c.Id == create.JSON200.Id })
				r.True(ok)
				a.Equal(create.JSON200.Slug, found.Slug)
				a.Equal(create.JSON200.Name, found.Name)
			})

			t.Run("rename_category_updates_slug", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				originalName := "Category " + uuid.NewString()
				newName := "Category " + uuid.NewString()
				newSlug := mark.Slugify(newName)
				create := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
					Colour:      "#def456",
					Description: "category testing",
					Name:        originalName,
				}, adminSession))(t, http.StatusOK)
				r.NotNil(create.JSON200)
				oldSlug := create.JSON200.Slug

				update := tests.AssertRequest(cl.CategoryUpdateWithResponse(root, oldSlug, openapi.CategoryUpdateJSONRequestBody{
					Name: lo.ToPtr(newName),
					Slug: lo.ToPtr(newSlug),
				}, adminSession))(t, http.StatusOK)
				r.NotNil(update.JSON200)
				a.Equal(newSlug, update.JSON200.Slug)
				a.Equal(newName, update.JSON200.Name)

				get := tests.AssertRequest(cl.CategoryGetWithResponse(root, newSlug, adminSession))(t, http.StatusOK)
				r.NotNil(get.JSON200)
				a.Equal(newSlug, get.JSON200.Slug)
				a.Equal(newName, get.JSON200.Name)

				oldGet := tests.AssertRequest(cl.CategoryGetWithResponse(root, oldSlug, adminSession))(t, http.StatusNotFound)
				r.NotNil(oldGet)

				list := tests.AssertRequest(cl.CategoryListWithResponse(root, adminSession))(t, http.StatusOK)
				r.NotNil(list.JSON200)
				found, ok := lo.Find(list.JSON200.Categories, func(c openapi.Category) bool { return c.Id == update.JSON200.Id })
				r.True(ok)
				a.Equal(newSlug, found.Slug)
				a.Equal(newName, found.Name)
			})

			t.Run("create_category_sanitises_invalid_slug", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				prefix := uuid.NewString()
				create := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
					Colour:      "#abc123",
					Description: "category testing",
					Name:        "Category " + prefix,
					Slug:        lo.ToPtr(prefix + "/a-slash"),
				}, adminSession))(t, http.StatusOK)
				r.NotNil(create.JSON200)
				a.Equal(prefix+"-a-slash", create.JSON200.Slug)
			})

			t.Run("update_category_sanitises_invalid_slug", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				name := "Category " + uuid.NewString()
				create := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
					Colour:      "#abc123",
					Description: "category testing",
					Name:        name,
				}, adminSession))(t, http.StatusOK)
				r.NotNil(create.JSON200)

				prefix := uuid.NewString()
				update := tests.AssertRequest(cl.CategoryUpdateWithResponse(root, create.JSON200.Slug, openapi.CategoryUpdateJSONRequestBody{
					Slug: lo.ToPtr(prefix + "/a-slash"),
				}, adminSession))(t, http.StatusOK)
				r.NotNil(update.JSON200)
				a.Equal(prefix+"-a-slash", update.JSON200.Slug)
			})

			t.Run("create_duplicate_names_get_unique_slugs", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				newCategory := func(name string, parent *openapi.Identifier) *openapi.CategoryCreateOK {
					res := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
						Colour:      "#abc123",
						Description: "category testing",
						Name:        name,
						Parent:      parent,
					}, adminSession))(t, http.StatusOK)
					r.NotNil(res.JSON200)
					return res.JSON200
				}

				parentA := newCategory("Semester "+uuid.NewString(), nil)
				parentB := newCategory("Semester "+uuid.NewString(), nil)

				name := "Allgemein " + uuid.NewString()
				base := mark.Slugify(name)

				first := newCategory(name, lo.ToPtr(parentA.Id))
				a.Equal(base, first.Slug)

				second := newCategory(name, lo.ToPtr(parentB.Id))
				a.Equal(parentB.Slug+"-"+base, second.Slug)

				third := newCategory(name, lo.ToPtr(parentB.Id))
				a.Equal(parentB.Slug+"-"+base+"-2", third.Slug)

				rootLevel := newCategory(name, nil)
				a.Equal(base+"-2", rootLevel.Slug)
			})

			t.Run("create_without_slug_or_description", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				name := "Category " + uuid.NewString()
				create := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
					Colour: "#abc123",
					Name:   name,
				}, adminSession))(t, http.StatusOK)
				r.NotNil(create.JSON200)
				a.Equal(mark.Slugify(name), create.JSON200.Slug)
				a.Equal("", create.JSON200.Description)
			})

			t.Run("update_keeps_own_slug_and_dedupes_taken_slug", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				otherName := "Category " + uuid.NewString()
				other := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
					Colour:      "#abc123",
					Description: "category testing",
					Name:        otherName,
				}, adminSession))(t, http.StatusOK)
				r.NotNil(other.JSON200)

				create := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
					Colour:      "#abc123",
					Description: "category testing",
					Name:        "Category " + uuid.NewString(),
				}, adminSession))(t, http.StatusOK)
				r.NotNil(create.JSON200)
				ownSlug := create.JSON200.Slug

				same := tests.AssertRequest(cl.CategoryUpdateWithResponse(root, ownSlug, openapi.CategoryUpdateJSONRequestBody{
					Name:   lo.ToPtr(otherName),
					Slug:   lo.ToPtr(ownSlug),
					Colour: lo.ToPtr("#ff0000"),
				}, adminSession))(t, http.StatusOK)
				r.NotNil(same.JSON200)
				a.Equal(ownSlug, same.JSON200.Slug)
				a.Equal(otherName, same.JSON200.Name)

				taken := tests.AssertRequest(cl.CategoryUpdateWithResponse(root, ownSlug, openapi.CategoryUpdateJSONRequestBody{
					Slug: lo.ToPtr(other.JSON200.Slug),
				}, adminSession))(t, http.StatusOK)
				r.NotNil(taken.JSON200)
				a.Equal(other.JSON200.Slug+"-2", taken.JSON200.Slug)

				otherGet := tests.AssertRequest(cl.CategoryGetWithResponse(root, other.JSON200.Slug, adminSession))(t, http.StatusOK)
				r.NotNil(otherGet.JSON200)
				a.Equal(other.JSON200.Id, otherGet.JSON200.Id)
			})

			t.Run("move_category_to_new_parent", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				parentName := "Category " + uuid.NewString()
				parent := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
					Colour:      "#aaaaaa",
					Description: "category testing",
					Name:        parentName,
				}, adminSession))(t, http.StatusOK)
				r.NotNil(parent.JSON200)

				targetParentName := "Category " + uuid.NewString()
				targetParent := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
					Colour:      "#bbbbbb",
					Description: "category testing",
					Name:        targetParentName,
				}, adminSession))(t, http.StatusOK)
				r.NotNil(targetParent.JSON200)

				childName := "Category " + uuid.NewString()
				child := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
					Colour:      "#cccccc",
					Description: "category testing",
					Name:        childName,
					Parent:      lo.ToPtr(parent.JSON200.Id),
				}, adminSession))(t, http.StatusOK)
				r.NotNil(child.JSON200)

				moveBody := openapi.CategoryUpdatePositionJSONRequestBody{Parent: nullable.NewNullableWithValue(openapi.NullableIdentifier(targetParent.JSON200.Id))}
				move := tests.AssertRequest(cl.CategoryUpdatePositionWithResponse(root, child.JSON200.Slug, moveBody, adminSession))(t, http.StatusOK)
				r.NotNil(move.JSON200)

				updated := tests.AssertRequest(cl.CategoryGetWithResponse(root, child.JSON200.Slug, adminSession))(t, http.StatusOK)
				r.NotNil(updated.JSON200)
				r.NotNil(updated.JSON200.Parent)
				a.Equal(openapi.Identifier(targetParent.JSON200.Id), *updated.JSON200.Parent)
			})

			t.Run("move_category_to_root", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				parentName := "Category " + uuid.NewString()
				parent := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
					Colour:      "#aaaa11",
					Description: "category testing",
					Name:        parentName,
				}, adminSession))(t, http.StatusOK)
				r.NotNil(parent.JSON200)

				childName := "Category " + uuid.NewString()
				child := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
					Colour:      "#bbbb22",
					Description: "category testing",
					Name:        childName,
					Parent:      lo.ToPtr(parent.JSON200.Id),
				}, adminSession))(t, http.StatusOK)
				r.NotNil(child.JSON200)

				moveBody := openapi.CategoryUpdatePositionJSONRequestBody{Parent: nullable.NewNullNullable[openapi.NullableIdentifier]()}
				move := tests.AssertRequest(cl.CategoryUpdatePositionWithResponse(root, child.JSON200.Slug, moveBody, adminSession))(t, http.StatusOK)
				r.NotNil(move.JSON200)

				updated := tests.AssertRequest(cl.CategoryGetWithResponse(root, child.JSON200.Slug, adminSession))(t, http.StatusOK)
				r.NotNil(updated.JSON200)
				a.Nil(updated.JSON200.Parent)
			})

			t.Run("resort_categories", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				parentName := "Category " + uuid.NewString()
				parent := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
					Colour:      "#909090",
					Description: "category testing",
					Name:        parentName,
				}, adminSession))(t, http.StatusOK)
				r.NotNil(parent.JSON200)

				firstName := "Category " + uuid.NewString()
				first := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
					Colour:      "#101010",
					Description: "category testing",
					Name:        firstName,
					Parent:      lo.ToPtr(parent.JSON200.Id),
				}, adminSession))(t, http.StatusOK)
				r.NotNil(first.JSON200)

				secondName := "Category " + uuid.NewString()
				second := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
					Colour:      "#202020",
					Description: "category testing",
					Name:        secondName,
					Parent:      lo.ToPtr(parent.JSON200.Id),
				}, adminSession))(t, http.StatusOK)
				r.NotNil(second.JSON200)

				thirdName := "Category " + uuid.NewString()
				third := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
					Colour:      "#303030",
					Description: "category testing",
					Name:        thirdName,
					Parent:      lo.ToPtr(parent.JSON200.Id),
				}, adminSession))(t, http.StatusOK)
				r.NotNil(third.JSON200)

				moveBody := openapi.CategoryUpdatePositionJSONRequestBody{Before: lo.ToPtr(openapi.Identifier(first.JSON200.Id))}
				move := tests.AssertRequest(cl.CategoryUpdatePositionWithResponse(root, third.JSON200.Slug, moveBody, adminSession))(t, http.StatusOK)
				r.NotNil(move.JSON200)

				firstGet := tests.AssertRequest(cl.CategoryGetWithResponse(root, first.JSON200.Slug, adminSession))(t, http.StatusOK)
				r.NotNil(firstGet.JSON200)
				r.NotNil(firstGet.JSON200.Parent)
				a.Equal(openapi.Identifier(parent.JSON200.Id), *firstGet.JSON200.Parent)

				secondGet := tests.AssertRequest(cl.CategoryGetWithResponse(root, second.JSON200.Slug, adminSession))(t, http.StatusOK)
				r.NotNil(secondGet.JSON200)
				r.NotNil(secondGet.JSON200.Parent)
				a.Equal(openapi.Identifier(parent.JSON200.Id), *secondGet.JSON200.Parent)

				thirdGet := tests.AssertRequest(cl.CategoryGetWithResponse(root, third.JSON200.Slug, adminSession))(t, http.StatusOK)
				r.NotNil(thirdGet.JSON200)
				r.NotNil(thirdGet.JSON200.Parent)
				a.Equal(openapi.Identifier(parent.JSON200.Id), *thirdGet.JSON200.Parent)

				a.Less(thirdGet.JSON200.Sort, firstGet.JSON200.Sort)
				a.Less(firstGet.JSON200.Sort, secondGet.JSON200.Sort)
			})

			t.Run("delete_category", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				originName := "Category " + uuid.NewString()
				targetName := "Category " + uuid.NewString()
				origin := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
					Colour:      "#123123",
					Description: "category testing",
					Name:        originName,
				}, adminSession))(t, http.StatusOK)
				r.NotNil(origin.JSON200)

				target := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
					Colour:      "#321321",
					Description: "category testing",
					Name:        targetName,
				}, adminSession))(t, http.StatusOK)
				r.NotNil(target.JSON200)

				deleted := tests.AssertRequest(cl.CategoryDeleteWithResponse(root, origin.JSON200.Slug, openapi.CategoryDeleteJSONRequestBody{
					MoveTo: target.JSON200.Id,
				}, adminSession))(t, http.StatusOK)
				r.NotNil(deleted.JSON200)
				a.Equal(origin.JSON200.Id, deleted.JSON200.Id)

				getDeleted := tests.AssertRequest(cl.CategoryGetWithResponse(root, origin.JSON200.Slug, adminSession))(t, http.StatusNotFound)
				r.NotNil(getDeleted)

				list := tests.AssertRequest(cl.CategoryListWithResponse(root, adminSession))(t, http.StatusOK)
				r.NotNil(list.JSON200)
				_, ok := lo.Find(list.JSON200.Categories, func(c openapi.Category) bool { return c.Id == origin.JSON200.Id })
				a.False(ok)
			})

			// Regression test: slugs containing "/" are sanitised on create and
			// update (see create_category_sanitises_invalid_slug and
			// update_category_sanitises_invalid_slug above), but rows written
			// before that validation existed can still have one. Deleting such
			// a category previously 404'd because the raw "/" split the
			// request path into two segments before it reached the router.
			t.Run("delete_category_with_legacy_slash_in_slug", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				originName := "Category " + uuid.NewString()
				targetName := "Category " + uuid.NewString()
				origin := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
					Colour:      "#654654",
					Description: "category testing",
					Name:        originName,
				}, adminSession))(t, http.StatusOK)
				r.NotNil(origin.JSON200)

				target := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
					Colour:      "#456456",
					Description: "category testing",
					Name:        targetName,
				}, adminSession))(t, http.StatusOK)
				r.NotNil(target.JSON200)

				// Bypass service-level slug validation to simulate legacy data,
				// the same way the repository layer would have accepted it
				// before validation was added at the service layer.
				legacySlug := origin.JSON200.Slug + "/dt"
				_, err := category_repo.UpdateCategory(root, origin.JSON200.Slug, category.WithSlug(legacySlug))
				r.NoError(err)

				deleted := tests.AssertRequest(cl.CategoryDeleteWithResponse(root, legacySlug, openapi.CategoryDeleteJSONRequestBody{
					MoveTo: target.JSON200.Id,
				}, adminSession))(t, http.StatusOK)
				r.NotNil(deleted.JSON200)
				a.Equal(origin.JSON200.Id, deleted.JSON200.Id)

				list := tests.AssertRequest(cl.CategoryListWithResponse(root, adminSession))(t, http.StatusOK)
				r.NotNil(list.JSON200)
				_, ok := lo.Find(list.JSON200.Categories, func(c openapi.Category) bool { return c.Id == origin.JSON200.Id })
				a.False(ok)
			})
		}))
	}))
}
