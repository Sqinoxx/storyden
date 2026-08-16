package admin_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Southclaws/opt"
	"github.com/rs/xid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/services/account/semester"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
)

func TestAdminStatistics(t *testing.T) {
	t.Parallel()

	integration.Test(t, nil, e2e.Setup(), fx.Invoke(func(
		lc fx.Lifecycle,
		root context.Context,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
	) {
		lc.Append(fx.StartHook(func() {
			t.Run("returns_usage_statistics_for_admin", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
				adminSession := sh.WithSession(adminCtx)

				memberCtx, member := e2e.WithAccount(root, aw, seed.Account_004_Loki)
				memberSession := sh.WithSession(memberCtx)

				updateSemester, err := cl.AccountUpdateWithResponse(memberCtx, openapi.AccountMutableProps{
					Meta: &openapi.Metadata{
						semester.MetadataKey: map[string]any{"semester": 3},
					},
				}, memberSession)
				require.NoError(t, err)
				r.Equal(http.StatusOK, updateSemester.StatusCode(), "%s", string(updateSemester.Body))

				catResp, err := cl.CategoryCreateWithResponse(adminCtx, openapi.CategoryInitialProps{
					Name:   "Statistics" + xid.New().String(),
					Colour: "#123456",
				}, adminSession)
				require.NoError(t, err)
				r.Equal(http.StatusOK, catResp.StatusCode(), "%s", string(catResp.Body))

				vis := openapi.VisibilityPublished
				createThread, err := cl.ThreadCreateWithResponse(memberCtx, openapi.ThreadInitialProps{
					Title:      "Statistics regression thread",
					Body:       opt.New("<p>Test content</p>").Ptr(),
					Category:   opt.New(catResp.JSON200.Id).Ptr(),
					Visibility: &vis,
				}, memberSession)
				require.NoError(t, err)
				r.Equal(http.StatusOK, createThread.StatusCode(), "%s", string(createThread.Body))

				stats, err := cl.AdminStatisticsWithResponse(adminCtx, adminSession)
				r.NoError(err)
				r.Equal(http.StatusOK, stats.StatusCode())
				r.NotNil(stats.JSON200)

				body := stats.JSON200

				a.GreaterOrEqual(body.Totals.Accounts, 2)
				a.GreaterOrEqual(body.Totals.Threads, 1)
				a.GreaterOrEqual(body.Totals.ActiveAccounts30d, 1)

				a.Len(body.AccountsDaily, 30)
				a.Len(body.AccountsMonthly, 12)
				a.Len(body.AccountsYearly, 5)
				a.Len(body.ThreadsDaily, 30)
				a.Len(body.ThreadsMonthly, 12)
				a.Len(body.ThreadsYearly, 5)
				a.Len(body.ThreadsBySemester, 8)

				today := body.AccountsDaily[len(body.AccountsDaily)-1]
				a.Equal(time.Now().UTC().Format("2006-01-02"), today.Date.Format("2006-01-02"))
				a.GreaterOrEqual(today.Count, 2)

				todayThreads := body.ThreadsDaily[len(body.ThreadsDaily)-1]
				a.GreaterOrEqual(todayThreads.Count, 1)

				currentTerm := semester.TermFor(time.Now().UTC()).String()
				currentPoint, found := lo.Find(body.ThreadsBySemester, func(p openapi.StatisticsSemesterPoint) bool {
					return p.Term == currentTerm
				})
				r.True(found, "current term should be present in the semester series")
				a.GreaterOrEqual(currentPoint.Count, 1)

				a.Len(body.ThreadsByFachsemester, 12) // unknown (0) + semesters 1-11

				fachsemester3, found := lo.Find(body.ThreadsByFachsemester, func(p openapi.StatisticsFachsemesterPoint) bool {
					return p.Semester == 3
				})
				r.True(found, "cohort for semester 3 should be present")
				a.GreaterOrEqual(fachsemester3.Count, 1)

				contributor, found := lo.Find(body.TopContributors, func(c openapi.StatisticsContributor) bool {
					return c.AccountId == openapi.Identifier(member.ID.String())
				})
				r.True(found, "the thread author should appear in top contributors")
				a.Equal(member.Handle, contributor.Handle)
				a.EqualValues(3, contributor.Semester)
				a.GreaterOrEqual(contributor.ThreadCount, 1)
				a.WithinDuration(time.Now(), contributor.LastThreadAt, time.Minute)
			})

			t.Run("requires_admin_permission", func(t *testing.T) {
				r := require.New(t)

				memberCtx, _ := e2e.WithAccount(root, aw, seed.Account_004_Loki)
				memberSession := sh.WithSession(memberCtx)

				stats, err := cl.AdminStatisticsWithResponse(memberCtx, memberSession)
				r.NoError(err)
				r.Equal(http.StatusForbidden, stats.StatusCode())
			})
		}))
	}))
}
