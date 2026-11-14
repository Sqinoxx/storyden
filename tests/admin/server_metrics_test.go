package admin_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
)

func TestAdminServerMetrics(t *testing.T) {
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

			t.Run("admin_sees_machine_metrics", func(t *testing.T) {
				var metrics *openapi.ServerMetrics
				require.EventuallyWithT(t, func(c *assert.CollectT) {
					resp, err := cl.AdminServerMetricsWithResponse(adminCtx, nil, adminSession)
					require.NoError(c, err)
					require.Equal(c, http.StatusOK, resp.StatusCode(), "%s", string(resp.Body))
					require.NotNil(c, resp.JSON200)
					require.Positive(c, resp.JSON200.Current.MemTotal)
					metrics = resp.JSON200
				}, 10*time.Second, 100*time.Millisecond)

				a := assert.New(t)
				a.Positive(metrics.Static.Cores)
				a.LessOrEqual(metrics.Current.MemUsed, metrics.Current.MemTotal)
				a.Positive(metrics.Current.DiskTotal)
				a.Positive(metrics.Current.Goroutines)
				a.Positive(metrics.Current.HeapBytes)
				a.NotNil(metrics.Current.DbSizeBytes)
				a.NotNil(metrics.Current.OcrQueue)
				a.NotNil(metrics.Current.AiEnabled)
				a.False(*metrics.Current.AiEnabled)
				a.LessOrEqual(len(metrics.History), 60)
			})

			t.Run("all_ranges_are_accepted", func(t *testing.T) {
				for _, rng := range []openapi.AdminServerMetricsParamsRange{"live", "1h", "24h"} {
					resp, err := cl.AdminServerMetricsWithResponse(adminCtx, &openapi.AdminServerMetricsParams{Range: &rng}, adminSession)
					require.NoError(t, err)
					require.Equal(t, http.StatusOK, resp.StatusCode(), "%s: %s", rng, string(resp.Body))
				}
			})

			t.Run("member_is_forbidden", func(t *testing.T) {
				memberCtx, _ := e2e.WithAccount(root, aw, seed.Account_004_Loki)
				resp, err := cl.AdminServerMetricsWithResponse(memberCtx, nil, sh.WithSession(memberCtx))
				require.NoError(t, err)
				require.Equal(t, http.StatusForbidden, resp.StatusCode())
			})

			t.Run("guest_is_unauthorised", func(t *testing.T) {
				resp, err := cl.AdminServerMetricsWithResponse(root, nil)
				require.NoError(t, err)
				require.Equal(t, http.StatusUnauthorized, resp.StatusCode())
			})
		}))
	}))
}
