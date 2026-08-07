package asset_querier

import (
	"context"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/ftag"

	"github.com/Southclaws/storyden/app/resources/asset"
	"github.com/Southclaws/storyden/internal/ent"
	ent_asset "github.com/Southclaws/storyden/internal/ent/asset"
)

type Querier struct {
	db *ent.Client
}

func New(db *ent.Client) *Querier {
	return &Querier{db}
}

func (q *Querier) Get(ctx context.Context, id asset.Filename) (*asset.Asset, error) {
	r, err := q.db.Asset.Query().Where(
		ent_asset.Filename(id.String()),
	).First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fault.Wrap(err, fctx.With(ctx), ftag.With(ftag.NotFound))
		}
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return asset.Map(r), nil
}

func (q *Querier) GetByID(ctx context.Context, id asset.AssetID) (*asset.Asset, error) {
	r, err := q.db.Asset.Query().Where(
		ent_asset.ID(id),
	).First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fault.Wrap(err, fctx.With(ctx), ftag.With(ftag.NotFound))
		}
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return asset.Map(r), nil
}

func (q *Querier) GetAll(ctx context.Context) ([]*asset.Asset, error) {
	assets, err := q.db.Asset.Query().All(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}
	result := make([]*asset.Asset, len(assets))
	for i, a := range assets {
		result[i] = asset.Map(a)
	}
	return result, nil
}

func (q *Querier) GetPendingOCR(ctx context.Context, limit int) ([]*asset.Asset, error) {
	assets, err := q.db.Asset.Query().
		Where(
			ent_asset.Or(
				ent_asset.OcrStatusEQ(ent_asset.OcrStatusPending),
				ent_asset.OcrStatusEQ(ent_asset.OcrStatusFailed),
			),
		).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	result := make([]*asset.Asset, len(assets))
	for i, a := range assets {
		result[i] = asset.Map(a)
	}
	return result, nil
}

type OCRStats struct {
	Total     int `json:"total"`
	Pending   int `json:"pending"`
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
	Skipped   int `json:"skipped"`
}

func (q *Querier) GetOCRStats(ctx context.Context) (*OCRStats, error) {
	total, err := q.db.Asset.Query().Count(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}
	completed, err := q.db.Asset.Query().Where(ent_asset.OcrStatusEQ(ent_asset.OcrStatusCompleted)).Count(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}
	failed, err := q.db.Asset.Query().Where(ent_asset.OcrStatusEQ(ent_asset.OcrStatusFailed)).Count(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}
	skipped, err := q.db.Asset.Query().Where(ent_asset.OcrStatusEQ(ent_asset.OcrStatusSkipped)).Count(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}
	pending := total - (completed + failed + skipped)

	return &OCRStats{
		Total:     total,
		Pending:   pending,
		Completed: completed,
		Failed:    failed,
		Skipped:   skipped,
	}, nil
}
