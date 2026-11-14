package tag_backfill_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func TestAdminTagBackfill(t *testing.T) {
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
			admin := sh.WithSession(adminCtx)
			member := sh.WithSession(memberCtx)

			cat, err := cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{Name: uuid.NewString(), Colour: "#000"}, admin)
			tests.Ok(t, err, cat)

			create := func(title string, tags ...string) *openapi.Thread {
				var list *openapi.TagNameList
				if len(tags) > 0 {
					list = (*openapi.TagNameList)(&tags)
				}
				res, err := cl.ThreadCreateWithResponse(root, openapi.ThreadInitialProps{
					Title:      title,
					Body:       opt.New("<p>" + title + " Erfahrungen?</p>").Ptr(),
					Category:   opt.New(cat.JSON200.Id).Ptr(),
					Tags:       list,
					Visibility: opt.New(openapi.VisibilityPublished).Ptr(),
				}, member)
				tests.Ok(t, err, res)
				return res.JSON200
			}

			create("Wurzelkanal Aufbereitung", "endo")
			create("Wurzelfüllung im Phantomkurs", "endo", "phantom")
			create("Wurzelspitzenresektion", "endo")
			untagged := create("Wurzelbehandlung Frage")
			mistagged := create("Wurzel und Zahnspange", "kfo")
			create("Mensa Öffnungszeiten")

			_, err = idx.Sync(root)
			require.NoError(t, err)

			t.Run("members_are_forbidden", func(t *testing.T) {
				res, err := cl.AdminTagBackfillStartWithResponse(root, member)
				tests.Status(t, err, res, http.StatusForbidden)
			})

			var proposals []openapi.TagBackfillProposal
			t.Run("suggests_only_missing_tags", func(t *testing.T) {
				tests.AssertRequest(cl.AdminTagBackfillStartWithResponse(root, admin))(t, http.StatusOK)

				var status *openapi.TagBackfillStatus
				require.Eventually(t, func() bool {
					res := tests.AssertRequest(cl.AdminTagBackfillStatusWithResponse(root, admin))(t, http.StatusOK)
					status = res.JSON200
					return !status.Running
				}, 30*time.Second, 50*time.Millisecond)

				require.Equal(t, 6, status.Total)
				require.Equal(t, 6, status.Checked)
				require.Zero(t, status.Failed)

				byID := map[string]openapi.TagBackfillProposal{}
				for _, p := range status.Proposals {
					byID[p.ThreadId] = p
				}
				require.Len(t, byID, 2)
				require.Equal(t, []string{"endo"}, byID[untagged.Id].Add)
				require.Empty(t, byID[untagged.Id].Current)
				require.Equal(t, []string{"endo"}, byID[mistagged.Id].Add)
				require.Equal(t, []string{"kfo"}, byID[mistagged.Id].Current)
				proposals = status.Proposals
			})

			t.Run("apply_adds_existing_tags_only", func(t *testing.T) {
				before := tests.AssertRequest(cl.ThreadGetWithResponse(root, untagged.Slug, nil, admin))(t, http.StatusOK)

				var items []openapi.TagBackfillApplyItem
				for _, p := range proposals {
					items = append(items, openapi.TagBackfillApplyItem{ThreadId: p.ThreadId, Tags: append(p.Add, "erfunden")})
				}

				res := tests.AssertRequest(cl.AdminTagBackfillApplyWithResponse(root, openapi.AdminTagBackfillApplyJSONRequestBody{Items: items}, admin))(t, http.StatusOK)
				require.Equal(t, 2, res.JSON200.Applied)
				require.Empty(t, res.JSON200.Proposals)

				after := tests.AssertRequest(cl.ThreadGetWithResponse(root, untagged.Slug, nil, admin))(t, http.StatusOK)
				require.Equal(t, []string{"endo"}, tagNames(after.JSON200.Tags))
				require.True(t, before.JSON200.UpdatedAt.Equal(after.JSON200.UpdatedAt))

				other := tests.AssertRequest(cl.ThreadGetWithResponse(root, mistagged.Slug, nil, admin))(t, http.StatusOK)
				require.ElementsMatch(t, []string{"kfo", "endo"}, tagNames(other.JSON200.Tags))

				tag, err := cl.TagGetWithResponse(root, "erfunden", admin)
				tests.Status(t, err, tag, http.StatusNotFound)
			})

			t.Run("apply_twice_adds_nothing", func(t *testing.T) {
				res := tests.AssertRequest(cl.AdminTagBackfillApplyWithResponse(root, openapi.AdminTagBackfillApplyJSONRequestBody{Items: []openapi.TagBackfillApplyItem{
					{ThreadId: untagged.Id, Tags: []string{"endo"}},
				}}, admin))(t, http.StatusOK)
				require.Zero(t, res.JSON200.Applied)
			})
		}))
	}))
}

func tagNames(tags openapi.TagReferenceList) []string {
	out := make([]string, len(tags))
	for i, t := range tags {
		out[i] = t.Name
	}
	return out
}
