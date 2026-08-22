package node_versions_test

import (
	"context"
	"net/http"
	"testing"

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

// A library manager reviewing or applying a proposed change must be able to
// do so even when the underlying page itself hasn't been published or
// submitted for review yet (i.e. it's still in "draft" node visibility, not
// merely a node_version with status "draft"). Regression test for a bug
// where the review/apply endpoints returned 404 for such pages because the
// node visibility filter excluded them for anyone but the page's owner.
func TestNodeVersionManagerCanReviewAndApplyOnUnpublishedNode(t *testing.T) {
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

			authorCtx, _ := e2e.WithAccount(root, aw, seed.Account_007_Freyr)
			authorSession := sh.WithSession(authorCtx)

			draft := openapi.VisibilityDraft
			name := "manager-draft-visibility-" + uuid.NewString()

			node, err := cl.NodeCreateWithResponse(root, openapi.NodeInitialProps{
				Name:       name,
				Visibility: &draft,
			}, authorSession)
			tests.Ok(t, err, node)
			require.Equal(t, openapi.VisibilityDraft, node.JSON200.Visibility)

			t.Run("manager can fetch the unpublished node", func(t *testing.T) {
				t.Parallel()

				get, err := cl.NodeGetWithResponse(root, node.JSON200.Slug, &openapi.NodeGetParams{}, adminSession)
				tests.Ok(t, err, get)
				require.NotNil(t, get.JSON200)
				assert.Equal(t, node.JSON200.Id, get.JSON200.Id)
			})

			t.Run("manager can review and apply a draft proposal on it", func(t *testing.T) {
				t.Parallel()
				a := assert.New(t)
				r := require.New(t)

				updatedName := "Reviewed name " + uuid.NewString()
				version := createDraftVersion(t, root, cl, authorSession, node.JSON200.Slug, updatedName)

				get, err := cl.NodeVersionGetWithResponse(root, node.JSON200.Slug, version.Id, adminSession)
				tests.Ok(t, err, get)
				r.NotNil(get.JSON200)
				a.Equal(version.Id, get.JSON200.Id)
				a.Equal(openapi.NodeVersionStatusDraft, get.JSON200.Status)

				apply, err := cl.NodeVersionUpdateStatusWithResponse(root, node.JSON200.Slug, version.Id, openapi.NodeVersionUpdateStatusJSONRequestBody{
					Status: openapi.NodeVersionStatusApplied,
				}, adminSession)
				tests.Ok(t, err, apply)
				r.NotNil(apply.JSON200)
				a.Equal(openapi.NodeVersionStatusApplied, apply.JSON200.Status)
				a.Equal(updatedName, apply.JSON200.Name)
			})

			t.Run("a non-manager other member still cannot see it", func(t *testing.T) {
				t.Parallel()

				otherCtx, _ := e2e.WithAccount(root, aw, seed.Account_004_Loki)
				otherSession := sh.WithSession(otherCtx)

				get, err := cl.NodeGetWithResponse(root, node.JSON200.Slug, &openapi.NodeGetParams{}, otherSession)
				tests.Status(t, err, get, http.StatusNotFound)
			})
		}))
	}))
}
