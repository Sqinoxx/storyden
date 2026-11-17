package search_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Southclaws/opt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/services/semdex/semdex_indexer"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/config"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

// Rubber dam threads and the colloquial word for it share an embedding axis
// but no words, so only the semantic index can connect them.
func meaningModel(t *testing.T) *httptest.Server {
	axis := func(text string) []float32 {
		lower := strings.ToLower(text)
		switch {
		case strings.Contains(lower, "kofferdam"), strings.Contains(lower, "spanngummi"):
			return []float32{1, 0, 0}
		case strings.Contains(lower, "zahnspange"):
			return []float32{0, 1, 0}
		default:
			return []float32{0, 0, 1}
		}
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		w.Header().Set("Content-Type", "application/json")
		input, _ := body["input"].(string)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "list",
			"data":   []map[string]any{{"object": "embedding", "index": 0, "embedding": axis(input)}},
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestSemanticSearch(t *testing.T) {
	t.Parallel()

	srv := meaningModel(t)

	integration.Test(t, &config.Config{
		LanguageModelProvider:       "openai",
		OpenAIBaseURL:               srv.URL + "/v1",
		LanguageModelChatModel:      "m",
		LanguageModelEmbeddingModel: "e",
		SemdexProvider:              "chromem",
		SemdexLocalPath:             t.TempDir(),
	}, e2e.Setup(), fx.Invoke(func(
		root context.Context,
		lc fx.Lifecycle,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
		idx *semdex_indexer.Indexer,
	) {
		lc.Append(fx.StartHook(func() {
			adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			memberCtx, _ := e2e.WithAccount(root, aw, seed.Account_003_Baldur)
			member := sh.WithSession(memberCtx)

			cat, err := cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{Name: uuid.NewString(), Colour: "#000"}, sh.WithSession(adminCtx))
			tests.Ok(t, err, cat)

			create := func(title string) string {
				res, err := cl.ThreadCreateWithResponse(root, openapi.ThreadInitialProps{
					Title:      title,
					Body:       opt.New("<p>" + title + ", wer hat Tipps?</p>").Ptr(),
					Category:   opt.New(cat.JSON200.Id).Ptr(),
					Visibility: opt.New(openapi.VisibilityPublished).Ptr(),
				}, member)
				tests.Ok(t, err, res)
				return res.JSON200.Id
			}

			dam := create("Kofferdam im Phantomkurs anlegen " + uuid.NewString()[:6])
			braces := create("Zahnspange mit Brackets " + uuid.NewString()[:6])

			_, err = idx.Sync(root)
			require.NoError(t, err)

			search := func(t *testing.T, q string, page string) *openapi.DatagraphSearchResult {
				res, err := cl.DatagraphSearchWithResponse(root, &openapi.DatagraphSearchParams{
					Q:    opt.New(q).Ptr(),
					Kind: &openapi.DatagraphKindQuery{openapi.DatagraphItemKindThread},
					Page: opt.New(page).Ptr(),
				}, member)
				tests.Ok(t, err, res)
				return res.JSON200
			}

			t.Run("finds_meaning_without_shared_words", func(t *testing.T) {
				res := search(t, "Spanngummi", "1")
				require.NotNil(t, res.SemanticMatches)
				require.Equal(t, []openapi.Identifier{openapi.Identifier(dam)}, *res.SemanticMatches)

				raw, _ := json.Marshal(res.Items)
				require.Contains(t, string(raw), dam)
				require.NotContains(t, string(raw), braces)
			})

			t.Run("keyword_hit_is_not_labelled", func(t *testing.T) {
				res := search(t, "Kofferdam", "1")
				raw, _ := json.Marshal(res.Items)
				require.Contains(t, string(raw), dam)
				require.Nil(t, res.SemanticMatches)
			})

			t.Run("later_pages_are_keyword_only", func(t *testing.T) {
				res := search(t, "Spanngummi", "2")
				require.Nil(t, res.SemanticMatches)
			})
		}))
	}))
}
