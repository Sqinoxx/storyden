package search

import (
	"context"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"

	"github.com/Southclaws/storyden/app/resources/datagraph"
	"github.com/Southclaws/storyden/app/resources/pagination"
	"github.com/Southclaws/storyden/app/services/access/access_check"
	"github.com/Southclaws/storyden/app/services/search/searcher"
)

// accessFilteredSearcher sits in front of every search provider so access
// passes apply no matter which backend produced the results.
type accessFilteredSearcher struct {
	next   searcher.Searcher
	access *access_check.Checker
}

func (s *accessFilteredSearcher) Search(ctx context.Context, q string, p pagination.Parameters, opts searcher.Options) (*pagination.Result[datagraph.Item], error) {
	r, err := s.next.Search(ctx, q, p, opts)
	if err != nil {
		return nil, err
	}

	items, err := s.access.FilterItems(ctx, r.Items)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	r.Items = items

	return r, nil
}

func (s *accessFilteredSearcher) MatchFast(ctx context.Context, q string, limit int, opts searcher.Options) (datagraph.MatchList, error) {
	m, err := s.next.MatchFast(ctx, q, limit, opts)
	if err != nil {
		return nil, err
	}

	return s.access.FilterMatches(ctx, m)
}
