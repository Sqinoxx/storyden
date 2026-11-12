package thread_summary_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/Southclaws/opt"
	"github.com/rs/xid"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/asset/asset_writer"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/config"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

type recorder struct {
	mu      sync.Mutex
	prompts []string
}

func (r *recorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.prompts)
}

func (r *recorder) last() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.prompts[len(r.prompts)-1]
}

func fakeModel(t *testing.T) (*httptest.Server, *recorder) {
	rec := &recorder{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))

		rec.mu.Lock()
		rec.prompts = append(rec.prompts, body.Messages[len(body.Messages)-1].Content)
		rec.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "x", "object": "chat.completion", "created": 1, "model": "m",
			"choices": []map[string]any{{
				"index": 0, "finish_reason": "stop",
				"message": map[string]any{"role": "assistant", "content": "```html\n<p><strong>Worum geht's</strong></p><ul><li>Hirnnerven</li></ul><script>alert(1)</script>\n```"},
			}},
		})
	}))
	t.Cleanup(srv.Close)
	return srv, rec
}

func TestThreadSummarise(t *testing.T) {
	t.Parallel()

	srv, rec := fakeModel(t)

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
		assets *asset_writer.Writer,
	) {
		lc.Append(fx.StartHook(func() {
			adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			admin := sh.WithSession(adminCtx)
			memberCtx, _ := e2e.WithAccount(root, aw, seed.Account_003_Baldur)
			member := sh.WithSession(memberCtx)
			otherCtx, _ := e2e.WithAccount(root, aw, seed.Account_004_Loki)
			other := sh.WithSession(otherCtx)

			cat := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{Name: xid.New().String(), Colour: "#000"}, admin))(t, http.StatusOK)

			createThread := func(t *testing.T, replies int, vis openapi.Visibility) string {
				th := tests.AssertRequest(cl.ThreadCreateWithResponse(root, openapi.ThreadInitialProps{
					Title:      "Hirnnerven Eselsbrücke",
					Category:   opt.New(cat.JSON200.Id).Ptr(),
					Body:       opt.New("<p>Kennt jemand eine Eselsbrücke für die Hirnnerven?</p>").Ptr(),
					Visibility: opt.New(vis).Ptr(),
				}, member))(t, http.StatusOK)
				for i := range replies {
					tests.AssertRequest(cl.ReplyCreateWithResponse(root, th.JSON200.Slug, openapi.ReplyInitialProps{
						Body: fmt.Sprintf("<p>Antwort Nummer %d zu den Hirnnerven</p>", i+1),
					}, member))(t, http.StatusOK)
				}
				return th.JSON200.Slug
			}

			summarise := func(t *testing.T, slug string) *openapi.ThreadSummary {
				res := tests.AssertRequest(cl.ThreadSummariseWithResponse(root, slug, member))(t, http.StatusOK)
				return res.JSON200
			}

			long := createThread(t, 5, openapi.VisibilityPublished)

			t.Run("disabled_by_default", func(t *testing.T) {
				res, err := cl.ThreadSummariseWithResponse(root, long, member)
				tests.Status(t, err, res, http.StatusForbidden)

				info := tests.AssertRequest(cl.GetInfoWithResponse(root))(t, http.StatusOK)
				require.False(t, info.JSON200.AiFeatures.ThreadSummary)
				require.True(t, info.JSON200.AiFeatures.TagSuggestion)
			})

			tests.AssertRequest(cl.AdminSettingsUpdateWithResponse(root, openapi.AdminSettingsUpdateJSONRequestBody{
				Services: &openapi.AdminSettingsServiceProps{
					Ai: &openapi.AIServiceSettings{ThreadSummary: opt.New(true).Ptr()},
				},
			}, admin))(t, http.StatusOK)

			t.Run("enabled_flag_in_info", func(t *testing.T) {
				info := tests.AssertRequest(cl.GetInfoWithResponse(root))(t, http.StatusOK)
				require.True(t, info.JSON200.AiFeatures.ThreadSummary)
			})

			t.Run("short_thread_is_not_summarised", func(t *testing.T) {
				before := rec.count()
				res := summarise(t, createThread(t, 2, openapi.VisibilityPublished))
				require.Empty(t, res.Summary)
				require.Equal(t, before, rec.count())
			})

			t.Run("long_thread_is_summarised_with_attachments", func(t *testing.T) {
				data := []byte("%PDF-1.4\n%%EOF")
				upload := tests.AssertRequest(cl.AssetUploadWithBodyWithResponse(
					root,
					&openapi.AssetUploadParams{ContentLength: int64(len(data)), Filename: opt.New("altklausur.pdf").Ptr()},
					"application/octet-stream",
					bytes.NewReader(data),
					member,
				))(t, http.StatusOK)
				id, err := xid.FromString(upload.JSON200.Id)
				require.NoError(t, err)
				_, err = assets.UpdateOCRCompleted(root, id, "Frage 3: Welcher Hirnnerv innerviert den Musculus masseter?")
				require.NoError(t, err)

				tests.AssertRequest(cl.ReplyCreateWithResponse(root, long, openapi.ReplyInitialProps{
					Body:     "<p>Hier die Altklausur dazu</p>",
					AssetIds: &[]string{upload.JSON200.Id},
				}, member))(t, http.StatusOK)

				before := rec.count()
				res := summarise(t, long)
				require.Equal(t, before+1, rec.count())
				require.Contains(t, res.Summary, "Hirnnerven")
				require.NotContains(t, res.Summary, "<script")
				require.NotContains(t, res.Summary, "```")
				require.True(t, res.IncludesAttachments)

				prompt := rec.last()
				require.Contains(t, prompt, "Kennt jemand eine Eselsbrücke")
				require.Contains(t, prompt, "Antwort Nummer 5")
				require.Contains(t, prompt, "[altklausur")
				require.Contains(t, prompt, "Musculus masseter")
			})

			t.Run("unchanged_thread_is_cached", func(t *testing.T) {
				before := rec.count()
				summarise(t, long)
				require.Equal(t, before, rec.count())
			})

			t.Run("new_reply_invalidates_cache", func(t *testing.T) {
				tests.AssertRequest(cl.ReplyCreateWithResponse(root, long, openapi.ReplyInitialProps{
					Body: "<p>Noch eine Antwort</p>",
				}, member))(t, http.StatusOK)
				before := rec.count()
				summarise(t, long)
				require.Equal(t, before+1, rec.count())
			})

			t.Run("draft_of_another_member_is_hidden", func(t *testing.T) {
				draft := createThread(t, 5, openapi.VisibilityDraft)
				before := rec.count()
				res, err := cl.ThreadSummariseWithResponse(root, draft, other)
				require.NoError(t, err)
				require.GreaterOrEqual(t, res.StatusCode(), 400)
				require.Equal(t, before, rec.count())
			})
		}))
	}))
}

func TestAIFeatureSwitches(t *testing.T) {
	t.Parallel()

	srv, _ := fakeModel(t)

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

			semester := func(t *testing.T) *string {
				res := tests.AssertRequest(cl.ThreadSuggestSemesterWithResponse(root, openapi.ThreadSemesterSuggestionRequest{
					Filenames: &[]string{"Anatomie_WS22-23.pdf"},
				}, member))(t, http.StatusOK)
				return res.JSON200.Term
			}

			require.NotNil(t, semester(t))

			tests.AssertRequest(cl.AdminSettingsUpdateWithResponse(root, openapi.AdminSettingsUpdateJSONRequestBody{
				Services: &openapi.AdminSettingsServiceProps{
					Ai: &openapi.AIServiceSettings{
						SemesterSuggestion: opt.New(false).Ptr(),
						TagSuggestion:      opt.New(false).Ptr(),
						LibraryAssist:      opt.New(false).Ptr(),
					},
				},
			}, admin))(t, http.StatusOK)

			t.Run("settings_round_trip", func(t *testing.T) {
				res := tests.AssertRequest(cl.AdminSettingsGetWithResponse(root, admin))(t, http.StatusOK)
				ai := res.JSON200.Services.Ai
				require.NotNil(t, ai)
				require.False(t, *ai.SemesterSuggestion)
				require.False(t, *ai.TagSuggestion)
				require.False(t, *ai.LibraryAssist)
				require.Nil(t, ai.ThreadSummary)
			})

			t.Run("info_flags_off", func(t *testing.T) {
				info := tests.AssertRequest(cl.GetInfoWithResponse(root))(t, http.StatusOK)
				flags := info.JSON200.AiFeatures
				require.False(t, flags.SemesterSuggestion)
				require.False(t, flags.TagSuggestion)
				require.False(t, flags.LibraryAssist)
				require.False(t, flags.ThreadSummary)
			})

			t.Run("semester_suggestion_off", func(t *testing.T) {
				require.Nil(t, semester(t))
			})

			t.Run("tag_suggestion_off", func(t *testing.T) {
				res := tests.AssertRequest(cl.ThreadSuggestTagsWithResponse(root, openapi.ThreadTagSuggestionRequest{
					Title: opt.New("Hirnnerven lernen für das Physikum").Ptr(),
				}, member))(t, http.StatusOK)
				require.Empty(t, res.JSON200.Tags)
			})

			t.Run("library_assist_off", func(t *testing.T) {
				res, err := cl.NodeGenerateTitleWithResponse(root, "any", openapi.NodeGenerateTitleRequest{Content: "<p>Hirnnerven</p>"}, admin)
				tests.Status(t, err, res, http.StatusForbidden)
			})
		}))
	}))
}

func TestAIFeatureFlagsWithoutLanguageModel(t *testing.T) {
	t.Parallel()

	integration.Test(t, nil, e2e.Setup(), fx.Invoke(func(
		root context.Context,
		lc fx.Lifecycle,
		cl *openapi.ClientWithResponses,
	) {
		lc.Append(fx.StartHook(func() {
			info := tests.AssertRequest(cl.GetInfoWithResponse(root))(t, http.StatusOK)
			flags := info.JSON200.AiFeatures
			require.True(t, flags.SemesterSuggestion)
			require.False(t, flags.TagSuggestion)
			require.False(t, flags.LibraryAssist)
			require.False(t, flags.ThreadSummary)
		}))
	}))
}
