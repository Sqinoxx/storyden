package drive_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/config"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

// The admin tab and the navigation entry are both gated on this capability, so
// a configured installation that fails to advertise it has no reachable UI at
// all, however well the endpoints behave.
func TestDriveCapabilityAdvertised(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{GoogleDriveServiceAccountJSON: "mock"}

	integration.Test(t, cfg, e2e.Setup(), withFakeDrive(fakeDrive()), fx.Invoke(func(
		lc fx.Lifecycle,
		root context.Context,
		cl *openapi.ClientWithResponses,
	) {
		lc.Append(fx.StartHook(func() {
			r := require.New(t)

			resp := tests.AssertRequest(cl.GetInfoWithResponse(root))(t, http.StatusOK)

			r.NotNil(resp.JSON200)
			r.Contains(resp.JSON200.Capabilities, openapi.InstanceCapability("drive"))
		}))
	}))
}

func TestDriveCapabilityAbsentWhenUnconfigured(t *testing.T) {
	t.Parallel()

	integration.Test(t, nil, e2e.Setup(), fx.Invoke(func(
		lc fx.Lifecycle,
		root context.Context,
		cl *openapi.ClientWithResponses,
	) {
		lc.Append(fx.StartHook(func() {
			r := require.New(t)

			resp := tests.AssertRequest(cl.GetInfoWithResponse(root))(t, http.StatusOK)

			r.NotNil(resp.JSON200)
			r.NotContains(resp.JSON200.Capabilities, openapi.InstanceCapability("drive"))
		}))
	}))
}
