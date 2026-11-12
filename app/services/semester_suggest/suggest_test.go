package semester_suggest

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Southclaws/storyden/app/services/account/semester"
	"github.com/Southclaws/storyden/internal/config"
	"github.com/Southclaws/storyden/internal/infrastructure/ai"
)

var now = time.Date(2026, time.November, 12, 12, 0, 0, 0, time.UTC)

func ws(y int) semester.Term { return semester.Term{Year: y, Winter: true} }
func ss(y int) semester.Term { return semester.Term{Year: y} }

func TestDetect(t *testing.T) {
	cases := []struct {
		text string
		want []semester.Term
	}{
		{"Altklausur WS 22/23 Anatomie", []semester.Term{ws(2022)}},
		{"Altklausur_WS22-23.pdf", []semester.Term{ws(2022)}},
		{"Protokoll_WS2223.pdf", []semester.Term{ws(2022)}},
		{"WiSe 2023/24 Physikum", []semester.Term{ws(2023)}},
		{"Wintersemester 2024/2025", []semester.Term{ws(2024)}},
		{"Wintersemester 2024", []semester.Term{ws(2024)}},
		{"Wintersemester 2021", nil},
		{"Testat SS24", []semester.Term{ss(2024)}},
		{"SoSe 2025 Kons", []semester.Term{ss(2025)}},
		{"Sommersemester 2019 Klausur", []semester.Term{ss(2019)}},
		{"Kurs im ss 23", []semester.Term{ss(2023)}},
		{"Altklausuren WS 22/23 und WS 22/23 nochmal", []semester.Term{ws(2022)}},
		{"WS 22/23 und SS 23", []semester.Term{ws(2022), ss(2023)}},

		{"WS 22/24", nil},
		{"Boss 22 Prothetik", nil},
		{"Class 2023", nil},
		{"Ich hatte 2023 Physikum", nil},
		{"SS 2005", nil},
		{"WS 2027/28", nil},
		{"Hirnnerven lernen", nil},
	}

	for _, c := range cases {
		t.Run(c.text, func(t *testing.T) {
			var got []semester.Term
			for _, m := range detect(c.text, now) {
				got = append(got, m.term)
			}
			assert.Equal(t, c.want, got)
		})
	}
}

func TestDetectIndirect(t *testing.T) {
	// now is 2026-11-12, inside WS 2026/27.
	cases := []struct {
		text string
		want []semester.Term
	}{
		{"im letzten Wintersemester", []semester.Term{ws(2025)}},
		{"letztes Semester", []semester.Term{ss(2026)}},
		{"letztes Sommersemester", []semester.Term{ss(2026)}},
		{"dieses Semester", []semester.Term{ws(2026)}},
		{"dieses Wintersemester", []semester.Term{ws(2026)}},
		{"nächstes Semester", nil},
		{"Anmeldeschluss 15.01.2027", nil},
		{"Abgabe 20.11.2026", nil},
		{"Prüfung im Dezember 2026", nil},
		{"Klausur vom 14.02.2024", []semester.Term{ws(2023)}},
		{"Klausur vom 14.2.24", []semester.Term{ws(2023)}},
		{"Testat am 03.05.2025", []semester.Term{ss(2025)}},
		{"Kons Klausur Februar 2025", []semester.Term{ws(2024)}},
		{"Testat im Sommer 2023", []semester.Term{ss(2023)}},
		{"Winter 2023/24", []semester.Term{ws(2023)}},

		{"dieses Sommersemester", nil},
		{"Winter 2023", nil},
		{"Ich bin im 5. Semester", nil},
		{"Ich hatte 2023 Physikum", nil},
		{"Datum 31.13.2024", nil},
	}

	for _, c := range cases {
		t.Run(c.text, func(t *testing.T) {
			var got []semester.Term
			for _, m := range detectIndirect(c.text, now) {
				got = append(got, m.term)
			}
			assert.Equal(t, c.want, got)
		})
	}
}

func TestAccepted(t *testing.T) {
	assert.True(t, accepted(confirmation{"exam", 90}))
	assert.True(t, accepted(confirmation{" EXAM ", 100}))
	assert.False(t, accepted(confirmation{"exam", 89}))
	assert.False(t, accepted(confirmation{"other", 100}))
	assert.False(t, accepted(confirmation{"", 0}))
}

func newFakeModel(t *testing.T, answer confirmation) (ai.Prompter, *int) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		content, _ := json.Marshal(answer)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "x", "object": "chat.completion", "created": 1, "model": "m",
			"choices": []map[string]any{{
				"index": 0, "finish_reason": "stop",
				"message": map[string]any{"role": "assistant", "content": string(content)},
			}},
		})
	}))
	t.Cleanup(srv.Close)

	p, err := ai.New(config.Config{LanguageModelProvider: "openai", OpenAIBaseURL: srv.URL + "/v1", LanguageModelChatModel: "m"})
	require.NoError(t, err)
	return p, &calls
}

func TestSuggest(t *testing.T) {
	ctx := context.Background()

	suggester := func(p ai.Prompter) *Suggester {
		s := New(slog.Default(), p)
		s.now = func() time.Time { return now }
		return s
	}

	t.Run("pattern_wins_without_model_call", func(t *testing.T) {
		p, calls := newFakeModel(t, confirmation{"exam", 100})
		got, err := suggester(p).Suggest(ctx, Input{Title: "Altklausur", Body: "vom 14.02.2024", Filenames: []string{"Anatomie_WS22-23.pdf"}})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, ws(2022), got.Term)
		assert.Equal(t, SourcePattern, got.Source)
		assert.Equal(t, "WS22-23", got.Evidence)
		assert.Zero(t, *calls)
	})

	t.Run("conflicting_patterns_give_nothing", func(t *testing.T) {
		p, calls := newFakeModel(t, confirmation{"exam", 100})
		got, err := suggester(p).Suggest(ctx, Input{Title: "WS 22/23 oder SS 23?"})
		require.NoError(t, err)
		assert.Nil(t, got)
		assert.Zero(t, *calls)
	})

	t.Run("confirmed_indirect_reference_is_computed_in_code", func(t *testing.T) {
		p, calls := newFakeModel(t, confirmation{"exam", 95})
		got, err := suggester(p).Suggest(ctx, Input{Title: "Kons", Body: "Die Klausur im letzten Wintersemester war hart"})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, ws(2025), got.Term)
		assert.Equal(t, SourceModel, got.Source)
		assert.Equal(t, "letzten Wintersemester", got.Evidence)
		assert.Equal(t, 1, *calls)
	})

	t.Run("rejected_indirect_reference_gives_nothing", func(t *testing.T) {
		p, calls := newFakeModel(t, confirmation{"other", 100})
		got, err := suggester(p).Suggest(ctx, Input{Body: "Mein Geburtstag ist am 12.06.2025"})
		require.NoError(t, err)
		assert.Nil(t, got)
		assert.Equal(t, 1, *calls)
	})

	t.Run("unsure_confirmation_gives_nothing", func(t *testing.T) {
		p, _ := newFakeModel(t, confirmation{"exam", 60})
		got, err := suggester(p).Suggest(ctx, Input{Body: "Klausur vom 14.02.2024"})
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("several_indirect_references_give_nothing", func(t *testing.T) {
		p, calls := newFakeModel(t, confirmation{"exam", 100})
		got, err := suggester(p).Suggest(ctx, Input{Body: "Klausur vom 14.02.2024, Nachklausur im Sommer 2024"})
		require.NoError(t, err)
		assert.Nil(t, got)
		assert.Zero(t, *calls)
	})

	t.Run("no_reference_skips_model", func(t *testing.T) {
		p, calls := newFakeModel(t, confirmation{"exam", 100})
		got, err := suggester(p).Suggest(ctx, Input{Body: "Tipps zum Lernen der Hirnnerven"})
		require.NoError(t, err)
		assert.Nil(t, got)
		assert.Zero(t, *calls)
	})

	t.Run("disabled_model_gives_nothing_for_indirect", func(t *testing.T) {
		p, err := ai.New(config.Config{})
		require.NoError(t, err)
		got, err := suggester(p).Suggest(ctx, Input{Body: "letztes Wintersemester"})
		require.NoError(t, err)
		assert.Nil(t, got)
	})
}
