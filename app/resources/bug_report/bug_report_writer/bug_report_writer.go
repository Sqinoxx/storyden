package bug_report_writer

import (
	"context"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/ftag"
	"github.com/Southclaws/opt"
	"github.com/rs/xid"

	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/app/resources/asset"
	"github.com/Southclaws/storyden/app/resources/bug_report"
	"github.com/Southclaws/storyden/app/resources/bug_report/bug_report_querier"
	"github.com/Southclaws/storyden/internal/ent"
)

type Writer struct {
	db      *ent.Client
	querier *bug_report_querier.Querier
}

func New(db *ent.Client, querier *bug_report_querier.Querier) *Writer {
	return &Writer{db: db, querier: querier}
}

type CreateParams struct {
	Category    bug_report.Category
	Description string
	PageURL     opt.Optional[string]
	UserAgent   opt.Optional[string]
	Screenshot  opt.Optional[asset.AssetID]
}

func (w *Writer) Create(ctx context.Context, author account.AccountID, p CreateParams) (*bug_report.BugReport, error) {
	create := w.db.BugReport.Create().
		SetAccountID(xid.ID(author)).
		SetCategory(p.Category.String()).
		SetDescription(p.Description).
		SetStatus(bug_report.StatusOpen.String())

	p.PageURL.Call(func(v string) { create.SetPageURL(v) })
	p.UserAgent.Call(func(v string) { create.SetUserAgent(v) })
	p.Screenshot.Call(func(v asset.AssetID) { create.SetAssetID(v) })

	r, err := create.Save(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return w.querier.Get(ctx, bug_report.ID(r.ID))
}

func (w *Writer) UpdateStatus(ctx context.Context, id bug_report.ID, status bug_report.Status) (*bug_report.BugReport, error) {
	err := w.db.BugReport.UpdateOneID(xid.ID(id)).
		SetStatus(status.String()).
		Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fault.Wrap(err, fctx.With(ctx), ftag.With(ftag.NotFound))
		}
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return w.querier.Get(ctx, id)
}

func (w *Writer) Delete(ctx context.Context, id bug_report.ID) error {
	err := w.db.BugReport.DeleteOneID(xid.ID(id)).Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return fault.Wrap(err, fctx.With(ctx), ftag.With(ftag.NotFound))
		}
		return fault.Wrap(err, fctx.With(ctx))
	}

	return nil
}
