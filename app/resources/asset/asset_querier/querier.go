package asset_querier

import (
	"context"
	"time"

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

// GetPendingOCR returns assets that need text extraction: those never
// processed, those that previously failed, and those stuck in `processing`
// for longer than stuckAfter (e.g. because the process crashed mid-run).
func (q *Querier) GetPendingOCR(ctx context.Context, limit int, stuckAfter time.Duration) ([]*asset.Asset, error) {
	stuckBefore := time.Now().Add(-stuckAfter)

	assets, err := q.db.Asset.Query().
		Where(
			ent_asset.Or(
				ent_asset.OcrStatusEQ(ent_asset.OcrStatusPending),
				ent_asset.OcrStatusEQ(ent_asset.OcrStatusFailed),
				ent_asset.And(
					ent_asset.OcrStatusEQ(ent_asset.OcrStatusProcessing),
					ent_asset.OcrProcessedAtLT(stuckBefore),
				),
			),
		).
		Order(ent.Asc(ent_asset.FieldID)).
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
	Total      int `json:"total"`
	Pending    int `json:"pending"`
	Processing int `json:"processing"`
	Completed  int `json:"completed"`
	Failed     int `json:"failed"`
	Skipped    int `json:"skipped"`
}

func (q *Querier) GetOCRStats(ctx context.Context) (*OCRStats, error) {
	total, err := q.db.Asset.Query().Count(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	type row struct {
		Status ent_asset.OcrStatus `json:"ocr_status"`
		Count  int                 `json:"count"`
	}

	var rows []row
	err = q.db.Asset.Query().
		GroupBy(ent_asset.FieldOcrStatus).
		Aggregate(ent.Count()).
		Scan(ctx, &rows)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	stats := &OCRStats{Total: total}
	for _, r := range rows {
		switch r.Status {
		case ent_asset.OcrStatusPending:
			stats.Pending = r.Count
		case ent_asset.OcrStatusProcessing:
			stats.Processing = r.Count
		case ent_asset.OcrStatusCompleted:
			stats.Completed = r.Count
		case ent_asset.OcrStatusFailed:
			stats.Failed = r.Count
		case ent_asset.OcrStatusSkipped:
			stats.Skipped = r.Count
		}
	}

	return stats, nil
}
