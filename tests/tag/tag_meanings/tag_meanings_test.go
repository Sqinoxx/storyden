package tag_meanings_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Southclaws/opt"
	"github.com/rs/xid"
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

func fakeModel(t *testing.T) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		prompt := body.Messages[len(body.Messages)-1].Content

		answer := `{"meaning":"anatomy\nand nerves","confident":true}`
		if strings.Contains(prompt, `tagged "unklar"`) {
			answer = `{"meaning":"something","confident":false}`
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "x", "object": "chat.completion", "created": 1, "model": "m",
			"choices": []map[string]any{{
				"index": 0, "finish_reason": "stop",
				"message": map[string]any{"role": "assistant", "content": answer},
			}},
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestAdminGenerateTagMeanings(t *testing.T) {
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

			cat := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{Name: xid.New().String(), Colour: "#000"}, admin))(t, http.StatusOK)

			thread := func(title string, tags ...string) {
				tests.AssertRequest(cl.ThreadCreateWithResponse(root, openapi.ThreadInitialProps{
					Title:      title,
					Body:       opt.New("<p>" + title + "</p>").Ptr(),
					Category:   opt.New(cat.JSON200.Id).Ptr(),
					Tags:       &tags,
					Visibility: opt.New(openapi.VisibilityPublished).Ptr(),
				}, member))(t, http.StatusOK)
			}

			thread("Hirnnerven Eselsbrücke", "anatomie", "endo", "unklar")
			thread("Muskeln des Kopfes", "anatomie", "endo", "unklar")
			thread("Testat Schädel", "anatomie", "endo", "unklar", "selten")

			tests.AssertRequest(cl.AdminSettingsUpdateWithResponse(root, openapi.AdminSettingsUpdateJSONRequestBody{
				Metadata: &openapi.Metadata{"ai": map[string]any{"tagMeanings": "ENDO: endodontics, root canals\n"}},
			}, admin))(t, http.StatusOK)

			t.Run("drafts_missing_meanings", func(t *testing.T) {
				res := tests.AssertRequest(cl.AdminGenerateTagMeaningsWithResponse(root, admin))(t, http.StatusOK)
				require.Equal(t, []string{"anatomie"}, res.JSON200.Added)
				require.Equal(t, []string{"unklar"}, res.JSON200.Skipped)
				require.Equal(t, "ENDO: endodontics, root canals\nanatomie: anatomy and nerves", res.JSON200.Meanings)
			})

			t.Run("nothing_is_saved", func(t *testing.T) {
				res := tests.AssertRequest(cl.AdminSettingsGetWithResponse(root, admin))(t, http.StatusOK)
				ai := (*res.JSON200.Metadata)["ai"].(map[string]any)
				require.Equal(t, "ENDO: endodontics, root canals\n", ai["tagMeanings"])
			})

			t.Run("members_are_forbidden", func(t *testing.T) {
				res, err := cl.AdminGenerateTagMeaningsWithResponse(root, member)
				tests.Status(t, err, res, http.StatusForbidden)
			})
		}))
	}))
}

func TestAdminGenerateTagMeaningsWithoutLanguageModel(t *testing.T) {
	t.Parallel()

	integration.Test(t, nil, e2e.Setup(), fx.Invoke(func(
		root context.Context,
		lc fx.Lifecycle,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
	) {
		lc.Append(fx.StartHook(func() {
			adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			res := tests.AssertRequest(cl.AdminGenerateTagMeaningsWithResponse(root, sh.WithSession(adminCtx)))(t, http.StatusOK)
			require.Empty(t, res.JSON200.Added)
			require.Empty(t, res.JSON200.Skipped)
		}))
	}))
}
