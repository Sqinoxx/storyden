// Package thread_similar finds published threads close in meaning to a
// thread or to a draft. It only uses stored embeddings or the embedding of
// the draft, never the chat model, so it is cheap enough to run on every view.
package thread_similar

import (
	"context"
	"strings"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/rs/xid"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/datagraph"
	"github.com/Southclaws/storyden/app/resources/datagraph/hydrate"
	"github.com/Southclaws/storyden/app/resources/pagination"
	"github.com/Southclaws/storyden/app/resources/post"
	"github.com/Southclaws/storyden/app/resources/post/thread"
	"github.com/Southclaws/storyden/app/services/access/access_check"
	"github.com/Southclaws/storyden/app/services/search/searcher"
	"github.com/Southclaws/storyden/app/services/semdex"
)

const (
	// Measured with bge-m3 on the production threads: two threads about the
	// same exam or subject reach 0.7, while a draft (a short question) reaches
	// about 0.65 with clearly matching threads and stays below about 0.58
	// with unrelated ones.
	MinRelatedSimilarity = 0.7
	MinDraftSimilarity   = 0.65

	MaxRelated    = 5
	MaxForDraft   = 3
	candidates    = 12
	minDraftRunes = 20
	maxDraftRunes = 4000
)

type Finder struct {
	searcher    semdex.Searcher
	recommender semdex.Recommender
	hydrator    *hydrate.Hydrator
	access      *access_check.Checker
}

func Build() fx.Option {
	return fx.Provide(New)
}

func New(searcher semdex.Searcher, recommender semdex.Recommender, hydrator *hydrate.Hydrator, access *access_check.Checker) *Finder {
	return &Finder{searcher: searcher, recommender: recommender, hydrator: hydrator, access: access}
}

func (f *Finder) Enabled() bool {
	_, disabled := f.searcher.(*semdex.Disabled)
	return f.searcher != nil && !disabled
}

// Related lists threads similar to an indexed thread, the thread itself
// excluded. A thread that is not indexed yet has no related threads.
func (f *Finder) Related(ctx context.Context, threadID post.ID) (datagraph.ItemList, error) {
	if !f.Enabled() {
		return nil, nil
	}

	self := xid.ID(threadID)
	refs, err := f.recommender.RecommendRefs(ctx, &thread.Thread{Post: post.Post{ID: threadID}})
	if err != nil {
		return nil, nil
	}

	return f.hydrate(ctx, Pick(refs, self, MinRelatedSimilarity, MaxRelated))
}

// ForDraft lists threads similar to a post being written, using the same text
// as the tag suggestion so the embedding is computed only once.
func (f *Finder) ForDraft(ctx context.Context, title, body string) (datagraph.ItemList, error) {
	if !f.Enabled() {
		return nil, nil
	}

	title, body = strings.TrimSpace(title), strings.TrimSpace(body)
	if len([]rune(title+body)) < minDraftRunes {
		return nil, nil
	}
	if r := []rune(body); len(r) > maxDraftRunes {
		body = string(r[:maxDraftRunes])
	}

	res, err := f.searcher.SearchRefs(ctx, title+"\n\n"+body, pagination.NewPageParams(1, candidates), searcher.Options{})
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return f.hydrate(ctx, Pick(res.Items, xid.NilID(), MinDraftSimilarity, MaxForDraft))
}

// Pick keeps the most similar threads from minSimilarity.
func Pick(refs []*datagraph.Ref, exclude xid.ID, minSimilarity float64, limit int) []*datagraph.Ref {
	var out []*datagraph.Ref
	for _, r := range refs {
		if len(out) == limit {
			break
		}
		if r.ID == exclude || r.Kind != datagraph.KindThread || r.Relevance < minSimilarity {
			continue
		}
		out = append(out, r)
	}
	return out
}

func (f *Finder) hydrate(ctx context.Context, refs []*datagraph.Ref) (datagraph.ItemList, error) {
	if len(refs) == 0 {
		return nil, nil
	}

	items, err := f.hydrator.Hydrate(ctx, refs...)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	visible, err := f.access.FilterItems(ctx, items)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}
	return visible, nil
}
