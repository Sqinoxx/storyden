package category_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/xid"
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

func TestCategoryRelated(t *testing.T) {
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
			memberCtx, _ := e2e.WithAccount(root, aw, seed.Account_003_Baldur)
			memberSession := sh.WithSession(memberCtx)

			create := func(t *testing.T) *openapi.Category {
				res := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
					Colour:      "#abc123",
					Description: "related testing",
					Name:        "Category " + uuid.NewString(),
				}, adminSession))(t, http.StatusOK)
				require.NotNil(t, res.JSON200)
				return res.JSON200
			}

			relatedIDs := func(c *openapi.Category) []openapi.Identifier {
				if c.Related == nil {
					return nil
				}
				return lo.Map(*c.Related, func(r openapi.CategoryRelated, _ int) openapi.Identifier { return r.Id })
			}

			t.Run("link_is_one_directional", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				ca, cb, cc := create(t), create(t), create(t)

				update := tests.AssertRequest(cl.CategoryUpdateWithResponse(root, ca.Slug, openapi.CategoryUpdateJSONRequestBody{
					Related: &openapi.CategoryRelatedIDs{cb.Id, cc.Id},
				}, adminSession))(t, http.StatusOK)
				r.NotNil(update.JSON200)
				a.ElementsMatch([]openapi.Identifier{cb.Id, cc.Id}, relatedIDs(update.JSON200))

				getA := tests.AssertRequest(cl.CategoryGetWithResponse(root, ca.Slug, memberSession))(t, http.StatusOK)
				r.NotNil(getA.JSON200)
				a.ElementsMatch([]openapi.Identifier{cb.Id, cc.Id}, relatedIDs(getA.JSON200))
				found, ok := lo.Find(*getA.JSON200.Related, func(c openapi.CategoryRelated) bool { return c.Id == cb.Id })
				r.True(ok)
				a.Equal(cb.Slug, found.Slug)
				a.Equal(cb.Name, found.Name)

				getB := tests.AssertRequest(cl.CategoryGetWithResponse(root, cb.Slug, memberSession))(t, http.StatusOK)
				r.NotNil(getB.JSON200)
				a.Empty(relatedIDs(getB.JSON200))
			})

			t.Run("replace_and_clear", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				ca, cb, cc := create(t), create(t), create(t)

				tests.AssertRequest(cl.CategoryUpdateWithResponse(root, ca.Slug, openapi.CategoryUpdateJSONRequestBody{
					Related: &openapi.CategoryRelatedIDs{cb.Id},
				}, adminSession))(t, http.StatusOK)

				replaced := tests.AssertRequest(cl.CategoryUpdateWithResponse(root, ca.Slug, openapi.CategoryUpdateJSONRequestBody{
					Related: &openapi.CategoryRelatedIDs{cc.Id},
				}, adminSession))(t, http.StatusOK)
				r.NotNil(replaced.JSON200)
				a.Equal([]openapi.Identifier{cc.Id}, relatedIDs(replaced.JSON200))

				untouched := tests.AssertRequest(cl.CategoryUpdateWithResponse(root, ca.Slug, openapi.CategoryUpdateJSONRequestBody{
					Description: lo.ToPtr("changed"),
				}, adminSession))(t, http.StatusOK)
				r.NotNil(untouched.JSON200)
				a.Equal([]openapi.Identifier{cc.Id}, relatedIDs(untouched.JSON200))

				cleared := tests.AssertRequest(cl.CategoryUpdateWithResponse(root, ca.Slug, openapi.CategoryUpdateJSONRequestBody{
					Related: &openapi.CategoryRelatedIDs{},
				}, adminSession))(t, http.StatusOK)
				r.NotNil(cleared.JSON200)
				a.Empty(relatedIDs(cleared.JSON200))
			})

			t.Run("self_link_is_ignored", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				ca, cb := create(t), create(t)

				update := tests.AssertRequest(cl.CategoryUpdateWithResponse(root, ca.Slug, openapi.CategoryUpdateJSONRequestBody{
					Related: &openapi.CategoryRelatedIDs{ca.Id, cb.Id, cb.Id},
				}, adminSession))(t, http.StatusOK)
				r.NotNil(update.JSON200)
				a.Equal([]openapi.Identifier{cb.Id}, relatedIDs(update.JSON200))
			})

			t.Run("unknown_category_is_rejected", func(t *testing.T) {
				ca := create(t)

				tests.AssertRequest(cl.CategoryUpdateWithResponse(root, ca.Slug, openapi.CategoryUpdateJSONRequestBody{
					Related: &openapi.CategoryRelatedIDs{openapi.Identifier(xid.New().String())},
				}, adminSession))(t, http.StatusBadRequest)
			})

			t.Run("member_cannot_set_related", func(t *testing.T) {
				ca, cb := create(t), create(t)

				tests.AssertRequest(cl.CategoryUpdateWithResponse(root, ca.Slug, openapi.CategoryUpdateJSONRequestBody{
					Related: &openapi.CategoryRelatedIDs{cb.Id},
				}, memberSession))(t, http.StatusForbidden)
			})

			t.Run("deleted_category_is_unlinked", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				ca, cb, cc := create(t), create(t), create(t)

				tests.AssertRequest(cl.CategoryUpdateWithResponse(root, ca.Slug, openapi.CategoryUpdateJSONRequestBody{
					Related: &openapi.CategoryRelatedIDs{cb.Id, cc.Id},
				}, adminSession))(t, http.StatusOK)

				tests.AssertRequest(cl.CategoryDeleteWithResponse(root, cb.Slug, openapi.CategoryDeleteJSONRequestBody{
					MoveTo: ca.Id,
				}, adminSession))(t, http.StatusOK)

				get := tests.AssertRequest(cl.CategoryGetWithResponse(root, ca.Slug, adminSession))(t, http.StatusOK)
				r.NotNil(get.JSON200)
				a.Equal([]openapi.Identifier{cc.Id}, relatedIDs(get.JSON200))
			})
		}))
	}))
}
