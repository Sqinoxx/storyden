// Tags are suggested from two sources. Learned: the tags of the most similar
// already-tagged threads in the semdex vote, weighted by similarity, so the
// suggestions improve as members tag posts and need no maintenance or language
// model. Meanings: small models guess wrongly at abbreviated tag names and pad
// their answer, so a tag with an admin-written meaning is only suggested when
// the model picks it and it is also among the embedding-nearest tags.
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
	"github.com/rs/xid"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/datagraph"
	"github.com/Southclaws/storyden/app/resources/pagination"
	"github.com/Southclaws/storyden/app/resources/settings"
	"github.com/Southclaws/storyden/app/resources/tag/tag_querier"
	"github.com/Southclaws/storyden/app/services/search/searcher"
	"github.com/Southclaws/storyden/app/services/semdex"
	"github.com/Southclaws/storyden/internal/infrastructure/ai"
)

const (
	maxTags        = 3
	nearestTags    = 3
	minSimilarity  = 0.40
	minInputChars  = 20
	maxInputChars  = 4000
	requestTimeout = 20 * time.Second

	searchCandidates       = 20
	nearestThreads         = 5
	minNeighbourSimilarity = 0.65
	minVotes               = 2
	minVoteShare           = 0.6
)

type Suggester struct {
	logger   *slog.Logger
	prompter ai.Prompter
	settings *settings.SettingsRepository
	tags     *tag_querier.Querier
	semdex   semdex.Searcher

	mu      sync.Mutex
	vectors map[string][]float32
}

func Build() fx.Option {
	return fx.Provide(New)
}

func New(logger *slog.Logger, prompter ai.Prompter, settings *settings.SettingsRepository, tags *tag_querier.Querier, semdex semdex.Searcher) *Suggester {
	return &Suggester{
		logger:   logger,
		prompter: prompter,
		settings: settings,
		tags:     tags,
		semdex:   semdex,
		vectors:  map[string][]float32{},
	}
}

func (s *Suggester) Enabled() bool {
	return ai.IsOpenAI(s.prompter)
}

func (s *Suggester) Suggest(ctx context.Context, title, body string) ([]string, error) {
	return s.SuggestExcluding(ctx, title, body, xid.NilID())
}

// SuggestExcluding ignores the thread itself when an existing thread is
// re-tagged, otherwise its own tags would vote for themselves.
func (s *Suggester) SuggestExcluding(ctx context.Context, title, body string, exclude xid.ID) ([]string, error) {
	if !ai.IsOpenAI(s.prompter) {
		return nil, nil
	}

	title, body = strings.TrimSpace(title), strings.TrimSpace(body)
	if len([]rune(title+body)) < minInputChars {
		return nil, nil
	}
	if r := []rune(body); len(r) > maxInputChars {
		body = string(r[:maxInputChars])
	}

	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	fromMeanings, err := s.fromMeanings(ctx, title, body)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	learned, err := s.fromSimilarThreads(ctx, title+"\n\n"+body, exclude)
	if err != nil {
		s.logger.Warn("tag suggestion from similar threads failed", slog.String("error", err.Error()))
	}

	return merge(maxTags, fromMeanings, learned), nil
}

func (s *Suggester) fromMeanings(ctx context.Context, title, body string) ([]string, error) {
	meanings, err := s.meanings(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}
	if len(meanings) == 0 {
		return nil, nil
	}

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

func (s *Suggester) fromSimilarThreads(ctx context.Context, text string, exclude xid.ID) ([]string, error) {
	if s.semdex == nil {
		return nil, nil
	}
	if _, disabled := s.semdex.(*semdex.Disabled); disabled {
		return nil, nil
	}

	res, err := s.semdex.SearchRefs(ctx, text, pagination.NewPageParams(1, searchCandidates), searcher.Options{})
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	var neighbours []Neighbour
	var ids []xid.ID
	for _, r := range res.Items {
		if len(neighbours) == nearestThreads {
			break
		}
		if r.ID == exclude {
			continue
		}
		if r.Kind == datagraph.KindThread && r.Relevance >= minNeighbourSimilarity {
			ids = append(ids, r.ID)
			neighbours = append(neighbours, Neighbour{ID: r.ID, Similarity: r.Relevance})
		}
	}

	tags, err := s.tags.PublishedThreadTags(ctx, ids)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	for i := range neighbours {
		neighbours[i].Tags = tags[neighbours[i].ID]
	}

	return Vote(neighbours), nil
}

type Neighbour struct {
	ID         xid.ID
	Similarity float64
	Tags       []string
}

// Vote suggests the tags carried by at least minVotes of the tagged neighbours
// that together hold at least minVoteShare of their similarity weight.
func Vote(neighbours []Neighbour) []string {
	type tally struct {
		name  string
		score float64
		votes int
	}

	var total float64
	byKey := map[string]*tally{}
	for _, n := range neighbours {
		if len(n.Tags) == 0 {
			continue
		}
		total += n.Similarity

		seen := map[string]bool{}
		for _, name := range n.Tags {
			key := strings.ToLower(name)
			if seen[key] {
				continue
			}
			seen[key] = true

			t, ok := byKey[key]
			if !ok {
				t = &tally{name: name}
				byKey[key] = t
			}
			t.score += n.Similarity
			t.votes++
		}
	}
	if total == 0 {
		return nil
	}

	var winners []*tally
	for _, t := range byKey {
		if t.votes >= minVotes && t.score/total >= minVoteShare {
			winners = append(winners, t)
		}
	}
	sort.Slice(winners, func(i, j int) bool {
		if winners[i].score != winners[j].score {
			return winners[i].score > winners[j].score
		}
		return winners[i].name < winners[j].name
	})

	var out []string
	for i := 0; i < len(winners) && i < maxTags; i++ {
		out = append(out, winners[i].name)
	}
	return out
}

func merge(limit int, lists ...[]string) []string {
	seen := map[string]bool{}
	var out []string
	for _, list := range lists {
		for _, t := range list {
			key := strings.ToLower(t)
			if seen[key] || len(out) >= limit {
				continue
			}
			seen[key] = true
			out = append(out, t)
		}
	}
	return out
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
