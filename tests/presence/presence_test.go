package presence_test

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

func TestPresenceHeartbeat(t *testing.T) {
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
			user1Ctx, _ := e2e.WithAccount(root, aw, seed.Account_003_Baldur)
			user2Ctx, _ := e2e.WithAccount(root, aw, seed.Account_004_Loki)
			adminSession := sh.WithSession(adminCtx)
			user1Session := sh.WithSession(user1Ctx)
			user2Session := sh.WithSession(user2Ctx)

			catA, err := cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{Colour: "#fe4efd", Description: "presence a", Name: "Presence A " + uuid.NewString()}, adminSession)
			tests.Ok(t, err, catA)
			catB, err := cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{Colour: "#fe4efd", Description: "presence b", Name: "Presence B " + uuid.NewString()}, adminSession)
			tests.Ok(t, err, catB)

			t.Run("unauthenticated", func(t *testing.T) {
				res, err := cl.PresenceHeartbeatWithResponse(root, openapi.PresenceHeartbeatJSONRequestBody{})
				r.NoError(err)
				a.Equal(http.StatusUnauthorized, res.StatusCode())
			})

			t.Run("counts_online_and_category", func(t *testing.T) {
				res, err := cl.PresenceHeartbeatWithResponse(root, openapi.PresenceHeartbeatJSONRequestBody{CategoryId: &catA.JSON200.Id}, user1Session)
				tests.Ok(t, err, res)
				a.Equal(1, res.JSON200.Online)
				r.NotNil(res.JSON200.InCategory)
				a.Equal(0, *res.JSON200.InCategory)

				res, err = cl.PresenceHeartbeatWithResponse(root, openapi.PresenceHeartbeatJSONRequestBody{CategoryId: &catB.JSON200.Id}, adminSession)
				tests.Ok(t, err, res)
				a.Equal(2, res.JSON200.Online)
				a.Equal(0, *res.JSON200.InCategory)

				res, err = cl.PresenceHeartbeatWithResponse(root, openapi.PresenceHeartbeatJSONRequestBody{CategoryId: &catA.JSON200.Id}, user2Session)
				tests.Ok(t, err, res)
				a.Equal(3, res.JSON200.Online)
				a.Equal(1, *res.JSON200.InCategory)

				res, err = cl.PresenceHeartbeatWithResponse(root, openapi.PresenceHeartbeatJSONRequestBody{}, user1Session)
				tests.Ok(t, err, res)
				a.Equal(3, res.JSON200.Online)
				a.Nil(res.JSON200.InCategory)
			})
		}))
	}))
}
