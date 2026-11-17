package semester_backfill_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Southclaws/opt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/services/account/semester"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

func semesterMeta(term string) *openapi.Metadata {
	return &openapi.Metadata{"semester": map[string]any{"term": term}}
}

func threadTerm(th *openapi.Thread) string {
	if th.Meta == nil {
		return ""
	}
	if raw, ok := (*th.Meta)["semester"].(map[string]any); ok {
		if term, ok := raw["term"].(string); ok {
			return term
		}
	}
	return ""
}

func TestAdminSemesterBackfill(t *testing.T) {
	t.Parallel()

	integration.Test(t, nil, e2e.Setup(), fx.Invoke(func(
		root context.Context,
		lc fx.Lifecycle,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
	) {
		lc.Append(fx.StartHook(func() {
			adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			memberCtx, _ := e2e.WithAccount(root, aw, seed.Account_003_Baldur)
			admin := sh.WithSession(adminCtx)
			member := sh.WithSession(memberCtx)

			cat, err := cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{Name: uuid.NewString(), Colour: "#000"}, admin)
			tests.Ok(t, err, cat)

			create := func(title string, meta *openapi.Metadata) *openapi.Thread {
				res, err := cl.ThreadCreateWithResponse(root, openapi.ThreadInitialProps{
					Title:      title,
					Body:       opt.New("<p>Hier die Fragen</p>").Ptr(),
					Category:   opt.New(cat.JSON200.Id).Ptr(),
					Meta:       meta,
					Visibility: opt.New(openapi.VisibilityPublished).Ptr(),
				}, member)
				tests.Ok(t, err, res)
				return res.JSON200
			}

			current := semester.TermFor(time.Now()).String()

			implicit := create("Altklausur WS 22/23 Anatomie", nil)
			wrong := create("Gedächtnisprotokoll SoSe 2025", semesterMeta(current))
			create("Testat SS24", semesterMeta("2024-SS"))
			create("Mensa Öffnungszeiten", nil)
			create("WS 22/23 und SS 23 Sammlung", nil)

			t.Run("admin_can_change_any_thread_semester", func(t *testing.T) {
				th := create("Kons Skript", nil)
				res := tests.AssertRequest(cl.ThreadUpdateWithResponse(root, th.Slug, openapi.ThreadMutableProps{
					Meta: semesterMeta("2023-WS"),
				}, admin))(t, http.StatusOK)
				require.Equal(t, "2023-WS", threadTerm(res.JSON200))
			})

			t.Run("members_are_forbidden", func(t *testing.T) {
				res, err := cl.AdminSemesterBackfillStartWithResponse(root, member)
				tests.Status(t, err, res, http.StatusForbidden)
			})

			var proposals []openapi.SemesterBackfillProposal
			t.Run("proposes_only_differing_semesters", func(t *testing.T) {
				tests.AssertRequest(cl.AdminSemesterBackfillStartWithResponse(root, admin))(t, http.StatusOK)

				var status *openapi.SemesterBackfillStatus
				require.Eventually(t, func() bool {
					res := tests.AssertRequest(cl.AdminSemesterBackfillStatusWithResponse(root, admin))(t, http.StatusOK)
					status = res.JSON200
					return !status.Running
				}, 30*time.Second, 50*time.Millisecond)

				require.Equal(t, 6, status.Total)
				require.Equal(t, 6, status.Checked)
				require.Zero(t, status.Failed)

				byID := map[string]openapi.SemesterBackfillProposal{}
				for _, p := range status.Proposals {
					byID[p.ThreadId] = p
				}
				require.Len(t, byID, 2)

				require.Equal(t, "2022-WS", byID[implicit.Id].Suggested)
				require.Equal(t, current, byID[implicit.Id].Current)
				require.False(t, byID[implicit.Id].CurrentExplicit)
				require.Equal(t, openapi.SemesterBackfillProposalSource("pattern"), byID[implicit.Id].Source)

				require.Equal(t, "2025-SS", byID[wrong.Id].Suggested)
				require.Equal(t, current, byID[wrong.Id].Current)
				require.True(t, byID[wrong.Id].CurrentExplicit)
				require.Equal(t, "SoSe 2025", byID[wrong.Id].Evidence)

				proposals = status.Proposals
			})

			t.Run("apply_rejects_malformed_terms", func(t *testing.T) {
				res, err := cl.AdminSemesterBackfillApplyWithResponse(root, openapi.AdminSemesterBackfillApplyJSONRequestBody{Items: []openapi.SemesterBackfillApplyItem{
					{ThreadId: implicit.Id, Term: "2022-XX"},
				}}, admin)
				tests.Status(t, err, res, http.StatusBadRequest)
			})

			t.Run("apply_sets_semester_without_bumping_updated", func(t *testing.T) {
				before := tests.AssertRequest(cl.ThreadGetWithResponse(root, implicit.Slug, nil, admin))(t, http.StatusOK)

				var items []openapi.SemesterBackfillApplyItem
				for _, p := range proposals {
					items = append(items, openapi.SemesterBackfillApplyItem{ThreadId: p.ThreadId, Term: p.Suggested})
				}

				res := tests.AssertRequest(cl.AdminSemesterBackfillApplyWithResponse(root, openapi.AdminSemesterBackfillApplyJSONRequestBody{Items: items}, admin))(t, http.StatusOK)
				require.Equal(t, 2, res.JSON200.Applied)
				require.Empty(t, res.JSON200.Proposals)

				after := tests.AssertRequest(cl.ThreadGetWithResponse(root, implicit.Slug, nil, admin))(t, http.StatusOK)
				require.Equal(t, "2022-WS", threadTerm(after.JSON200))
				require.True(t, before.JSON200.UpdatedAt.Equal(after.JSON200.UpdatedAt))

				other := tests.AssertRequest(cl.ThreadGetWithResponse(root, wrong.Slug, nil, admin))(t, http.StatusOK)
				require.Equal(t, "2025-SS", threadTerm(other.JSON200))
			})

			t.Run("apply_twice_changes_nothing", func(t *testing.T) {
				res := tests.AssertRequest(cl.AdminSemesterBackfillApplyWithResponse(root, openapi.AdminSemesterBackfillApplyJSONRequestBody{Items: []openapi.SemesterBackfillApplyItem{
					{ThreadId: implicit.Id, Term: "2022-WS"},
				}}, admin))(t, http.StatusOK)
				require.Zero(t, res.JSON200.Applied)
			})
		}))
	}))
}
