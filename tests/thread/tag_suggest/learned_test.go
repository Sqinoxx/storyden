package tag_suggest_test

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

// Embeddings put root canal texts and braces texts on separate axes. The chat
// model never picks anything, so every suggestion must come from the tags of
// similar threads.
func topicModel(t *testing.T) *httptest.Server {
	axis := func(text string) []float32 {
		lower := strings.ToLower(text)
		switch {
		case strings.Contains(lower, "wurzel"):
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

		if strings.HasSuffix(r.URL.Path, "/embeddings") {
			input, _ := body["input"].(string)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"object": "list",
				"data":   []map[string]any{{"object": "embedding", "index": 0, "embedding": axis(input)}},
			})
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "x", "object": "chat.completion", "created": 1, "model": "m",
			"choices": []map[string]any{{
				"index": 0, "finish_reason": "stop",
				"message": map[string]any{"role": "assistant", "content": `{"tags":[]}`},
			}},
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestThreadSuggestTagsLearnedFromSimilarThreads(t *testing.T) {
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

			create := func(title string, vis openapi.Visibility, tags ...string) {
				res, err := cl.ThreadCreateWithResponse(root, openapi.ThreadInitialProps{
					Title:      title,
					Body:       opt.New("<p>" + title + " Erfahrungen?</p>").Ptr(),
					Category:   opt.New(cat.JSON200.Id).Ptr(),
					Tags:       (*openapi.TagNameList)(&tags),
					Visibility: opt.New(vis).Ptr(),
				}, member)
				tests.Ok(t, err, res)
			}

			create("Wurzelkanal Aufbereitung", openapi.VisibilityPublished, "endo")
			create("Wurzelfüllung im Phantomkurs", openapi.VisibilityPublished, "endo", "phantom")
			create("Wurzelspitzenresektion", openapi.VisibilityPublished, "endo")
			create("Zahnspange mit Brackets", openapi.VisibilityPublished, "kfo")
			create("Lose Zahnspange", openapi.VisibilityDraft, "kfo", "geheim")

			_, err = idx.Sync(root)
			require.NoError(t, err)

			suggest := func(t *testing.T, title, body string) []string {
				res, err := cl.ThreadSuggestTagsWithResponse(root, openapi.ThreadTagSuggestionRequest{
					Title: opt.New(title).Ptr(),
					Body:  opt.New(body).Ptr(),
				}, member)
				tests.Ok(t, err, res)
				return res.JSON200.Tags
			}

			t.Run("tags_shared_by_similar_threads", func(t *testing.T) {
				require.Equal(t, []string{"endo"}, suggest(t, "Frage zur Wurzelbehandlung", "<p>Wie lange dauert das?</p>"))
			})

			t.Run("one_similar_thread_is_not_enough_and_drafts_do_not_count", func(t *testing.T) {
				require.Empty(t, suggest(t, "Zahnspange tragen", "<p>Wie gewöhnt man sich daran?</p>"))
			})

			t.Run("unrelated_post", func(t *testing.T) {
				require.Empty(t, suggest(t, "Mensa Öffnungszeiten", "<p>Hat jemand den aktuellen Plan?</p>"))
			})
		}))
	}))
}
