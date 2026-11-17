package similar_test

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

func topicModel(t *testing.T) *httptest.Server {
	axis := func(text string) []float32 {
		lower := strings.ToLower(text)
		switch {
		case strings.Contains(lower, "kofferdam"):
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

func ids(t *testing.T, list *openapi.ThreadSimilarList) []string {
	raw, err := json.Marshal(list.Items)
	require.NoError(t, err)
	var items []struct {
		Ref struct {
			ID string `json:"id"`
		} `json:"ref"`
	}
	require.NoError(t, json.Unmarshal(raw, &items))
	out := []string{}
	for _, it := range items {
		out = append(out, it.Ref.ID)
	}
	return out
}

func TestSimilarThreads(t *testing.T) {
	t.Parallel()

	srv := topicModel(t)

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

			create := func(title string) (string, string) {
				res, err := cl.ThreadCreateWithResponse(root, openapi.ThreadInitialProps{
					Title:      title + " " + uuid.NewString()[:6],
					Body:       opt.New("<p>" + title + ", wer hat Erfahrungen?</p>").Ptr(),
					Category:   opt.New(cat.JSON200.Id).Ptr(),
					Visibility: opt.New(openapi.VisibilityPublished).Ptr(),
				}, member)
				tests.Ok(t, err, res)
				return res.JSON200.Id, res.JSON200.Slug
			}

			dam1, dam1Slug := create("Kofferdam anlegen im Phantomkurs")
			dam2, _ := create("Kofferdam Klammer hält nicht")
			braces, _ := create("Zahnspange mit Brackets")

			_, err = idx.Sync(root)
			require.NoError(t, err)

			t.Run("flag_is_on_with_semantic_index", func(t *testing.T) {
				info := tests.AssertRequest(cl.GetInfoWithResponse(root))(t, http.StatusOK)
				require.True(t, info.JSON200.AiFeatures.SimilarThreads)
			})

			t.Run("related_threads_exclude_the_thread_itself", func(t *testing.T) {
				res := tests.AssertRequest(cl.ThreadRelatedWithResponse(root, dam1Slug))(t, http.StatusOK)
				got := ids(t, res.JSON200)
				require.Equal(t, []string{dam2}, got)
				require.NotContains(t, got, braces)
			})

			t.Run("similar_threads_for_a_draft", func(t *testing.T) {
				res := tests.AssertRequest(cl.ThreadSimilarWithResponse(root, openapi.ThreadTagSuggestionRequest{
					Title: opt.New("Kofferdam richtig spannen").Ptr(),
					Body:  opt.New("<p>Bei mir reißt der Gummi ständig.</p>").Ptr(),
				}, member))(t, http.StatusOK)
				got := ids(t, res.JSON200)
				require.ElementsMatch(t, []string{dam1, dam2}, got)
			})

			t.Run("short_draft_has_no_suggestions", func(t *testing.T) {
				res := tests.AssertRequest(cl.ThreadSimilarWithResponse(root, openapi.ThreadTagSuggestionRequest{
					Title: opt.New("Kofferdam").Ptr(),
				}, member))(t, http.StatusOK)
				require.Empty(t, ids(t, res.JSON200))
			})

			t.Run("draft_suggestions_need_an_account", func(t *testing.T) {
				res, err := cl.ThreadSimilarWithResponse(root, openapi.ThreadTagSuggestionRequest{
					Title: opt.New("Kofferdam richtig spannen bitte").Ptr(),
				})
				tests.Status(t, err, res, http.StatusUnauthorized)
			})
		}))
	}))
}

func TestSimilarThreadsWithoutSemanticIndex(t *testing.T) {
	t.Parallel()

	integration.Test(t, nil, e2e.Setup(), fx.Invoke(func(
		root context.Context,
		lc fx.Lifecycle,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
	) {
		lc.Append(fx.StartHook(func() {
			info := tests.AssertRequest(cl.GetInfoWithResponse(root))(t, http.StatusOK)
			require.False(t, info.JSON200.AiFeatures.SimilarThreads)

			memberCtx, _ := e2e.WithAccount(root, aw, seed.Account_003_Baldur)
			res := tests.AssertRequest(cl.ThreadSimilarWithResponse(root, openapi.ThreadTagSuggestionRequest{
				Title: opt.New("Kofferdam richtig spannen, Tipps gesucht").Ptr(),
			}, sh.WithSession(memberCtx)))(t, http.StatusOK)
			require.Empty(t, ids(t, res.JSON200))
		}))
	}))
}
