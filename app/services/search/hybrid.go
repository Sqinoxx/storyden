package search

import (
	"context"
	"log/slog"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rs/xid"

	"github.com/Southclaws/storyden/app/resources/datagraph"
	"github.com/Southclaws/storyden/app/resources/datagraph/hydrate"
	"github.com/Southclaws/storyden/app/resources/pagination"
	"github.com/Southclaws/storyden/app/services/search/searcher"
	"github.com/Southclaws/storyden/app/services/semdex"
)

const (
	semanticTimeout    = 1500 * time.Millisecond
	semanticCandidates = 20
	semanticMaxResults = 8
	minQueryRunes      = 3
	rrfK               = 60.0
)

// bge-m3 rates almost any two forum posts above 0.4. From 0.6 a thread's
// title still finds 4.5 of its 5 nearest threads on average (checked on 211
// production threads), anything weaker is mostly noise.
const MinSemanticSimilarity = 0.6

// hybridSearcher adds threads that match the meaning of the query to the
// keyword results of the first page. Keyword results whose title contains the
// query stay on top, the rest is ranked by reciprocal rank fusion.
type hybridSearcher struct {
	next     searcher.Searcher
	semdex   semdex.Searcher
	hydrator *hydrate.Hydrator
	logger   *slog.Logger
}

func (h *hybridSearcher) MatchFast(ctx context.Context, q string, limit int, opts searcher.Options) (datagraph.MatchList, error) {
	return h.next.MatchFast(ctx, q, limit, opts)
}

func semanticApplies(q string, p pagination.Parameters, opts searcher.Options) bool {
	if p.PageOneIndexed() != 1 || len([]rune(strings.TrimSpace(q))) < minQueryRunes {
		return false
	}
	if opts.Authors.Ok() || opts.Categories.Ok() || opts.Tags.Ok() {
		return false
	}
	if kinds, ok := opts.Kinds.Get(); ok && !slices.Contains(kinds, datagraph.KindThread) && !slices.Contains(kinds, datagraph.KindPost) {
		return false
	}
	return true
}

func (h *hybridSearcher) Search(ctx context.Context, q string, p pagination.Parameters, opts searcher.Options) (*pagination.Result[datagraph.Item], error) {
	if !semanticApplies(q, p, opts) {
		return h.next.Search(ctx, q, p, opts)
	}

	var (
		wg       sync.WaitGroup
		semantic datagraph.ItemList
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		sctx, cancel := context.WithTimeout(ctx, semanticTimeout)
		defer cancel()

		items, err := h.semantic(sctx, q)
		if err != nil {
			h.logger.Warn("semantic search failed, using keyword results only", slog.String("error", err.Error()))
			return
		}
		semantic = items
	}()

	keyword, err := h.next.Search(ctx, q, p, opts)
	wg.Wait()
	if err != nil {
		return nil, err
	}
	if len(semantic) == 0 {
		return keyword, nil
	}

	merged, added := Fuse(q, keyword.Items, semantic)
	if collector, ok := ctx.Value(semanticKey{}).(*SemanticMatches); ok {
		collector.add(added)
	}

	out := *keyword
	out.Items = merged
	out.Results += len(added)
	return &out, nil
}

func (h *hybridSearcher) semantic(ctx context.Context, q string) (datagraph.ItemList, error) {
	res, err := h.semdex.SearchRefs(ctx, q, pagination.NewPageParams(1, semanticCandidates), searcher.Options{})
	if err != nil {
		return nil, err
	}

	var refs []*datagraph.Ref
	for _, r := range res.Items {
		if r.Kind == datagraph.KindThread && r.Relevance >= MinSemanticSimilarity && len(refs) < semanticMaxResults {
			refs = append(refs, r)
		}
	}
	if len(refs) == 0 {
		return nil, nil
	}

	return h.hydrator.Hydrate(ctx, refs...)
}

// Fuse merges keyword and semantic results and returns the IDs only the
// semantic search found.
func Fuse(q string, keyword, semantic datagraph.ItemList) (datagraph.ItemList, []xid.ID) {
	needle := strings.ToLower(strings.TrimSpace(q))

	type entry struct {
		item   datagraph.Item
		score  float64
		pinned bool
		order  int
	}

	byID := map[xid.ID]*entry{}
	var entries []*entry
	get := func(item datagraph.Item) *entry {
		if e, ok := byID[item.GetID()]; ok {
			return e
		}
		e := &entry{item: item, order: len(entries)}
		byID[item.GetID()] = e
		entries = append(entries, e)
		return e
	}

	for i, item := range keyword {
		e := get(item)
		e.score += 1 / (rrfK + float64(i+1))
		e.pinned = needle != "" && strings.Contains(strings.ToLower(item.GetName()), needle)
	}

	inKeyword := len(entries)
	var added []xid.ID
	for i, item := range semantic {
		e := get(item)
		e.score += 1 / (rrfK + float64(i+1))
		if e.order >= inKeyword {
			added = append(added, item.GetID())
		}
	}

	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.pinned != b.pinned {
			return a.pinned
		}
		if a.pinned {
			return a.order < b.order
		}
		return a.score > b.score
	})

	out := make(datagraph.ItemList, len(entries))
	for i, e := range entries {
		out[i] = e.item
	}
	return out, added
}

type semanticKey struct{}

// SemanticMatches collects the IDs of results only found by meaning, so the
// API can label them.
type SemanticMatches struct {
	mu  sync.Mutex
	ids []xid.ID
}

func WithSemanticMatches(ctx context.Context) (context.Context, *SemanticMatches) {
	m := &SemanticMatches{}
	return context.WithValue(ctx, semanticKey{}, m), m
}

func (m *SemanticMatches) add(ids []xid.ID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ids = append(m.ids, ids...)
}

func (m *SemanticMatches) IDs() []xid.ID {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.ids)
}
