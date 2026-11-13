package asset_writer

import (
	"context"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/rs/xid"

	ent_asset "github.com/Southclaws/storyden/internal/ent/asset"
)

func (w *Writer) SetSummary(ctx context.Context, id xid.ID, html, hash string, quality ent_asset.AiSummaryQuality) error {
	q := w.db.Asset.UpdateOneID(id).
		SetAiSummary(html).
		SetAiSummaryHash(hash).
		SetAiSummaryQuality(quality)
	if quality == ent_asset.AiSummaryQualityFull {
		q = q.SetAiSummaryStatus(ent_asset.AiSummaryStatusDone)
	} else {
		q = q.SetAiSummaryStatus(ent_asset.AiSummaryStatusQueued)
	}
	if err := q.Exec(ctx); err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}
	return nil
}

func (w *Writer) SetSummaryStatus(ctx context.Context, id xid.ID, status ent_asset.AiSummaryStatus) error {
	if err := w.db.Asset.UpdateOneID(id).SetAiSummaryStatus(status).Exec(ctx); err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}
	return nil
}

func (w *Writer) QueueUnsummarisedPDFs(ctx context.Context) (int, error) {
	n, err := w.db.Asset.Update().
		Where(
			ent_asset.AiSummaryStatusEQ(ent_asset.AiSummaryStatusNone),
			ent_asset.MimeTypeEQ("application/pdf"),
			ent_asset.OcrStatusEQ(ent_asset.OcrStatusCompleted),
		).
		SetAiSummaryStatus(ent_asset.AiSummaryStatusQueued).
		Save(ctx)
	if err != nil {
		return 0, fault.Wrap(err, fctx.With(ctx))
	}
	return n, nil
}
