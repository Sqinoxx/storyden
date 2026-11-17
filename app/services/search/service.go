package search

import (
	"log/slog"

	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/datagraph/hydrate"

	"github.com/Southclaws/storyden/app/services/access/access_check"
	"github.com/Southclaws/storyden/app/services/search/asset_match"
	"github.com/Southclaws/storyden/app/services/search/bleve_search"
	"github.com/Southclaws/storyden/app/services/search/redis_search"
	"github.com/Southclaws/storyden/app/services/search/searcher"
	"github.com/Southclaws/storyden/app/services/search/simplesearch"
	"github.com/Southclaws/storyden/app/services/semdex"
	"github.com/Southclaws/storyden/internal/config"
)

func newSearcher(
	cfg config.Config,
	logger *slog.Logger,
	simpleSearcher *simplesearch.ParallelSearcher,
	bleveSearcher *bleve_search.BleveSearcher,
	redisSearcher *redis_search.RedisSearcher,
	access *access_check.Checker,
	sd semdex.Searcher,
	hydrator *hydrate.Hydrator,
) searcher.Searcher {
	next := providerSearcher(cfg, simpleSearcher, bleveSearcher, redisSearcher)

	if _, disabled := sd.(*semdex.Disabled); !disabled && sd != nil {
		next = &hybridSearcher{next: next, semdex: sd, hydrator: hydrator, logger: logger}
	}

	return &accessFilteredSearcher{next: next, access: access}
}

func providerSearcher(
	cfg config.Config,
	simpleSearcher *simplesearch.ParallelSearcher,
	bleveSearcher *bleve_search.BleveSearcher,
	redisSearcher *redis_search.RedisSearcher,
) searcher.Searcher {
	switch cfg.SearchProvider {
	case "bleve":
		return bleveSearcher

	case "redis":
		return redisSearcher

	case "database":
		fallthrough
	default:
		return simpleSearcher
	}
}

func newIndexer(
	cfg config.Config,
	bleveSearcher *bleve_search.BleveSearcher,
	redisSearcher *redis_search.RedisSearcher,
) searcher.Indexer {
	switch cfg.SearchProvider {
	case "bleve":
		return bleveSearcher

	case "redis":
		return redisSearcher

	default:
		return nil
	}
}

func Build() fx.Option {
	return fx.Options(
		fx.Provide(
			newSearcher,
			newIndexer,
			simplesearch.NewParallelSearcher,
			asset_match.New,
		),
	)
}
