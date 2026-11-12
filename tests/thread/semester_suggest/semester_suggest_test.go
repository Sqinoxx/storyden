package semester_suggest_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/Southclaws/opt"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

func TestThreadSuggestSemester(t *testing.T) {
	t.Parallel()

	integration.Test(t, nil, e2e.Setup(), fx.Invoke(func(
		root context.Context,
		lc fx.Lifecycle,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
	) {
		lc.Append(fx.StartHook(func() {
			memberCtx, _ := e2e.WithAccount(root, aw, seed.Account_003_Baldur)
			session := sh.WithSession(memberCtx)

			t.Run("explicit_term_in_filename", func(t *testing.T) {
				r := require.New(t)
				res, err := cl.ThreadSuggestSemesterWithResponse(root, openapi.ThreadSemesterSuggestionRequest{
					Title:     opt.New("Altklausur Anatomie").Ptr(),
					Body:      opt.New("<p>Hier die Fragen</p>").Ptr(),
					Filenames: &[]string{"Anatomie_WS22-23.pdf"},
				}, session)
				tests.Ok(t, err, res)
				r.Equal("2022-WS", opt.NewPtr(res.JSON200.Term).OrZero())
				r.Equal(openapi.ThreadSemesterSuggestionSource("pattern"), opt.NewPtr(res.JSON200.Source).OrZero())
				r.Equal("WS22-23", opt.NewPtr(res.JSON200.Evidence).OrZero())
			})

			t.Run("explicit_term_in_rich_text_body", func(t *testing.T) {
				r := require.New(t)
				res, err := cl.ThreadSuggestSemesterWithResponse(root, openapi.ThreadSemesterSuggestionRequest{
					Body: opt.New("<p>Gedächtnisprotokoll <strong>SoSe 2025</strong></p>").Ptr(),
				}, session)
				tests.Ok(t, err, res)
				r.Equal("2025-SS", opt.NewPtr(res.JSON200.Term).OrZero())
			})

			t.Run("indirect_reference_without_model_is_empty", func(t *testing.T) {
				r := require.New(t)
				res, err := cl.ThreadSuggestSemesterWithResponse(root, openapi.ThreadSemesterSuggestionRequest{
					Body: opt.New("<p>Klausur vom 14.02.2024</p>").Ptr(),
				}, session)
				tests.Ok(t, err, res)
				r.Nil(res.JSON200.Term)
			})

			t.Run("ambiguous_is_empty", func(t *testing.T) {
				r := require.New(t)
				res, err := cl.ThreadSuggestSemesterWithResponse(root, openapi.ThreadSemesterSuggestionRequest{
					Title: opt.New("WS 22/23 oder SS 23?").Ptr(),
				}, session)
				tests.Ok(t, err, res)
				r.Nil(res.JSON200.Term)
			})

			t.Run("requires_authentication", func(t *testing.T) {
				res, err := cl.ThreadSuggestSemesterWithResponse(root, openapi.ThreadSemesterSuggestionRequest{
					Title: opt.New("WS 22/23").Ptr(),
				})
				tests.Status(t, err, res, http.StatusUnauthorized)
			})
		}))
	}))
}
