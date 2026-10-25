package account_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/Southclaws/opt"
	"github.com/rs/xid"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/services/account/semester"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/internal/utils"
)

func TestProfileStats(t *testing.T) {
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

			getStats := func() *openapi.ProfileStatsResult {
				res, err := cl.ProfileStatsWithResponse(root, adminSession)
				r.NoError(err)
				r.Equal(http.StatusOK, res.StatusCode(), "%s", string(res.Body))
				return res.JSON200
			}

			semesterCount := func(s *openapi.ProfileStatsResult, sem int) int {
				for _, p := range s.BySemester {
					if p.Semester == sem {
						return p.Count
					}
				}
				t.Fatalf("missing semester bucket %d", sem)
				return 0
			}

			signup := func() (string, context.Context, openapi.RequestEditorFn) {
				handle := "stats-" + xid.New().String()
				res, err := cl.AuthPasswordSignupWithResponse(root, nil, openapi.AuthPair{Identifier: handle, Token: "password"})
				r.NoError(err)
				r.Equal(http.StatusOK, res.StatusCode(), "%s", string(res.Body))
				id := account.AccountID(utils.Must(xid.FromString(res.JSON200.Id)))
				ctx := e2e.WithAccountID(root, id)
				return handle, ctx, sh.WithSession(ctx)
			}

			before := getStats()
			r.Len(before.BySemester, semester.Max+2)

			_, activeCtx, activeSession := signup()
			goneHandle, _, _ := signup()

			upd, err := cl.AccountUpdateWithResponse(activeCtx, openapi.AccountMutableProps{
				Meta: &openapi.Metadata{
					semester.MetadataKey: map[string]any{"semester": 3},
				},
			}, activeSession)
			r.NoError(err)
			r.Equal(http.StatusOK, upd.StatusCode(), "%s", string(upd.Body))

			cat, err := cl.CategoryCreateWithResponse(adminCtx, openapi.CategoryInitialProps{
				Name:   "Stats" + xid.New().String(),
				Colour: "#123456",
			}, adminSession)
			r.NoError(err)
			r.Equal(http.StatusOK, cat.StatusCode(), "%s", string(cat.Body))

			thread, err := cl.ThreadCreateWithResponse(activeCtx, openapi.ThreadInitialProps{
				Title:      "Stats thread",
				Body:       opt.New("<p>Hello</p>").Ptr(),
				Category:   opt.New(cat.JSON200.Id).Ptr(),
				Visibility: opt.New(openapi.VisibilityPublished).Ptr(),
			}, activeSession)
			r.NoError(err)
			r.Equal(http.StatusOK, thread.StatusCode(), "%s", string(thread.Body))

			after := getStats()
			r.Equal(before.Total+2, after.Total)
			r.Equal(before.Joined30d+2, after.Joined30d)
			r.Equal(before.Active30d+1, after.Active30d)
			r.Equal(semesterCount(before, 3)+1, semesterCount(after, 3))

			ban, err := cl.AdminAccountBanCreateWithResponse(root, goneHandle, adminSession)
			r.NoError(err)
			r.Equal(http.StatusOK, ban.StatusCode(), "%s", string(ban.Body))

			del, err := cl.AdminAccountDeleteWithResponse(root, goneHandle, adminSession)
			r.NoError(err)
			r.Equal(http.StatusNoContent, del.StatusCode(), "%s", string(del.Body))

			afterDelete := getStats()
			r.Equal(after.Total-1, afterDelete.Total)
			r.Equal(after.Joined30d-1, afterDelete.Joined30d)

			unauth, err := cl.ProfileStatsWithResponse(root)
			r.NoError(err)
			r.NotEqual(http.StatusInternalServerError, unauth.StatusCode())
		}))
	}))
}
