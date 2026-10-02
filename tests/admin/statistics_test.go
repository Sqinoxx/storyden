package admin_test

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
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

func onePixelPNG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}

	return buf.Bytes()
}

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

				tagName := openapi.TagName("stats-" + xid.New().String())
				vis := openapi.VisibilityPublished
				createThread, err := cl.ThreadCreateWithResponse(memberCtx, openapi.ThreadInitialProps{
					Title:      "Statistics regression thread",
					Body:       opt.New("<p>Test content</p>").Ptr(),
					Category:   opt.New(catResp.JSON200.Id).Ptr(),
					Visibility: &vis,
					Tags:       &openapi.TagNameList{tagName},
				}, memberSession)
				require.NoError(t, err)
				r.Equal(http.StatusOK, createThread.StatusCode(), "%s", string(createThread.Body))

				threadID := createThread.JSON200.Id

				like, err := cl.LikePostAddWithResponse(adminCtx, threadID, adminSession)
				require.NoError(t, err)
				r.Equal(http.StatusOK, like.StatusCode(), "%s", string(like.Body))

				react, err := cl.PostReactAddWithResponse(adminCtx, threadID, openapi.ReactInitialProps{Emoji: "🦷"}, adminSession)
				require.NoError(t, err)
				r.Equal(http.StatusOK, react.StatusCode(), "%s", string(react.Body))

				reportResp, err := cl.ReportCreateWithResponse(adminCtx, openapi.ReportInitialProps{
					TargetId:   threadID,
					TargetKind: openapi.DatagraphItemKindThread,
				}, adminSession)
				require.NoError(t, err)
				r.Equal(http.StatusOK, reportResp.StatusCode(), "%s", string(reportResp.Body))

				png := onePixelPNG()
				filename := "statistics-regression.png"
				uploadAsset, err := cl.AssetUploadWithBodyWithResponse(memberCtx, &openapi.AssetUploadParams{
					ContentLength: int64(len(png)),
					Filename:      &filename,
				}, "image/png", bytes.NewReader(png), memberSession)
				require.NoError(t, err)
				r.Equal(http.StatusOK, uploadAsset.StatusCode(), "%s", string(uploadAsset.Body))

				// Simulate two more logins for the member, on top of the
				// admin and member sessions already issued above.
				sh.WithSession(memberCtx)
				sh.WithSession(memberCtx)

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

				a.Len(body.ThreadsByFachsemester, 13) // unknown (0) + semesters 1-11 + finished (-1)
				a.Len(body.AssetsByFachsemester, 13)

				fachsemester3, found := lo.Find(body.ThreadsByFachsemester, func(p openapi.StatisticsFachsemesterPoint) bool {
					return p.Semester == 3
				})
				r.True(found, "cohort for semester 3 should be present")
				a.GreaterOrEqual(fachsemester3.Count, 1)

				assetFachsemester3, found := lo.Find(body.AssetsByFachsemester, func(p openapi.StatisticsFachsemesterPoint) bool {
					return p.Semester == 3
				})
				r.True(found, "asset cohort for semester 3 should be present")
				a.GreaterOrEqual(assetFachsemester3.Count, 1)

				contributor, found := lo.Find(body.TopContributors, func(c openapi.StatisticsContributor) bool {
					return c.AccountId == openapi.Identifier(member.ID.String())
				})
				r.True(found, "the thread author should appear in top contributors")
				a.Equal(member.Handle, contributor.Handle)
				a.EqualValues(3, contributor.Semester)
				a.GreaterOrEqual(contributor.ThreadCount, 1)
				a.WithinDuration(time.Now(), contributor.LastThreadAt, time.Minute)

				a.Len(body.LoginsDaily, 30)
				a.Len(body.LoginsMonthly, 12)
				a.Len(body.LoginsYearly, 5)
				a.Len(body.LoginsByHour, 24)
				a.Len(body.LoginsByWeekday, 7)
				a.Len(body.ActiveAccountsDaily, 30)
				a.Len(body.ActiveAccountsMonthly, 12)
				a.Len(body.ActiveAccountsYearly, 5)
				a.Len(body.AssetsDaily, 30)
				a.Len(body.AssetsMonthly, 12)
				a.Len(body.AssetsYearly, 5)

				// admin session + 3 member sessions (initial + 2 simulated
				// repeat logins) were issued above, all today.
				todayLogins := body.LoginsDaily[len(body.LoginsDaily)-1]
				a.GreaterOrEqual(todayLogins.Count, 4)

				todayActiveAccounts := body.ActiveAccountsDaily[len(body.ActiveAccountsDaily)-1]
				a.GreaterOrEqual(todayActiveAccounts.Count, 1)

				todayAssets := body.AssetsDaily[len(body.AssetsDaily)-1]
				a.GreaterOrEqual(todayAssets.Count, 1)

				a.GreaterOrEqual(body.Totals.SessionsActive, 4)

				topCategory, found := lo.Find(body.TopCategories, func(c openapi.StatisticsCategoryPoint) bool {
					return c.CategoryId == catResp.JSON200.Id
				})
				r.True(found, "the created category should appear in top categories")
				a.GreaterOrEqual(topCategory.ThreadCount, 1)

				a.Len(body.LikesDaily, 30)
				a.Len(body.ReactsMonthly, 12)
				a.GreaterOrEqual(body.LikesDaily[len(body.LikesDaily)-1].Count, 1)
				a.GreaterOrEqual(body.ReactsDaily[len(body.ReactsDaily)-1].Count, 1)
				a.GreaterOrEqual(body.Totals.Likes, 1)
				a.GreaterOrEqual(body.Totals.Reacts, 1)
				a.GreaterOrEqual(body.Totals.Tags, 1)

				_, found = lo.Find(body.TopEmojis, func(e openapi.StatisticsEmojiPoint) bool {
					return e.Emoji == "🦷"
				})
				a.True(found, "the reaction emoji should appear in top emojis")

				likedThread, found := lo.Find(body.TopLikedThreads, func(p openapi.StatisticsThreadPoint) bool {
					return p.Id == threadID
				})
				r.True(found, "the liked thread should appear in top liked threads")
				a.Equal(1, likedThread.Count)
				a.Equal("Statistics regression thread", likedThread.Title)

				tag, found := lo.Find(body.TopTags, func(p openapi.StatisticsTagPoint) bool {
					return p.Name == tagName
				})
				r.True(found, "the thread's tag should appear in top tags")
				a.Equal(1, tag.ThreadCount)

				a.GreaterOrEqual(body.Totals.ReportsSubmitted, 1)
				a.GreaterOrEqual(body.Totals.ReportsLast30d, 1)

				a.GreaterOrEqual(body.Trends.Threads.Current, 1)
				a.GreaterOrEqual(body.Trends.Accounts.Current, 2)
				a.GreaterOrEqual(body.Trends.Likes.Current, 1)
				a.GreaterOrEqual(body.Trends.Reacts.Current, 1)
				a.GreaterOrEqual(body.Trends.Logins.Current, 4)
				a.GreaterOrEqual(body.Trends.Assets.Current, 1)
				a.GreaterOrEqual(body.Trends.ActiveAccounts.Current, 1)
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
