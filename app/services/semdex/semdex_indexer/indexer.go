package semdex_indexer

import (
	"context"
	"log/slog"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/rs/xid"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/datagraph"
	"github.com/Southclaws/storyden/app/resources/library"
	"github.com/Southclaws/storyden/app/resources/post/thread"
	"github.com/Southclaws/storyden/app/services/semdex"
	"github.com/Southclaws/storyden/internal/config"
	"github.com/Southclaws/storyden/internal/ent"
	ent_node "github.com/Southclaws/storyden/internal/ent/node"
	ent_post "github.com/Southclaws/storyden/internal/ent/post"
)

const (
	chunkSize    = 20
	pollInterval = time.Minute
)

type Indexer struct {
	logger *slog.Logger
	db     *ent.Client
	semdex semdex.Semdexer
}

func Build() fx.Option {
	return fx.Options(
		fx.Provide(newIndexer),
		fx.Invoke(runOnBoot),
	)
}

func newIndexer(cfg config.Config, logger *slog.Logger, db *ent.Client, sd semdex.Semdexer) *Indexer {
	if cfg.SemdexProvider == "" {
		return nil
	}

	return &Indexer{
		logger: logger.With(slog.String("service", "semdex_indexer")),
		db:     db,
		semdex: sd,
	}
}

func runOnBoot(ctx context.Context, lc fx.Lifecycle, idx *Indexer) {
	if idx == nil {
		return
	}

	lc.Append(fx.StartHook(func() {
		go idx.loop(ctx)
	}))
}

func (idx *Indexer) loop(ctx context.Context) {
	if err := idx.ResetIfIndexEmpty(ctx); err != nil {
		idx.logger.Error("failed to check semdex state", slog.String("error", err.Error()))
	}

	t := time.NewTicker(pollInterval)
	defer t.Stop()

	for {
		if _, err := idx.Sync(ctx); err != nil && ctx.Err() == nil {
			idx.logger.Error("semdex sync failed", slog.String("error", err.Error()))
		}

		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// An empty index with markers set means its storage was wiped, e.g. to switch
// embedding models.
func (idx *Indexer) ResetIfIndexEmpty(ctx context.Context) error {
	counter, ok := idx.semdex.(semdex.Counter)
	if !ok {
		return nil
	}

	n, err := counter.Count(ctx)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}
	if n > 0 {
		return nil
	}

	posts, err := idx.db.Post.Update().
		Where(ent_post.SemdexIndexedAtNotNil()).
		ClearSemdexIndexedAt().
		Save(ctx)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	nodes, err := idx.db.Node.Update().
		Where(ent_node.SemdexIndexedAtNotNil()).
		ClearSemdexIndexedAt().
		Save(ctx)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	if posts+nodes > 0 {
		idx.logger.Info("semdex is empty, rebuilding", slog.Int("items", posts+nodes))
	}

	return nil
}

func (idx *Indexer) Sync(ctx context.Context) (int, error) {
	started := time.Now()
	total := 0

	for _, step := range []func(context.Context) (int, error){
		idx.indexThreads,
		idx.indexNodes,
		idx.removeThreads,
		idx.removeNodes,
	} {
		n, err := step(ctx)
		total += n
		if err != nil {
			return total, fault.Wrap(err, fctx.With(ctx))
		}
	}

	if total > 0 {
		idx.logger.Info("semdex sync completed",
			slog.Int("items", total),
			slog.Duration("duration", time.Since(started)),
		)
	}

	return total, nil
}

func stale(indexedAt, updatedAt string) func(*sql.Selector) {
	return func(s *sql.Selector) {
		s.Where(sql.Or(
			sql.IsNull(indexedAt),
			sql.GT(updatedAt, sql.Raw(indexedAt)),
		))
	}
}

func (idx *Indexer) indexThreads(ctx context.Context) (int, error) {
	return idx.indexChunks(ctx, func() ([]datagraph.Item, []xid.ID, error) {
		rows, err := idx.db.Post.Query().
			Where(
				ent_post.RootPostIDIsNil(),
				ent_post.DeletedAtIsNil(),
				ent_post.VisibilityEQ(ent_post.VisibilityPublished),
				stale(ent_post.FieldSemdexIndexedAt, ent_post.FieldUpdatedAt),
			).
			WithTags().
			WithAssets().
			Order(ent_post.ByUpdatedAt(), ent_post.ByID()).
			Limit(chunkSize).
			All(ctx)
		if err != nil {
			return nil, nil, fault.Wrap(err, fctx.With(ctx))
		}

		var items []datagraph.Item
		var skipped []xid.ID
		for _, r := range rows {
			item, err := thread.ItemRef(r)
			if err != nil {
				idx.logger.Warn("skipping unreadable thread", slog.String("id", r.ID.String()), slog.String("error", err.Error()))
				skipped = append(skipped, r.ID)
				continue
			}
			items = append(items, item)
		}
		return items, skipped, nil
	}, idx.markPosts)
}

func (idx *Indexer) indexNodes(ctx context.Context) (int, error) {
	return idx.indexChunks(ctx, func() ([]datagraph.Item, []xid.ID, error) {
		rows, err := idx.db.Node.Query().
			Where(
				ent_node.DeletedAtIsNil(),
				ent_node.VisibilityEQ(ent_node.VisibilityPublished),
				stale(ent_node.FieldSemdexIndexedAt, ent_node.FieldUpdatedAt),
			).
			WithAssets().
			Order(ent_node.ByUpdatedAt(), ent_node.ByID()).
			Limit(chunkSize).
			All(ctx)
		if err != nil {
			return nil, nil, fault.Wrap(err, fctx.With(ctx))
		}

		var items []datagraph.Item
		var skipped []xid.ID
		for _, r := range rows {
			item, err := library.ItemRef(r)
			if err != nil {
				idx.logger.Warn("skipping unreadable node", slog.String("id", r.ID.String()), slog.String("error", err.Error()))
				skipped = append(skipped, r.ID)
				continue
			}
			items = append(items, item)
		}
		return items, skipped, nil
	}, idx.markNodes)
}

func (idx *Indexer) indexChunks(
	ctx context.Context,
	fetch func() ([]datagraph.Item, []xid.ID, error),
	mark func(context.Context, []xid.ID) error,
) (int, error) {
	processed := 0

	for {
		if err := ctx.Err(); err != nil {
			return processed, err
		}

		items, skipped, err := fetch()
		if err != nil {
			return processed, err
		}
		if len(items) == 0 && len(skipped) == 0 {
			return processed, nil
		}

		if len(skipped) > 0 {
			if err := mark(ctx, skipped); err != nil {
				return processed, err
			}
		}

		done := make([]xid.ID, 0, len(items))
		for _, item := range items {
			if _, err := idx.semdex.Index(ctx, item); err != nil {
				// Stop the round: a failing embedding backend would otherwise spin.
				if len(done) > 0 {
					if merr := mark(ctx, done); merr != nil {
						return processed, merr
					}
				}
				return processed + len(done), fault.Wrap(err, fctx.With(ctx))
			}
			done = append(done, item.GetID())
		}

		if err := mark(ctx, done); err != nil {
			return processed, err
		}
		processed += len(done)

		idx.logger.Debug("semdex indexed chunk", slog.Int("items", len(done)), slog.Int("total", processed))
	}
}

// markAt is slightly in the future so databases with one-second timestamp
// precision never round it below the item's updated_at.
func markAt() time.Time {
	return time.Now().Add(2 * time.Second)
}

func (idx *Indexer) markPosts(ctx context.Context, ids []xid.ID) error {
	return idx.db.Post.Update().
		Where(ent_post.IDIn(ids...)).
		SetSemdexIndexedAt(markAt()).
		Exec(ctx)
}

func (idx *Indexer) markNodes(ctx context.Context, ids []xid.ID) error {
	return idx.db.Node.Update().
		Where(ent_node.IDIn(ids...)).
		SetSemdexIndexedAt(markAt()).
		Exec(ctx)
}

func (idx *Indexer) removeThreads(ctx context.Context) (int, error) {
	ids, err := idx.db.Post.Query().
		Where(
			ent_post.RootPostIDIsNil(),
			ent_post.SemdexIndexedAtNotNil(),
			ent_post.Or(
				ent_post.DeletedAtNotNil(),
				ent_post.VisibilityNEQ(ent_post.VisibilityPublished),
			),
		).
		IDs(ctx)
	if err != nil {
		return 0, fault.Wrap(err, fctx.With(ctx))
	}

	return idx.remove(ctx, ids, func(ids []xid.ID) error {
		return idx.db.Post.Update().Where(ent_post.IDIn(ids...)).ClearSemdexIndexedAt().Exec(ctx)
	})
}

func (idx *Indexer) removeNodes(ctx context.Context) (int, error) {
	ids, err := idx.db.Node.Query().
		Where(
			ent_node.SemdexIndexedAtNotNil(),
			ent_node.Or(
				ent_node.DeletedAtNotNil(),
				ent_node.VisibilityNEQ(ent_node.VisibilityPublished),
			),
		).
		IDs(ctx)
	if err != nil {
		return 0, fault.Wrap(err, fctx.With(ctx))
	}

	return idx.remove(ctx, ids, func(ids []xid.ID) error {
		return idx.db.Node.Update().Where(ent_node.IDIn(ids...)).ClearSemdexIndexedAt().Exec(ctx)
	})
}

func (idx *Indexer) remove(ctx context.Context, ids []xid.ID, clear func([]xid.ID) error) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	for _, id := range ids {
		if _, err := idx.semdex.Delete(ctx, id); err != nil {
			return 0, fault.Wrap(err, fctx.With(ctx))
		}
	}

	if err := clear(ids); err != nil {
		return 0, fault.Wrap(err, fctx.With(ctx))
	}

	return len(ids), nil
}
