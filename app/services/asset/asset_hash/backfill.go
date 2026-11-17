// Package asset_hash fills in content hashes for assets uploaded before
// duplicate detection existed. Assets whose object cannot be read get an empty
// hash so they are not retried on every start.
package asset_hash

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"sync"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/asset"
	"github.com/Southclaws/storyden/app/resources/asset/asset_writer"
	"github.com/Southclaws/storyden/internal/ent"
	ent_asset "github.com/Southclaws/storyden/internal/ent/asset"
	"github.com/Southclaws/storyden/internal/infrastructure/object"
)

const batchSize = 50

func Build() fx.Option {
	return fx.Options(
		fx.Provide(New),
		fx.Invoke(func(*Backfill) {}),
	)
}

type Backfill struct {
	logger  *slog.Logger
	db      *ent.Client
	writer  *asset_writer.Writer
	objects object.Storer
}

func New(ctx context.Context, lc fx.Lifecycle, logger *slog.Logger, db *ent.Client, writer *asset_writer.Writer, objects object.Storer) *Backfill {
	b := &Backfill{logger: logger, db: db, writer: writer, objects: objects}

	run, cancel := context.WithCancel(ctx)
	var wg sync.WaitGroup

	lc.Append(fx.StartHook(func() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			n, err := b.Run(run)
			if err != nil && run.Err() == nil {
				logger.Warn("asset content hash backfill failed", slog.String("error", err.Error()))
				return
			}
			if n > 0 {
				logger.Info("asset content hash backfill finished", slog.Int("hashed", n))
			}
		}()
	}))

	lc.Append(fx.StopHook(func() {
		cancel()
		wg.Wait()
	}))

	return b
}

func (b *Backfill) Run(ctx context.Context) (int, error) {
	total := 0
	for {
		rows, err := b.db.Asset.Query().
			Where(ent_asset.ContentHashIsNil()).
			Select(ent_asset.FieldID, ent_asset.FieldFilename).
			Limit(batchSize).
			All(ctx)
		if err != nil {
			return total, fault.Wrap(err, fctx.With(ctx))
		}
		if len(rows) == 0 {
			return total, nil
		}

		for _, r := range rows {
			if err := ctx.Err(); err != nil {
				return total, err
			}

			hash := b.hash(ctx, asset.NewFilepathFilename(r.Filename))
			if err := b.writer.SetContentHash(ctx, r.ID, hash); err != nil {
				return total, fault.Wrap(err, fctx.With(ctx))
			}
			if hash != "" {
				total++
			}
		}
	}
}

func (b *Backfill) hash(ctx context.Context, name asset.Filename) string {
	r, _, err := b.objects.Read(ctx, asset.BuildAssetPath(name))
	if err != nil {
		return ""
	}
	if c, ok := r.(io.Closer); ok {
		defer c.Close()
	}

	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return ""
	}

	return hex.EncodeToString(h.Sum(nil))
}
