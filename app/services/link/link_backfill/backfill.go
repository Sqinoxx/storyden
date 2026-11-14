// Package link_backfill sets the link of threads whose body embeds a link
// preview card but which were saved before cards were promoted to the thread
// link, so list views can show the preview without opening the thread.
package link_backfill

import (
	"context"
	"log/slog"
	"net/url"

	"github.com/rs/xid"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/app/resources/account/account_repo"
	"github.com/Southclaws/storyden/app/resources/datagraph"
	"github.com/Southclaws/storyden/app/resources/post/thread_cache"
	"github.com/Southclaws/storyden/app/resources/rbac"
	"github.com/Southclaws/storyden/app/services/authentication/session"
	"github.com/Southclaws/storyden/app/services/link/fetcher"
	"github.com/Southclaws/storyden/internal/ent"
	ent_post "github.com/Southclaws/storyden/internal/ent/post"
)

func Build() fx.Option {
	return fx.Options(
		fx.Provide(New),
		fx.Invoke(start),
	)
}

type Backfill struct {
	logger   *slog.Logger
	db       *ent.Client
	fetcher  *fetcher.Fetcher
	cache    *thread_cache.Cache
	accounts *account_repo.Repository
}

func New(logger *slog.Logger, db *ent.Client, f *fetcher.Fetcher, cache *thread_cache.Cache, accounts *account_repo.Repository) *Backfill {
	return &Backfill{logger: logger, db: db, fetcher: f, cache: cache, accounts: accounts}
}

func start(lc fx.Lifecycle, b *Backfill) {
	ctx, cancel := context.WithCancel(context.Background())

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go b.Run(ctx)
			return nil
		},
		OnStop: func(context.Context) error {
			cancel()
			return nil
		},
	})
}

func (b *Backfill) Run(ctx context.Context) (updated int) {
	defer func() {
		if r := recover(); r != nil {
			b.logger.Error("link card backfill panicked", slog.Any("panic", r))
		}
	}()

	posts, err := b.db.Post.Query().
		Where(
			ent_post.RootPostIDIsNil(),
			ent_post.LinkIDIsNil(),
			ent_post.DeletedAtIsNil(),
			ent_post.BodyContains(`data-display="card"`),
		).
		Select(ent_post.FieldID, ent_post.FieldBody, ent_post.FieldAccountPosts).
		All(ctx)
	if err != nil {
		b.logger.Warn("link card backfill query failed", slog.String("error", err.Error()))
		return 0
	}

	for _, p := range posts {
		if ctx.Err() != nil {
			break
		}
		if b.apply(ctx, p) {
			updated++
		}
	}

	if updated > 0 {
		b.logger.Info("link card backfill finished", slog.Int("threads", updated))
	}

	return updated
}

func (b *Backfill) apply(ctx context.Context, p *ent.Post) bool {
	content, err := datagraph.NewRichText(p.Body)
	if err != nil {
		return false
	}

	cards := content.LinkCards()
	if len(cards) == 0 {
		return false
	}

	u, err := url.Parse(cards[0])
	if err != nil {
		return false
	}

	author, err := b.accounts.GetByID(ctx, account.AccountID(p.AccountPosts))
	if err != nil {
		return false
	}

	ln, err := b.fetcher.Fetch(session.WithAccountPermissions(ctx, author.Account, rbac.NewList()), *u)
	if err != nil {
		b.logger.Warn("link card backfill fetch failed", slog.String("url", cards[0]), slog.String("error", err.Error()))
		return false
	}

	err = b.db.Post.UpdateOneID(p.ID).
		Where(ent_post.LinkIDIsNil()).
		SetLinkID(xid.ID(ln.ID)).
		Exec(ctx)
	if err != nil {
		return false
	}

	_ = b.cache.Invalidate(ctx, p.ID)

	return true
}
