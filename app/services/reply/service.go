// Package reply provides APIs for managing posts within a thread.
package reply

import (
	"context"
	"log/slog"

	"github.com/Southclaws/opt"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_querier"
	"github.com/Southclaws/storyden/app/resources/asset"
	"github.com/Southclaws/storyden/app/resources/datagraph"
	"github.com/Southclaws/storyden/app/resources/post"
	"github.com/Southclaws/storyden/app/resources/post/reply_querier"
	"github.com/Southclaws/storyden/app/resources/post/reply_writer"
	"github.com/Southclaws/storyden/app/resources/post/thread_cache"
	"github.com/Southclaws/storyden/app/resources/visibility"
	"github.com/Southclaws/storyden/app/services/asset/asset_link"
	"github.com/Southclaws/storyden/app/services/link/fetcher"
	"github.com/Southclaws/storyden/app/services/moderation"
	"github.com/Southclaws/storyden/app/services/reply/reply_notify"
	"github.com/Southclaws/storyden/app/services/report/system_report"
	"github.com/Southclaws/storyden/internal/infrastructure/pubsub"
)

type Partial struct {
	Content    opt.Optional[datagraph.Content]
	ReplyTo    opt.Optional[post.ID]
	Meta       opt.Optional[map[string]any]
	Visibility opt.Optional[visibility.Visibility]
	Assets     opt.Optional[[]asset.AssetID]
}

func (p Partial) Opts() (opts []reply_writer.Option) {
	p.Content.Call(func(v datagraph.Content) { opts = append(opts, reply_writer.WithContent(v)) })
	p.ReplyTo.Call(func(v post.ID) { opts = append(opts, reply_writer.WithReplyTo(v)) })
	p.Meta.Call(func(v map[string]any) { opts = append(opts, reply_writer.WithMeta(v)) })
	p.Visibility.Call(func(v visibility.Visibility) { opts = append(opts, reply_writer.WithVisibility(v)) })
	p.Assets.Call(func(v []asset.AssetID) { opts = append(opts, reply_writer.WithAssets(v...)) })
	return
}

func Build() fx.Option {
	return fx.Options(
		fx.Provide(New),
		reply_notify.Build(),
	)
}

type Mutator struct {
	logger         *slog.Logger
	accountQuery   *account_querier.Querier
	replyQuerier   *reply_querier.Querier
	replyWriter    *reply_writer.Writer
	fetcher        *fetcher.Fetcher
	bus            *pubsub.Bus
	cpm            *moderation.Manager
	cache          *thread_cache.Cache
	systemReporter *system_report.Manager
	assetLink      *asset_link.Resolver
}

func New(
	logger *slog.Logger,
	accountQuery *account_querier.Querier,
	replyQuerier *reply_querier.Querier,
	replyWriter *reply_writer.Writer,
	fetcher *fetcher.Fetcher,
	bus *pubsub.Bus,
	cpm *moderation.Manager,
	cache *thread_cache.Cache,
	systemReporter *system_report.Manager,
	assetLink *asset_link.Resolver,
) *Mutator {
	return &Mutator{
		logger:         logger,
		accountQuery:   accountQuery,
		replyQuerier:   replyQuerier,
		replyWriter:    replyWriter,
		fetcher:        fetcher,
		bus:            bus,
		cpm:            cpm,
		cache:          cache,
		systemReporter: systemReporter,
		assetLink:      assetLink,
	}
}

// appendDerivedAssetOpts appends an additive asset-linking option derived
// from body content (e.g. file attachments referenced by URL, as emitted by
// the reply composer's non-image attachment path). This only runs when the
// caller did not explicitly set asset_ids, mirroring the thread service:
// reply_writer.WithAssets (explicit) replaces the edge set, so appending an
// additive option after it is harmless, but deriving IDs only when Assets is
// unset is what makes attachments searchable for callers that don't send
// asset_ids (e.g. the reply composer's non-image attachment path).
func (s *Mutator) appendDerivedAssetOpts(ctx context.Context, opts []reply_writer.Option, partial Partial) []reply_writer.Option {
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

	return append(opts, reply_writer.WithAssetsAdd(ids...))
}
