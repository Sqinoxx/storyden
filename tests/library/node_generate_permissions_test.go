package library_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/config"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

// TestNodeGeneratePermissions covers B17: NodeGenerateContent/Tags/Title call
// out to an LLM on the caller's behalf, which costs money and third-party API
// quota, but were reachable by any logged-in member regardless of robot
// permissions. They must require rbac.PermissionUseRobots the same as the
// rest of the robot/LLM surface.
func TestNodeGeneratePermissions(t *testing.T) {
	t.Parallel()

	integration.Test(t, &config.Config{}, e2e.Setup(), fx.Invoke(func(
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

			visibility := openapi.VisibilityPublished
			slug := "generate-perm-test-" + uuid.NewString()
			content := "<p>Some content.</p>"
			node, err := cl.NodeCreateWithResponse(root, openapi.NodeInitialProps{
				Name:       "Generate perm test",
				Slug:       &slug,
				Content:    &content,
				Visibility: &visibility,
			}, adminSession)
			tests.Ok(t, err, node)
			nodeSlug := node.JSON200.Slug

			body := "<p>Some content to summarise.</p>"

			t.Run("NodeGenerateContent", func(t *testing.T) {
				r := require.New(t)

				unauthed, err := cl.NodeGenerateContentWithResponse(root, nodeSlug, openapi.NodeGenerateContentJSONRequestBody{Content: body})
				r.NoError(err)
				r.Equal(http.StatusUnauthorized, unauthed.StatusCode())

				forbidden, err := cl.NodeGenerateContentWithResponse(root, nodeSlug, openapi.NodeGenerateContentJSONRequestBody{Content: body}, memberSession)
				r.NoError(err)
				r.Equal(http.StatusForbidden, forbidden.StatusCode(), "a member without USE_ROBOTS must not be able to trigger LLM generation")
			})

			t.Run("NodeGenerateTags", func(t *testing.T) {
				r := require.New(t)

				unauthed, err := cl.NodeGenerateTagsWithResponse(root, nodeSlug, openapi.NodeGenerateTagsJSONRequestBody{Content: body})
				r.NoError(err)
				r.Equal(http.StatusUnauthorized, unauthed.StatusCode())

				forbidden, err := cl.NodeGenerateTagsWithResponse(root, nodeSlug, openapi.NodeGenerateTagsJSONRequestBody{Content: body}, memberSession)
				r.NoError(err)
				r.Equal(http.StatusForbidden, forbidden.StatusCode(), "a member without USE_ROBOTS must not be able to trigger LLM generation")
			})

			t.Run("NodeGenerateTitle", func(t *testing.T) {
				r := require.New(t)

				unauthed, err := cl.NodeGenerateTitleWithResponse(root, nodeSlug, openapi.NodeGenerateTitleJSONRequestBody{Content: body})
				r.NoError(err)
				r.Equal(http.StatusUnauthorized, unauthed.StatusCode())

				forbidden, err := cl.NodeGenerateTitleWithResponse(root, nodeSlug, openapi.NodeGenerateTitleJSONRequestBody{Content: body}, memberSession)
				r.NoError(err)
				r.Equal(http.StatusForbidden, forbidden.StatusCode(), "a member without USE_ROBOTS must not be able to trigger LLM generation")
			})
		}))
	}))
}
