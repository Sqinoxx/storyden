package asset_writer

import (
	"context"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/ftag"
	"github.com/rs/xid"

	"github.com/Southclaws/storyden/app/resources/asset"
	"github.com/Southclaws/storyden/internal/ent"
	ent_asset "github.com/Southclaws/storyden/internal/ent/asset"
	"github.com/Southclaws/storyden/internal/mime"
)

type Writer struct {
	db *ent.Client
}

func New(db *ent.Client) *Writer {
	return &Writer{db}
}

func (w *Writer) Add(ctx context.Context,
	accountID xid.ID,
	filename asset.Filename,
	size int,
	mt mime.Type,
) (*asset.Asset, error) {
	r, err := w.db.Asset.
		Create().
		SetID(filename.GetID()).
		SetFilename(filename.String()).
		SetSize(size).
		SetMimeType(mt.String()).
		SetAccountID(xid.ID(accountID)).
		Save(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx), ftag.With(ftag.Internal))
	}

	return asset.Map(r), nil
}

func (w *Writer) AddVersion(ctx context.Context,
	accountID xid.ID,
	filename asset.Filename,
	size int,
	mt mime.Type,
	parent asset.AssetID,
) (*asset.Asset, error) {
	r, err := w.db.Asset.
		Create().
		SetID(filename.GetID()).
		SetParentAssetID(parent).
		SetFilename(filename.String()).
		SetSize(size).
		SetMimeType(mt.String()).
		SetAccountID(xid.ID(accountID)).
		Save(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	// The row we just wrote is already complete; only the parent edge is
	// missing. Loading that directly avoids re-reading the new row via an
	// eager-loading query that would round-trip twice more.
	p, err := w.db.Asset.Get(ctx, parent)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	r.Edges.Parent = p

	return asset.Map(r), nil
}

func (w *Writer) Remove(ctx context.Context, accountID xid.ID, id asset.Filename) error {
	q := w.db.Asset.
		Delete().Where(
		ent_asset.Filename(id.String()),
	)

	if _, err := q.Exec(ctx); err != nil {
		return fault.Wrap(err, fctx.With(ctx), ftag.With(ftag.Internal))
	}

	return nil
}

func (w *Writer) UpdateOCRCompleted(ctx context.Context, id xid.ID, text string) (*asset.Asset, error) {
	r, err := w.db.Asset.
		UpdateOneID(id).
		SetOcrStatus(ent_asset.OcrStatusCompleted).
		SetOcrText(text).
		SetOcrProcessedAt(time.Now()).
		ClearOcrError().
		Save(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}
	return asset.Map(r), nil
}

func (w *Writer) UpdateOCRFailed(ctx context.Context, id xid.ID, ocrErr string) (*asset.Asset, error) {
	r, err := w.db.Asset.
		UpdateOneID(id).
		SetOcrStatus(ent_asset.OcrStatusFailed).
		SetOcrError(ocrErr).
		SetOcrProcessedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}
	return asset.Map(r), nil
}

func (w *Writer) UpdateOCRStatus(ctx context.Context, id xid.ID, status ent_asset.OcrStatus) (*asset.Asset, error) {
	r, err := w.db.Asset.
		UpdateOneID(id).
		SetOcrStatus(status).
		Save(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}
	return asset.Map(r), nil
}

// UpdateOCRProcessing marks an asset as currently being processed. Setting
// ocr_processed_at here (not only on completion) is what allows
// asset_querier.GetPendingOCR to reclaim assets stuck in this state after a
// crash mid-run.
func (w *Writer) UpdateOCRProcessing(ctx context.Context, id xid.ID) (*asset.Asset, error) {
	r, err := w.db.Asset.
		UpdateOneID(id).
		SetOcrStatus(ent_asset.OcrStatusProcessing).
		SetOcrProcessedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}
	return asset.Map(r), nil
}

// UpdateOCRSkipped marks an asset as permanently skipped for a known reason
// (unsupported type, oversized, engine unavailable) as distinct from a
// transient failure. Skipped assets are not retried automatically.
func (w *Writer) UpdateOCRSkipped(ctx context.Context, id xid.ID, reason string) (*asset.Asset, error) {
	r, err := w.db.Asset.
		UpdateOneID(id).
		SetOcrStatus(ent_asset.OcrStatusSkipped).
		SetOcrError(reason).
		SetOcrProcessedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}
	return asset.Map(r), nil
}

// ResetOCRForReindex resets a bounded batch of previously-processed
// (completed/failed/skipped) assets back to pending, clearing prior results,
// so they are picked up by the backfill/processor again. Returns the IDs
// that were reset.
func (w *Writer) ResetOCRForReindex(ctx context.Context, limit int) ([]xid.ID, error) {
	ids, err := w.db.Asset.Query().
		Where(ent_asset.OcrStatusIn(
			ent_asset.OcrStatusCompleted,
			ent_asset.OcrStatusFailed,
			ent_asset.OcrStatusSkipped,
		)).
		Order(ent.Asc(ent_asset.FieldID)).
		Limit(limit).
		IDs(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	if len(ids) == 0 {
		return nil, nil
	}

	_, err = w.db.Asset.Update().
		Where(ent_asset.IDIn(ids...)).
		SetOcrStatus(ent_asset.OcrStatusPending).
		ClearOcrError().
		ClearOcrText().
		ClearOcrProcessedAt().
		Save(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return ids, nil
}
