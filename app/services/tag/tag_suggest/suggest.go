// Small models guess wrongly at abbreviated tag names and pad their answer to
// the maximum count, so a tag is only suggested when it has an admin-written
// meaning, the model picks it and it is also among the embedding-nearest tags.
package tag_suggest

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/settings"
	"github.com/Southclaws/storyden/app/resources/tag/tag_querier"
	"github.com/Southclaws/storyden/internal/infrastructure/ai"
)

const (
	maxTags        = 3
	nearestTags    = 3
	minSimilarity  = 0.40
	minInputChars  = 20
	maxInputChars  = 4000
	requestTimeout = 20 * time.Second
)

type Suggester struct {
	logger   *slog.Logger
	prompter ai.Prompter
	settings *settings.SettingsRepository
	tags     *tag_querier.Querier

	mu      sync.Mutex
	vectors map[string][]float32
}

func Build() fx.Option {
	return fx.Provide(New)
}

func New(logger *slog.Logger, prompter ai.Prompter, settings *settings.SettingsRepository, tags *tag_querier.Querier) *Suggester {
	return &Suggester{
		logger:   logger,
		prompter: prompter,
		settings: settings,
		tags:     tags,
		vectors:  map[string][]float32{},
	}
}

func (s *Suggester) Suggest(ctx context.Context, title, body string) ([]string, error) {
	if _, ok := s.prompter.(*ai.OpenAI); !ok {
		return nil, nil
	}

	title, body = strings.TrimSpace(title), strings.TrimSpace(body)
	if len([]rune(title+body)) < minInputChars {
		return nil, nil
	}
	if len(body) > maxInputChars {
		body = body[:maxInputChars]
	}

	meanings, err := s.meanings(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}
	if len(meanings) == 0 {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	picked, err := s.pick(ctx, title, body, meanings)
	if err != nil {
		s.logger.Warn("tag suggestion model call failed", slog.String("error", err.Error()))
		return nil, nil
	}
	if len(picked) == 0 {
		return nil, nil
	}

	near, err := s.nearest(ctx, title+"\n"+body, meanings)
	if err != nil {
		s.logger.Warn("tag suggestion embedding failed", slog.String("error", err.Error()))
		return nil, nil
	}

	var out []string
	for _, t := range picked {
		if near[t] {
			out = append(out, t)
		}
	}

	return out, nil
}

type meaning struct {
	tag  string
	text string
}

func (s *Suggester) meanings(ctx context.Context) ([]meaning, error) {
	set, err := s.settings.Get(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	glossary := ParseMeanings(rawMeanings(set.Metadata.OrZero()))
	if len(glossary) == 0 {
		return nil, nil
	}

	existing, err := s.tags.List(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	var out []meaning
	for _, name := range existing.Names() {
		tag := name.String()
		if m, ok := glossary[strings.ToLower(tag)]; ok {
			out = append(out, meaning{tag: tag, text: m})
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].tag < out[j].tag })
	return out, nil
}

func rawMeanings(meta map[string]any) string {
	section, ok := meta["ai"].(map[string]any)
	if !ok {
		return ""
	}
	raw, _ := section["tagMeanings"].(string)
	return raw
}

// ParseMeanings reads one "tag: meaning" entry per line, keyed by lowercase tag.
func ParseMeanings(raw string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(raw, "\n") {
		tag, text, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		tag = strings.ToLower(strings.TrimSpace(tag))
		text = strings.TrimSpace(text)
		if tag == "" || text == "" {
			continue
		}
		out[tag] = text
	}
	return out
}

const pickTemplate = `Pick tags for a post in a German dentistry students' forum.

Allowed tags with their meaning:
%s

Post title: %s
Post: %s

Choose at most %d tags whose meaning clearly matches the subject of the post. Only choose a tag if it is obviously correct. If no tag clearly fits, return an empty list.`

func (s *Suggester) pick(ctx context.Context, title, body string, meanings []meaning) ([]string, error) {
	tags := make([]string, len(meanings))
	lines := make([]string, len(meanings))
	for i, m := range meanings {
		tags[i] = m.tag
		lines[i] = fmt.Sprintf("- %s: %s", m.tag, m.text)
	}

	schema := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"tags"},
		"properties": map[string]any{
			"tags": map[string]any{
				"type":     "array",
				"maxItems": maxTags,
				"items":    map[string]any{"type": "string", "enum": tags},
			},
		},
	}

	var res struct {
		Tags []string `json:"tags"`
	}
	prompt := fmt.Sprintf(pickTemplate, strings.Join(lines, "\n"), title, body, maxTags)
	if err := ai.PromptSchema(ctx, s.prompter, "Pick tags for a forum post", prompt, schema, &res); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	allowed := make(map[string]bool, len(tags))
	for _, t := range tags {
		allowed[t] = true
	}

	seen := map[string]bool{}
	var out []string
	for _, t := range res.Tags {
		if allowed[t] && !seen[t] && len(out) < maxTags {
			seen[t] = true
			out = append(out, t)
		}
	}
	return out, nil
}

func (s *Suggester) nearest(ctx context.Context, text string, meanings []meaning) (map[string]bool, error) {
	embed := s.prompter.EmbeddingFunc()

	post, err := embed(ctx, text)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	type scored struct {
		tag string
		sim float64
	}
	scores := make([]scored, 0, len(meanings))
	for _, m := range meanings {
		v, err := s.vector(ctx, embed, m.tag+": "+m.text)
		if err != nil {
			return nil, fault.Wrap(err, fctx.With(ctx))
		}
		scores = append(scores, scored{m.tag, cosine(post, v)})
	}

	sort.Slice(scores, func(i, j int) bool { return scores[i].sim > scores[j].sim })

	out := map[string]bool{}
	for i := 0; i < len(scores) && i < nearestTags; i++ {
		if scores[i].sim >= minSimilarity {
			out[scores[i].tag] = true
		}
	}
	return out, nil
}

func (s *Suggester) vector(ctx context.Context, embed func(context.Context, string) ([]float32, error), text string) ([]float32, error) {
	s.mu.Lock()
	v, ok := s.vectors[text]
	s.mu.Unlock()
	if ok {
		return v, nil
	}

	v, err := embed(ctx, text)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.vectors[text] = v
	s.mu.Unlock()
	return v, nil
}

func cosine(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
