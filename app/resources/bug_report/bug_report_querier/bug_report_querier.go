package bug_report_querier

import (
	"context"

	"github.com/Southclaws/dt"
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/ftag"
	"github.com/Southclaws/opt"
	"github.com/rs/xid"

	"github.com/Southclaws/storyden/app/resources/bug_report"
	"github.com/Southclaws/storyden/app/resources/pagination"
	"github.com/Southclaws/storyden/internal/ent"
	entbugreport "github.com/Southclaws/storyden/internal/ent/bugreport"
)

type Querier struct {
	db *ent.Client
}

func New(db *ent.Client) *Querier {
	return &Querier{db: db}
}

func (q *Querier) List(
	ctx context.Context,
	page pagination.Parameters,
	status opt.Optional[bug_report.Status],
) (pagination.Result[*bug_report.BugReport], error) {
	query := q.db.BugReport.Query()

	status.Call(func(s bug_report.Status) {
		query.Where(entbugreport.Status(s.String()))
	})

	total, err := query.Clone().Count(ctx)
	if err != nil {
		return pagination.Result[*bug_report.BugReport]{}, fault.Wrap(err, fctx.With(ctx))
	}

	result, err := query.
		WithAuthor().
		WithScreenshot().
		Order(ent.Desc(entbugreport.FieldCreatedAt), ent.Desc(entbugreport.FieldID)).
		Limit(page.Limit()).
		Offset(page.Offset()).
		All(ctx)
	if err != nil {
		return pagination.Result[*bug_report.BugReport]{}, fault.Wrap(err, fctx.With(ctx))
	}

	reports, err := dt.MapErr(result, bug_report.Map)
	if err != nil {
		return pagination.Result[*bug_report.BugReport]{}, fault.Wrap(err, fctx.With(ctx))
	}

	return pagination.NewPageResult(page, total, reports), nil
}

func (q *Querier) Get(ctx context.Context, id bug_report.ID) (*bug_report.BugReport, error) {
	r, err := q.db.BugReport.Query().
		Where(entbugreport.ID(xid.ID(id))).
		WithAuthor().
		WithScreenshot().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fault.Wrap(err, fctx.With(ctx), ftag.With(ftag.NotFound))
		}
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return bug_report.Map(r)
}
