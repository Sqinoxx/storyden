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

// Filenames are not unique: branding assets (icons, banners) reuse a fixed
// filename and insert a fresh row on every re-upload. Newest-first ordering
// makes the winner deterministic and, since asset IDs are xids, also makes it
// the most recently uploaded one — which is what the stored object holds.
func newestFirst(q *ent.AssetQuery) *ent.AssetQuery {
	return q.Order(ent.Desc(ent_asset.FieldID))
}

func (q *Querier) Get(ctx context.Context, id asset.Filename) (*asset.Asset, error) {
	r, err := newestFirst(q.db.Asset.Query().Where(
		ent_asset.Filename(id.String()),
	)).First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fault.Wrap(err, fctx.With(ctx), ftag.With(ftag.NotFound))
		}
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return asset.Map(r), nil
}

// GetForDownload resolves the metadata needed to serve an asset's bytes without
// pulling ocr_text, which holds up to OCR_MAX_TEXT_LENGTH characters and would
// otherwise be read from the database on every single file request.
func (q *Querier) GetForDownload(ctx context.Context, id asset.Filename) (*asset.Asset, error) {
	r, err := newestFirst(q.db.Asset.Query().Where(
		ent_asset.Filename(id.String()),
	)).Select(
		ent_asset.FieldID,
		ent_asset.FieldFilename,
		ent_asset.FieldMimeType,
		ent_asset.FieldSize,
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

// GetPendingOCR returns the IDs of assets that need text extraction: those
// never processed, those that previously failed, and those stuck in
// `processing` for longer than stuckAfter (e.g. because the process crashed
// mid-run). Only IDs are returned because the processor re-reads each asset
// individually anyway, and selecting whole rows here would drag the ocr_text
// column of an entire batch into memory on every poll.
func (q *Querier) GetPendingOCR(ctx context.Context, limit int, stuckAfter time.Duration) ([]asset.AssetID, error) {
	stuckBefore := time.Now().Add(-stuckAfter)

	ids, err := q.db.Asset.Query().
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
		IDs(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return ids, nil
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
