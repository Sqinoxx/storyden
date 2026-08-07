// Package thread provides APIs for working with threads which are sequences of
// posts. Threads can be created with one post, listed, searched and updated.
package thread

import (
	"context"
	"log/slog"
	"net/url"

	"github.com/Southclaws/opt"
	"github.com/rs/xid"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/app/resources/asset" //NEU
	"github.com/Southclaws/storyden/app/resources/datagraph"
	"github.com/Southclaws/storyden/app/resources/pagination"
	"github.com/Southclaws/storyden/app/resources/post"
	"github.com/Southclaws/storyden/app/resources/post/category"
	"github.com/Southclaws/storyden/app/resources/post/thread"
	"github.com/Southclaws/storyden/app/resources/post/thread_cache"
	"github.com/Southclaws/storyden/app/resources/post/thread_querier"
	"github.com/Southclaws/storyden/app/resources/post/thread_writer"
	"github.com/Southclaws/storyden/app/resources/tag/tag_ref"
	"github.com/Southclaws/storyden/app/resources/tag/tag_writer"
	"github.com/Southclaws/storyden/app/resources/visibility"
	"github.com/Southclaws/storyden/app/services/asset/asset_link"
	"github.com/Southclaws/storyden/app/services/link/fetcher"
	"github.com/Southclaws/storyden/app/services/mention/mentioner"
	"github.com/Southclaws/storyden/app/services/moderation"
	"github.com/Southclaws/storyden/app/services/report/system_report"
	"github.com/Southclaws/storyden/app/services/semdex"
	"github.com/Southclaws/storyden/internal/infrastructure/instrumentation/spanner"
	"github.com/Southclaws/storyden/internal/infrastructure/pubsub"
)

type Service interface {
	// Create a new thread with optional category.
	Create(
		ctx context.Context,
		title string,
		authorID account.AccountID,
		meta map[string]any,
		partial Partial,
	) (*thread.Thread, error)

	Update(ctx context.Context, threadID post.ID, partial Partial) (*thread.Thread, error)

	Delete(ctx context.Context, id post.ID) error

	List(ctx context.Context,
		page int,
		size int,
		opts Params,
	) (*thread_querier.Result, error)

	// Get one thread and the posts within it.
	Get(
		ctx context.Context,
		threadID post.ID,
		pageParams pagination.Parameters,
	) (*thread.Thread, error)
}

type Partial struct {
	Title      opt.Optional[string]
	Content    opt.Optional[datagraph.Content]
	Category   opt.Optional[xid.ID]
	Tags       opt.Optional[tag_ref.Names]
	Visibility opt.Optional[visibility.Visibility]
	URL        opt.Optional[url.URL]
	Meta       opt.Optional[map[string]any]
	Pinned     opt.Optional[int]
	Assets     opt.Optional[[]asset.AssetID] // NEU
}

func (p Partial) Opts() (opts []thread_writer.Option) {
	p.Title.Call(func(v string) { opts = append(opts, thread_writer.WithTitle(v)) })
	p.Content.Call(func(v datagraph.Content) { opts = append(opts, thread_writer.WithContent(v)) })
	p.Category.Call(func(v xid.ID) { opts = append(opts, thread_writer.WithCategory(xid.ID(v))) })
	p.Visibility.Call(func(v visibility.Visibility) { opts = append(opts, thread_writer.WithVisibility(v)) })
	p.Meta.Call(func(v map[string]any) { opts = append(opts, thread_writer.WithMeta(v)) })
	p.Pinned.Call(func(v int) { opts = append(opts, thread_writer.WithPinned(v)) })
	p.Assets.Call(func(v []asset.AssetID) { opts = append(opts, thread_writer.WithAssets(v)) }) // NEU
	return
}

func Build() fx.Option {
	return fx.Options(
		fx.Provide(New),
	)
}

type service struct {
	logger *slog.Logger
	ins    spanner.Instrumentation

	categoryRepo   *category.Repository
	threadQuerier  *thread_querier.Querier
	threadWriter   *thread_writer.Writer
	tagWriter      *tag_writer.Writer
	fetcher        *fetcher.Fetcher
	recommender    semdex.Recommender
	bus            *pubsub.Bus
	mentioner      *mentioner.Mentioner
	cpm            *moderation.Manager
	cache          *thread_cache.Cache
	systemReporter *system_report.Manager
	assetLink      *asset_link.Resolver
}

func New(
	logger *slog.Logger,
	ins spanner.Builder,

	categoryRepo *category.Repository,
	threadQuerier *thread_querier.Querier,
	threadWriter *thread_writer.Writer,
	tagWriter *tag_writer.Writer,
	fetcher *fetcher.Fetcher,
	recommender semdex.Recommender,
	bus *pubsub.Bus,
	mentioner *mentioner.Mentioner,
	cpm *moderation.Manager,
	cache *thread_cache.Cache,
	systemReporter *system_report.Manager,
	assetLink *asset_link.Resolver,
) Service {
	return &service{
		logger: logger,
		ins:    ins.Build(),

		categoryRepo:   categoryRepo,
		threadQuerier:  threadQuerier,
		threadWriter:   threadWriter,
		tagWriter:      tagWriter,
		fetcher:        fetcher,
		recommender:    recommender,
		bus:            bus,
		mentioner:      mentioner,
		cpm:            cpm,
		cache:          cache,
		systemReporter: systemReporter,
		assetLink:      assetLink,
	}
}

// appendDerivedAssetOpts appends an additive asset-linking option derived
// from body content (e.g. file attachments referenced by URL) only when the
// caller did not explicitly set asset_ids. This must never fire alongside
// an explicit Assets value: thread_writer.WithAssets clears existing edges
// before re-adding, so appending an additive option after it is harmless,
// but computing derived IDs when Assets is unset and Content is present is
// what makes attachments in bodies without explicit asset_ids (e.g. replies
// mirrored into threads, or clients that don't send asset_ids) searchable.
func (s *service) appendDerivedAssetOpts(ctx context.Context, opts []thread_writer.Option, partial Partial) []thread_writer.Option {
	if partial.Assets.Ok() {
		return opts
	}

	content, ok := partial.Content.Get()
	if !ok {
		return opts
	}

	ids, err := s.assetLink.Resolve(ctx, content)
	if err != nil || len(ids) == 0 {
		return opts
	}

	return append(opts, thread_writer.WithAssetsAdd(ids))
}
