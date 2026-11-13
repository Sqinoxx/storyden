package asset_querier

import (
	"context"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/rs/xid"

	"github.com/Southclaws/storyden/app/resources/asset"
	"github.com/Southclaws/storyden/internal/ent"
	ent_asset "github.com/Southclaws/storyden/internal/ent/asset"
	ent_post "github.com/Southclaws/storyden/internal/ent/post"
)

func (q *Querier) GetInThread(ctx context.Context, id xid.ID, threadID xid.ID) (*asset.Asset, error) {
	r, err := q.db.Asset.Query().
		Where(
			ent_asset.ID(id),
			ent_asset.HasPostsWith(
				ent_post.DeletedAtIsNil(),
				ent_post.Or(ent_post.ID(threadID), ent_post.RootPostID(threadID)),
			),
		).
		Only(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}
	return asset.Map(r), nil
}

func (q *Querier) NextQueuedSummary(ctx context.Context) (*asset.Asset, error) {
	r, err := q.db.Asset.Query().
		Where(
			ent_asset.AiSummaryStatusEQ(ent_asset.AiSummaryStatusQueued),
			ent_asset.OcrStatusEQ(ent_asset.OcrStatusCompleted),
		).
		Order(ent.Desc(ent_asset.FieldCreatedAt)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, fault.Wrap(err, fctx.With(ctx))
	}
	return asset.Map(r), nil
}

type SummaryStats struct {
	Total int
	Done  int
}

func (q *Querier) GetSummaryStats(ctx context.Context) (*SummaryStats, error) {
	base := func() *ent.AssetQuery {
		return q.db.Asset.Query().Where(
			ent_asset.MimeTypeEQ("application/pdf"),
			ent_asset.OcrStatusEQ(ent_asset.OcrStatusCompleted),
		)
	}
	total, err := base().Count(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}
	done, err := base().Where(ent_asset.AiSummaryNotNil(), ent_asset.AiSummaryNEQ("")).Count(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}
	return &SummaryStats{Total: total, Done: done}, nil
}

func (q *Querier) CountProcessingOCR(ctx context.Context) (int, error) {
	n, err := q.db.Asset.Query().
		Where(ent_asset.OcrStatusEQ(ent_asset.OcrStatusProcessing)).
		Count(ctx)
	if err != nil {
		return 0, fault.Wrap(err, fctx.With(ctx))
	}
	return n, nil
}
