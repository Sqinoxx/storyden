package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Southclaws/storyden/internal/config"
)

type fakeCompatServer struct {
	mu              sync.Mutex
	chatModels      []string
	embeddingModels []string
	chatAnswer      string
}

func (f *fakeCompatServer) handler(t *testing.T) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Model string `json:"model"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))

		f.mu.Lock()
		f.chatModels = append(f.chatModels, body.Model)
		f.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      "chatcmpl-1",
			"object":  "chat.completion",
			"created": 1,
			"model":   body.Model,
			"choices": []map[string]any{{
				"index":         0,
				"finish_reason": "stop",
				"message": map[string]any{
					"role":    "assistant",
					"content": f.chatAnswer,
				},
			}},
		})
	})

	mux.HandleFunc("/v1/embeddings", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Model string `json:"model"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))

		f.mu.Lock()
		f.embeddingModels = append(f.embeddingModels, body.Model)
		f.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "list",
			"model":  body.Model,
			"data": []map[string]any{{
				"object":    "embedding",
				"index":     0,
				"embedding": []float32{0.6, 0.8, 0},
			}},
		})
	})

	return mux
}

func newCompatPrompter(t *testing.T, f *fakeCompatServer) Prompter {
	srv := httptest.NewServer(f.handler(t))
	t.Cleanup(srv.Close)

	p, err := New(config.Config{
		LanguageModelProvider:       "openai",
		OpenAIBaseURL:               srv.URL + "/v1",
		LanguageModelChatModel:      "qwen2.5:3b",
		LanguageModelEmbeddingModel: "bge-m3",
	})
	require.NoError(t, err)
	return p
}

func TestOpenAICompatible(t *testing.T) {
	ctx := context.Background()

	t.Run("prompt_uses_base_url_and_model", func(t *testing.T) {
		r := require.New(t)
		f := &fakeCompatServer{chatAnswer: "Hallo"}
		p := newCompatPrompter(t, f)

		res, err := p.Prompt(ctx, "Sag hallo")
		r.NoError(err)
		r.Equal("Hallo", res.Answer)
		r.Equal([]string{"qwen2.5:3b"}, f.chatModels)
	})

	t.Run("prompt_object_parses_json", func(t *testing.T) {
		r := require.New(t)
		f := &fakeCompatServer{chatAnswer: `{"tags":["anatomie","physikum"]}`}
		p := newCompatPrompter(t, f)

		res, err := PromptObject(ctx, p, "Suggest tags", "Inhalt", SuggestTagsResultSchema{})
		r.NoError(err)
		r.Equal([]string{"anatomie", "physikum"}, res.Tags)
		r.Equal([]string{"qwen2.5:3b"}, f.chatModels)
	})

	t.Run("embedding_uses_configured_model", func(t *testing.T) {
		r := require.New(t)
		f := &fakeCompatServer{}
		p := newCompatPrompter(t, f)

		vec, err := p.EmbeddingFunc()(ctx, "Zahnmedizin")
		r.NoError(err)
		r.Len(vec, 3)
		r.Equal([]string{"bge-m3"}, f.embeddingModels)
	})
}
