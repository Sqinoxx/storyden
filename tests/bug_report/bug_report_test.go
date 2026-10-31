package bug_report_test

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"strings"
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
	"github.com/Southclaws/storyden/tests"
)

func onePixelPNG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}

	return buf.Bytes()
}

func TestBugReports(t *testing.T) {
	t.Parallel()

	integration.Test(t, nil, e2e.Setup(), fx.Invoke(func(
		lc fx.Lifecycle,
		root context.Context,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
	) {
		lc.Append(fx.StartHook(func() {
			a := assert.New(t)
			r := require.New(t)

			adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			adminSession := sh.WithSession(adminCtx)

			memberCtx, member := e2e.WithAccount(root, aw, seed.Account_003_Baldur)
			memberSession := sh.WithSession(memberCtx)

			t.Run("unauthenticated", func(t *testing.T) {
				resp, err := cl.BugReportCreateWithResponse(root, openapi.BugReportInitialProps{
					Category:    openapi.BugReportCategoryBug,
					Description: "broken",
				})
				tests.Status(t, err, resp, http.StatusUnauthorized)
			})

			t.Run("empty_description", func(t *testing.T) {
				resp, err := cl.BugReportCreateWithResponse(root, openapi.BugReportInitialProps{
					Category:    openapi.BugReportCategoryBug,
					Description: "   ",
				}, memberSession)
				tests.Status(t, err, resp, http.StatusBadRequest)
			})

			t.Run("too_long_description", func(t *testing.T) {
				resp, err := cl.BugReportCreateWithResponse(root, openapi.BugReportInitialProps{
					Category:    openapi.BugReportCategoryBug,
					Description: strings.Repeat("a", 5001),
				}, memberSession)
				tests.Status(t, err, resp, http.StatusBadRequest)
			})

			t.Run("invalid_category", func(t *testing.T) {
				resp, err := cl.BugReportCreateWithResponse(root, openapi.BugReportInitialProps{
					Category:    "nonsense",
					Description: "x",
				}, memberSession)
				tests.Status(t, err, resp, http.StatusBadRequest)
			})

			t.Run("member_cannot_access_admin_endpoints", func(t *testing.T) {
				list, err := cl.AdminBugReportListWithResponse(root, &openapi.AdminBugReportListParams{}, memberSession)
				tests.Status(t, err, list, http.StatusForbidden)
			})

			t.Run("suggestion_feedback", func(t *testing.T) {
				authorCtx, author := e2e.WithAccount(root, aw, seed.Account_004_Loki)
				authorSession := sh.WithSession(authorCtx)

				otherCtx, _ := e2e.WithAccount(root, aw, seed.Account_006_Freyja)
				otherSession := sh.WithSession(otherCtx)

				suggestion, err := cl.BugReportCreateWithResponse(root, openapi.BugReportInitialProps{
					Category:    openapi.BugReportCategorySuggestion,
					Description: "Add a dark mode for the calendar",
				}, authorSession)
				tests.Ok(t, err, suggestion)

				bug, err := cl.BugReportCreateWithResponse(root, openapi.BugReportInitialProps{
					Category:    openapi.BugReportCategoryBug,
					Description: "Calendar crashes",
				}, authorSession)
				tests.Ok(t, err, bug)

				suggestionCategory := openapi.BugReportCategorySuggestion

				own, err := cl.BugReportListOwnWithResponse(root, &openapi.BugReportListOwnParams{}, authorSession)
				tests.Ok(t, err, own)
				a.Len(own.JSON200.BugReports, 2)

				ownSuggestions, err := cl.BugReportListOwnWithResponse(root, &openapi.BugReportListOwnParams{Category: &suggestionCategory}, authorSession)
				tests.Ok(t, err, ownSuggestions)
				r.Len(ownSuggestions.JSON200.BugReports, 1)
				a.Equal(suggestion.JSON200.Id, ownSuggestions.JSON200.BugReports[0].Id)

				others, err := cl.BugReportListOwnWithResponse(root, &openapi.BugReportListOwnParams{}, otherSession)
				tests.Ok(t, err, others)
				a.Empty(others.JSON200.BugReports)

				unauthenticated, err := cl.BugReportListOwnWithResponse(root, &openapi.BugReportListOwnParams{})
				tests.Status(t, err, unauthenticated, http.StatusUnauthorized)

				adminSuggestions, err := cl.AdminBugReportListWithResponse(root, &openapi.AdminBugReportListParams{Category: &suggestionCategory}, adminSession)
				tests.Ok(t, err, adminSuggestions)
				for _, rep := range adminSuggestions.JSON200.BugReports {
					a.Equal(openapi.BugReportCategorySuggestion, rep.Category)
				}

				bugResolved, err := cl.AdminBugReportUpdateWithResponse(root, bug.JSON200.Id, openapi.BugReportMutableProps{
					Status: openapi.BugReportStatusResolved,
				}, adminSession)
				tests.Ok(t, err, bugResolved)

				authorEvents := func() []openapi.NotificationEvent {
					notifications, err := cl.NotificationListWithResponse(root, &openapi.NotificationListParams{}, authorSession)
					if err != nil || notifications.JSON200 == nil {
						return nil
					}
					var events []openapi.NotificationEvent
					for _, n := range notifications.JSON200.Notifications {
						events = append(events, n.Event)
					}
					return events
				}

				for _, step := range []struct {
					status openapi.BugReportStatus
					event  openapi.NotificationEvent
				}{
					{openapi.BugReportStatusPlanned, openapi.SuggestionPlanned},
					{openapi.BugReportStatusResolved, openapi.SuggestionImplemented},
				} {
					updated, err := cl.AdminBugReportUpdateWithResponse(root, suggestion.JSON200.Id, openapi.BugReportMutableProps{
						Status: step.status,
					}, adminSession)
					tests.Ok(t, err, updated)
					a.Equal(step.status, updated.JSON200.Status)

					r.Eventually(func() bool {
						for _, e := range authorEvents() {
							if e == step.event {
								return true
							}
						}
						return false
					}, 5*time.Second, 100*time.Millisecond, "expected author to be notified with %s", step.event)
				}

				unchanged, err := cl.AdminBugReportUpdateWithResponse(root, suggestion.JSON200.Id, openapi.BugReportMutableProps{
					Status: openapi.BugReportStatusResolved,
				}, adminSession)
				tests.Ok(t, err, unchanged)

				count := map[openapi.NotificationEvent]int{}
				for _, e := range authorEvents() {
					count[e]++
				}
				a.Equal(1, count[openapi.SuggestionPlanned])
				a.Equal(1, count[openapi.SuggestionImplemented])
				a.Zero(count[openapi.SuggestionDeclined])

				final, err := cl.BugReportListOwnWithResponse(root, &openapi.BugReportListOwnParams{Category: &suggestionCategory}, authorSession)
				tests.Ok(t, err, final)
				r.Len(final.JSON200.BugReports, 1)
				a.Equal(openapi.BugReportStatusResolved, final.JSON200.BugReports[0].Status)
				a.Equal(author.ID.String(), final.JSON200.BugReports[0].Author.Id)
			})

			t.Run("lifecycle", func(t *testing.T) {
				img := onePixelPNG()
				filename := "screenshot.png"
				upload, err := cl.AssetUploadWithBodyWithResponse(root, &openapi.AssetUploadParams{
					ContentLength: int64(len(img)),
					Filename:      &filename,
				}, "image/png", bytes.NewReader(img), memberSession)
				tests.Ok(t, err, upload)

				pageURL := "https://example.com/t/some-thread"
				userAgent := "test-agent"
				created, err := cl.BugReportCreateWithResponse(root, openapi.BugReportInitialProps{
					Category:    openapi.BugReportCategorySuggestion,
					Description: "  The button does nothing  ",
					PageUrl:     &pageURL,
					UserAgent:   &userAgent,
					AssetId:     &upload.JSON200.Id,
				}, memberSession)
				tests.Ok(t, err, created)

				rep := created.JSON200
				a.Equal("The button does nothing", rep.Description)
				a.Equal(openapi.BugReportCategorySuggestion, rep.Category)
				a.Equal(openapi.BugReportStatusOpen, rep.Status)
				a.Equal(member.ID.String(), rep.Author.Id)
				a.Equal(&pageURL, rep.PageUrl)
				a.Equal(&userAgent, rep.UserAgent)
				if a.NotNil(rep.Screenshot) {
					a.Equal(upload.JSON200.Id, rep.Screenshot.Id)
				}

				findIn := func(list []openapi.BugReport) *openapi.BugReport {
					for i := range list {
						if list[i].Id == rep.Id {
							return &list[i]
						}
					}
					return nil
				}

				open := openapi.BugReportStatusOpen
				openList, err := cl.AdminBugReportListWithResponse(root, &openapi.AdminBugReportListParams{Status: &open}, adminSession)
				tests.Ok(t, err, openList)
				a.NotNil(findIn(openList.JSON200.BugReports))

				updated, err := cl.AdminBugReportUpdateWithResponse(root, rep.Id, openapi.BugReportMutableProps{
					Status: openapi.BugReportStatusResolved,
				}, adminSession)
				tests.Ok(t, err, updated)
				a.Equal(openapi.BugReportStatusResolved, updated.JSON200.Status)

				openList, err = cl.AdminBugReportListWithResponse(root, &openapi.AdminBugReportListParams{Status: &open}, adminSession)
				tests.Ok(t, err, openList)
				a.Nil(findIn(openList.JSON200.BugReports))

				resolved := openapi.BugReportStatusResolved
				resolvedList, err := cl.AdminBugReportListWithResponse(root, &openapi.AdminBugReportListParams{Status: &resolved}, adminSession)
				tests.Ok(t, err, resolvedList)
				a.NotNil(findIn(resolvedList.JSON200.BugReports))

				memberUpdate, err := cl.AdminBugReportUpdateWithResponse(root, rep.Id, openapi.BugReportMutableProps{
					Status: openapi.BugReportStatusOpen,
				}, memberSession)
				tests.Status(t, err, memberUpdate, http.StatusForbidden)

				memberDelete, err := cl.AdminBugReportDeleteWithResponse(root, rep.Id, memberSession)
				tests.Status(t, err, memberDelete, http.StatusForbidden)

				r.Eventually(func() bool {
					notifications, err := cl.NotificationListWithResponse(root, &openapi.NotificationListParams{}, adminSession)
					if err != nil || notifications.JSON200 == nil {
						return false
					}
					for _, n := range notifications.JSON200.Notifications {
						if n.Event == openapi.BugReportSubmitted && n.Source != nil && n.Source.Id == member.ID.String() {
							return true
						}
					}
					return false
				}, 5*time.Second, 100*time.Millisecond, "expected admin to be notified of bug report")

				deleted, err := cl.AdminBugReportDeleteWithResponse(root, rep.Id, adminSession)
				tests.Status(t, err, deleted, http.StatusNoContent)

				deletedAgain, err := cl.AdminBugReportDeleteWithResponse(root, rep.Id, adminSession)
				tests.Status(t, err, deletedAgain, http.StatusNotFound)

				all, err := cl.AdminBugReportListWithResponse(root, &openapi.AdminBugReportListParams{}, adminSession)
				tests.Ok(t, err, all)
				a.Nil(findIn(all.JSON200.BugReports))
			})
		}))
	}))
}
