package pdf_summary_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Southclaws/opt"
	"github.com/rs/xid"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/asset/asset_writer"
	"github.com/Southclaws/storyden/app/resources/message"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/services/pdf_summary"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/config"
	"github.com/Southclaws/storyden/internal/infrastructure/pubsub"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

type call struct {
	model     string
	maxTokens int
	stream    bool
	prompt    string
}

type recorder struct {
	mu    sync.Mutex
	calls []call
}

func (r *recorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.calls)
}

func (r *recorder) last() call {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls[len(r.calls)-1]
}

func fakeModel(t *testing.T) (*httptest.Server, *recorder) {
	rec := &recorder{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Model     string `json:"model"`
			MaxTokens int    `json:"max_tokens"`
			Stream    bool   `json:"stream"`
			Messages  []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))

		rec.mu.Lock()
		rec.calls = append(rec.calls, call{body.Model, body.MaxTokens, body.Stream, body.Messages[len(body.Messages)-1].Content})
		rec.mu.Unlock()

		if body.Stream {
			w.Header().Set("Content-Type", "text/event-stream")
			for _, part := range []string{"<ul><li>Altklausur ", "Endodontie</li>", "<li>Wurzelkanal</li></ul><script>x</script>"} {
				chunk, _ := json.Marshal(map[string]any{
					"id": "x", "object": "chat.completion.chunk", "created": 1, "model": body.Model,
					"choices": []map[string]any{{"index": 0, "delta": map[string]any{"content": part}}},
				})
				fmt.Fprintf(w, "data: %s\n\n", chunk)
				w.(http.Flusher).Flush()
			}
			fmt.Fprint(w, "data: [DONE]\n\n")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "x", "object": "chat.completion", "created": 1, "model": body.Model,
			"choices": []map[string]any{{
				"index": 0, "finish_reason": "stop",
				"message": map[string]any{"role": "assistant", "content": "<p>Altklausur Endodontie, ausführlich.</p><ul><li>Wurzelkanal</li></ul>"},
			}},
		})
	}))
	t.Cleanup(srv.Close)
	return srv, rec
}

func modelConfig(srv *httptest.Server) *config.Config {
	return &config.Config{
		LanguageModelProvider:       "openai",
		OpenAIBaseURL:               srv.URL + "/v1",
		LanguageModelChatModel:      "big",
		LanguageModelFastModel:      "fast",
		LanguageModelEmbeddingModel: "e",
	}
}

var examText = strings.Repeat("Aufgabe: Endodontie\nBeschreiben Sie die Aufbereitung des Wurzelkanals mit rotierenden Instrumenten und nennen Sie typische Komplikationen bei der Spülung.\n", 4) +
	"Altklausur Zahnerhaltung WS 22/23, Bearbeitungszeit 90 Minuten, Hilfsmittel sind nicht erlaubt.\n"

type env struct {
	root   context.Context
	cl     *openapi.ClientWithResponses
	admin  openapi.RequestEditorFn
	member openapi.RequestEditorFn
	other  openapi.RequestEditorFn
	assets *asset_writer.Writer
	cat    string
}

func (e env) upload(t *testing.T, name, text string) string {
	data := []byte("%PDF-1.4\n%%EOF")
	up := tests.AssertRequest(e.cl.AssetUploadWithBodyWithResponse(
		e.root,
		&openapi.AssetUploadParams{ContentLength: int64(len(data)), Filename: opt.New(name).Ptr()},
		"application/octet-stream",
		bytes.NewReader(data),
		e.member,
	))(t, http.StatusOK)
	id, err := xid.FromString(up.JSON200.Id)
	require.NoError(t, err)
	if text != "" {
		_, err = e.assets.UpdateOCRCompleted(e.root, id, text)
		require.NoError(t, err)
	}
	return up.JSON200.Id
}

func (e env) thread(t *testing.T, vis openapi.Visibility, assetIDs ...string) string {
	th := tests.AssertRequest(e.cl.ThreadCreateWithResponse(e.root, openapi.ThreadInitialProps{
		Title:      "Altklausur Endo " + xid.New().String(),
		Category:   &e.cat,
		Body:       opt.New("<p>Hier die Altklausur</p>").Ptr(),
		Visibility: opt.New(vis).Ptr(),
	}, e.member))(t, http.StatusOK)
	if len(assetIDs) > 0 {
		tests.AssertRequest(e.cl.ReplyCreateWithResponse(e.root, th.JSON200.Slug, openapi.ReplyInitialProps{
			Body:     "<p>Anhang</p>",
			AssetIds: &assetIDs,
		}, e.member))(t, http.StatusOK)
	}
	return th.JSON200.Slug
}

func (e env) enable(t *testing.T, ai openapi.AIServiceSettings) {
	tests.AssertRequest(e.cl.AdminSettingsUpdateWithResponse(e.root, openapi.AdminSettingsUpdateJSONRequestBody{
		Services: &openapi.AdminSettingsServiceProps{Ai: &ai},
	}, e.admin))(t, http.StatusOK)
}

func (e env) get(t *testing.T, slug, id string) *openapi.AttachmentSummary {
	return tests.AssertRequest(e.cl.ThreadAttachmentSummaryGetWithResponse(e.root, slug, id, e.member))(t, http.StatusOK).JSON200
}

func setup(t *testing.T, cfg *config.Config, run func(e env, bus *pubsub.Bus)) {
	integration.Test(t, cfg, e2e.Setup(), fx.Invoke(func(
		root context.Context,
		lc fx.Lifecycle,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
		assets *asset_writer.Writer,
		bus *pubsub.Bus,
		_ *pdf_summary.Worker,
	) {
		lc.Append(fx.StartHook(func() {
			adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			memberCtx, _ := e2e.WithAccount(root, aw, seed.Account_003_Baldur)
			otherCtx, _ := e2e.WithAccount(root, aw, seed.Account_004_Loki)
			admin := sh.WithSession(adminCtx)
			cat := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{Name: xid.New().String(), Colour: "#000"}, admin))(t, http.StatusOK)
			run(env{
				root:   root,
				cl:     cl,
				cat:    cat.JSON200.Id,
				admin:  admin,
				member: sh.WithSession(memberCtx),
				other:  sh.WithSession(otherCtx),
				assets: assets,
			}, bus)
		}))
	}))
}

func TestPDFSummaryLive(t *testing.T) {
	t.Parallel()

	srv, rec := fakeModel(t)

	setup(t, modelConfig(srv), func(e env, _ *pubsub.Bus) {
		pdf := e.upload(t, "altklausur-endo.pdf", examText)
		slug := e.thread(t, openapi.VisibilityPublished, pdf)

		t.Run("disabled_by_default", func(t *testing.T) {
			res, err := e.cl.ThreadAttachmentSummaryGetWithResponse(e.root, slug, pdf, e.member)
			tests.Status(t, err, res, http.StatusForbidden)

			info := tests.AssertRequest(e.cl.GetInfoWithResponse(e.root))(t, http.StatusOK)
			require.False(t, info.JSON200.AiFeatures.PdfSummary)
		})

		e.enable(t, openapi.AIServiceSettings{
			PdfSummary:           opt.New(true).Ptr(),
			PdfSummaryBackground: opt.New(false).Ptr(),
		})

		t.Run("settings_round_trip", func(t *testing.T) {
			info := tests.AssertRequest(e.cl.GetInfoWithResponse(e.root))(t, http.StatusOK)
			require.True(t, info.JSON200.AiFeatures.PdfSummary)

			res := tests.AssertRequest(e.cl.AdminSettingsGetWithResponse(e.root, e.admin))(t, http.StatusOK)
			require.True(t, *res.JSON200.Services.Ai.PdfSummary)
			require.False(t, *res.JSON200.Services.Ai.PdfSummaryBackground)
		})

		t.Run("preview_before_model", func(t *testing.T) {
			before := rec.count()
			res := e.get(t, slug, pdf)
			require.Equal(t, openapi.AttachmentSummaryStatusPending, res.Status)
			require.Nil(t, res.Summary)
			require.Contains(t, res.Preview, "Altklausur / Prüfung")
			require.Contains(t, res.Preview, "WS 22/23")
			require.Equal(t, before, rec.count())
		})

		t.Run("attachment_not_in_thread", func(t *testing.T) {
			stranger := e.upload(t, "fremd.pdf", examText)
			res, err := e.cl.ThreadAttachmentSummaryGetWithResponse(e.root, slug, stranger, e.member)
			tests.Status(t, err, res, http.StatusNotFound)

			stream, err := e.cl.ThreadAttachmentSummaryStreamWithResponse(e.root, slug, stranger, e.member)
			tests.Status(t, err, stream, http.StatusNotFound)
		})

		t.Run("no_text_is_unavailable", func(t *testing.T) {
			scan := e.upload(t, "scan.pdf", "")
			other := e.thread(t, openapi.VisibilityPublished, scan)
			res := e.get(t, other, scan)
			require.Equal(t, openapi.AttachmentSummaryStatusUnavailable, res.Status)
		})

		t.Run("draft_of_another_member_is_hidden", func(t *testing.T) {
			draft := e.thread(t, openapi.VisibilityDraft, pdf)
			res, err := e.cl.ThreadAttachmentSummaryGetWithResponse(e.root, draft, pdf, e.other)
			require.NoError(t, err)
			require.GreaterOrEqual(t, res.StatusCode(), 400)
		})

		t.Run("stream_uses_fast_model_and_stores_result", func(t *testing.T) {
			before := rec.count()
			res, err := e.cl.ThreadAttachmentSummaryStream(e.root, slug, pdf, e.member)
			require.NoError(t, err)
			defer res.Body.Close()
			require.Equal(t, http.StatusOK, res.StatusCode)
			require.Contains(t, res.Header.Get("Content-Type"), "text/event-stream")

			body, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			out := string(body)
			require.Contains(t, out, `data: {"t":`)
			require.Contains(t, out, "Altklausur ")
			_, done, ok := strings.Cut(out, "event: done\n")
			require.True(t, ok)
			require.Contains(t, done, "Wurzelkanal")
			require.NotContains(t, done, "script")

			require.Equal(t, before+1, rec.count())
			c := rec.last()
			require.True(t, c.stream)
			require.Equal(t, "fast", c.model)
			require.Equal(t, 180, c.maxTokens)
			require.Contains(t, c.prompt, "altklausur-endo")
			require.Contains(t, c.prompt, "Wurzelkanals")
		})

		t.Run("stored_summary_is_returned_without_model", func(t *testing.T) {
			before := rec.count()
			res := e.get(t, slug, pdf)
			require.Equal(t, openapi.AttachmentSummaryStatusReady, res.Status)
			require.NotNil(t, res.Summary)
			require.Contains(t, *res.Summary, "Endodontie")
			require.Equal(t, openapi.AttachmentSummaryQuality("quick"), *res.Quality)

			stream, err := e.cl.ThreadAttachmentSummaryStream(e.root, slug, pdf, e.member)
			require.NoError(t, err)
			body, _ := io.ReadAll(stream.Body)
			stream.Body.Close()
			require.Contains(t, string(body), "event: done")
			require.Equal(t, before, rec.count())
		})

		t.Run("changed_text_invalidates", func(t *testing.T) {
			id, _ := xid.FromString(pdf)
			_, err := e.assets.UpdateOCRCompleted(e.root, id, examText+"\nNachtrag: Lösungsskizze zu Aufgabe 2 mit Kofferdam und Längenbestimmung.")
			require.NoError(t, err)
			res := e.get(t, slug, pdf)
			require.Equal(t, openapi.AttachmentSummaryStatusPending, res.Status)
		})

	})
}

func TestPDFSummaryBackground(t *testing.T) {
	t.Parallel()

	srv, rec := fakeModel(t)

	setup(t, modelConfig(srv), func(e env, bus *pubsub.Bus) {
		e.enable(t, openapi.AIServiceSettings{PdfSummary: opt.New(true).Ptr()})

		pdf := e.upload(t, "skript-paro.pdf", examText)
		slug := e.thread(t, openapi.VisibilityPublished, pdf)

		id, _ := xid.FromString(pdf)
		bus.Publish(e.root, &message.EventAssetOCRCompleted{AssetID: id})

		require.Eventually(t, func() bool {
			res := e.get(t, slug, pdf)
			return res.Status == openapi.AttachmentSummaryStatusReady && res.Quality != nil && *res.Quality == openapi.AttachmentSummaryQuality("full")
		}, 20*time.Second, 100*time.Millisecond)

		c := rec.last()
		require.False(t, c.stream)
		require.Equal(t, "big", c.model)
		require.Equal(t, 300, c.maxTokens)

		stats := tests.AssertRequest(e.cl.AdminOCRStatsWithResponse(e.root, e.admin))(t, http.StatusOK)
		require.GreaterOrEqual(t, *stats.JSON200.PdfSummariesDone, 1)
		require.GreaterOrEqual(t, *stats.JSON200.PdfSummariesTotal, *stats.JSON200.PdfSummariesDone)
	})
}
