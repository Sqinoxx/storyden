package account_test

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
	"github.com/Southclaws/storyden/app/services/account/semester"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

func TestAccountSemester(t *testing.T) {
	t.Parallel()

	integration.Test(t, nil, e2e.Setup(), fx.Invoke(func(
		lc fx.Lifecycle,
		root context.Context,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
		rollover *semester.Rollover,
	) {
		lc.Append(fx.StartHook(func() {
			ctx, acc := e2e.WithAccount(root, aw, seed.Account_002_Frigg)
			session := sh.WithSession(ctx)

			readAcademic := func(t *testing.T, meta openapi.Metadata) map[string]any {
				t.Helper()

				academic, ok := meta[semester.MetadataKey].(map[string]any)
				require.True(t, ok, "academic metadata missing from %v", meta)

				return academic
			}

			t.Run("server_stamps_the_term_the_semester_was_chosen_in", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				update := tests.AssertRequest(cl.AccountUpdateWithResponse(root, openapi.AccountMutableProps{
					Meta: &openapi.Metadata{
						semester.MetadataKey: map[string]any{"semester": 3},
					},
				}, session))(t, http.StatusOK)
				r.NotNil(update.JSON200)

				academic := readAcademic(t, update.JSON200.Meta)
				a.EqualValues(3, academic["semester"])
				a.Equal(semester.TermFor(time.Now()).String(), academic["term"])
			})

			t.Run("out_of_range_values_are_clamped", func(t *testing.T) {
				a := assert.New(t)

				update := tests.AssertRequest(cl.AccountUpdateWithResponse(root, openapi.AccountMutableProps{
					Meta: &openapi.Metadata{
						semester.MetadataKey: map[string]any{"semester": 99},
					},
				}, session))(t, http.StatusOK)

				academic := readAcademic(t, update.JSON200.Meta)
				a.EqualValues(semester.Max, academic["semester"])
			})

			t.Run("updating_the_semester_preserves_other_settings", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				tests.AssertRequest(cl.AccountUpdateWithResponse(root, openapi.AccountMutableProps{
					Meta: &openapi.Metadata{
						"editor": map[string]any{"mode": "markdown"},
					},
				}, session))(t, http.StatusOK)

				update := tests.AssertRequest(cl.AccountUpdateWithResponse(root, openapi.AccountMutableProps{
					Meta: &openapi.Metadata{
						semester.MetadataKey: map[string]any{"semester": 5},
					},
				}, session))(t, http.StatusOK)
				meta := update.JSON200.Meta
				editor, ok := meta["editor"].(map[string]any)
				r.True(ok, "editor settings were clobbered by the semester update: %v", meta)
				a.Equal("markdown", editor["mode"])

				a.EqualValues(5, readAcademic(t, update.JSON200.Meta)["semester"])
			})

			t.Run("read_projects_the_semester_forward", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				// Anchor two terms in the past, so reads must report +2 without
				// the rollover job having run.
				twoTermsAgo := semester.TermFor(time.Now().AddDate(-1, 0, 0))

				_, err := aw.Update(root, acc.ID, account_writer.SetMetadata(map[string]any{
					semester.MetadataKey: map[string]any{
						"semester": 4,
						"term":     twoTermsAgo.String(),
					},
				}))
				r.NoError(err)

				get := tests.AssertRequest(cl.AccountGetWithResponse(root, session))(t, http.StatusOK)
				r.NotNil(get.JSON200)

				academic := readAcademic(t, get.JSON200.Meta)
				a.EqualValues(6, academic["semester"])
				a.Equal(semester.TermFor(time.Now()).String(), academic["term"])
			})

			t.Run("rollover_persists_the_projected_value", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				staleTerm := semester.TermFor(time.Now().AddDate(-1, 0, 0))

				_, err := aw.Update(root, acc.ID, account_writer.SetMetadata(map[string]any{
					semester.MetadataKey: map[string]any{
						"semester": 2,
						"term":     staleTerm.String(),
					},
				}))
				r.NoError(err)

				updated := rollover.Sweep(root, time.Now())
				a.Positive(updated, "rollover should have advanced at least this account")

				stored, err := aw.GetByID(root, acc.ID)
				r.NoError(err)

				academic, ok := stored.Metadata[semester.MetadataKey].(map[string]any)
				r.True(ok)
				a.EqualValues(4, academic["semester"])
				a.Equal(semester.TermFor(time.Now()).String(), academic["term"])

				// A second sweep is a no-op now that the term is current.
				a.Zero(rollover.Sweep(root, time.Now()))
			})
		}))
	}))
}
