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
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/config"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

// The fake model always picks anatomie and endo for posts about nerves, while
// the fake embeddings only place anatomie near such posts, so only the tag
// both agree on may be returned.
func fakeModel(t *testing.T) *httptest.Server {
	axis := func(text string) []float32 {
		lower := strings.ToLower(text)
		switch {
		case strings.HasPrefix(lower, "anatomie:"), strings.Contains(lower, "hirnnerven"):
			return []float32{1, 0, 0}
		case strings.HasPrefix(lower, "endo:"):
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
				"message": map[string]any{"role": "assistant", "content": `{"tags":["anatomie","endo","ghost"]}`},
			}},
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestThreadSuggestTags(t *testing.T) {
	t.Parallel()

	srv := fakeModel(t)

	integration.Test(t, &config.Config{
		LanguageModelProvider:       "openai",
		OpenAIBaseURL:               srv.URL + "/v1",
		LanguageModelChatModel:      "m",
		LanguageModelEmbeddingModel: "e",
	}, e2e.Setup(), fx.Invoke(func(
		root context.Context,
		lc fx.Lifecycle,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
	) {
		lc.Append(fx.StartHook(func() {
			adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			admin := sh.WithSession(adminCtx)
			memberCtx, _ := e2e.WithAccount(root, aw, seed.Account_003_Baldur)
			member := sh.WithSession(memberCtx)

			cat, err := cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{Name: uuid.NewString(), Colour: "#000"}, admin)
			tests.Ok(t, err, cat)
			th, err := cl.ThreadCreateWithResponse(root, openapi.ThreadInitialProps{
				Title:      "seed tags",
				Body:       opt.New("<p>seed</p>").Ptr(),
				Category:   opt.New(cat.JSON200.Id).Ptr(),
				Tags:       &openapi.TagNameList{"anatomie", "endo", "kfo"},
				Visibility: opt.New(openapi.VisibilityPublished).Ptr(),
			}, member)
			tests.Ok(t, err, th)

			setMeanings := func(t *testing.T, meanings string) {
				res, err := cl.AdminSettingsUpdateWithResponse(root, openapi.AdminSettingsUpdateJSONRequestBody{
					Metadata: &openapi.Metadata{"ai": map[string]any{"tagMeanings": meanings}},
				}, admin)
				tests.Ok(t, err, res)
			}

			suggest := func(t *testing.T, title, body string) []string {
				res, err := cl.ThreadSuggestTagsWithResponse(root, openapi.ThreadTagSuggestionRequest{
					Title: opt.New(title).Ptr(),
					Body:  opt.New(body).Ptr(),
				}, member)
				tests.Ok(t, err, res)
				return res.JSON200.Tags
			}

			t.Run("no_meanings_no_suggestions", func(t *testing.T) {
				require.Empty(t, suggest(t, "Hirnnerven lernen", "<p>Eselsbrücke für die Hirnnerven gesucht</p>"))
			})

			setMeanings(t, "anatomie: anatomy, nerves\nENDO: endodontics, root canals\nghost: not a real tag\nkaputt ohne doppelpunkt")

			t.Run("only_tags_both_agree_on", func(t *testing.T) {
				require.Equal(t, []string{"anatomie"}, suggest(t, "Hirnnerven lernen", "<p>Eselsbrücke für die Hirnnerven gesucht</p>"))
			})

			t.Run("short_text_is_skipped", func(t *testing.T) {
				require.Empty(t, suggest(t, "Hirnnerven", ""))
			})

			t.Run("requires_authentication", func(t *testing.T) {
				res, err := cl.ThreadSuggestTagsWithResponse(root, openapi.ThreadTagSuggestionRequest{
					Title: opt.New("Hirnnerven lernen für das Physikum").Ptr(),
				})
				tests.Status(t, err, res, http.StatusUnauthorized)
			})
		}))
	}))
}

func TestThreadSuggestTagsDisabled(t *testing.T) {
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
			res, err := cl.ThreadSuggestTagsWithResponse(root, openapi.ThreadTagSuggestionRequest{
				Title: opt.New("Hirnnerven lernen für das Physikum").Ptr(),
			}, sh.WithSession(memberCtx))
			tests.Ok(t, err, res)
			require.Empty(t, res.JSON200.Tags)
		}))
	}))
}
