package bindings

import (
	"context"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/opt"
	"github.com/rs/xid"

	"github.com/Southclaws/storyden/app/services/account/semester"
	"github.com/Southclaws/storyden/app/services/semester_backfill"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
)

func (a *Admin) AdminSemesterBackfillStatus(ctx context.Context, request openapi.AdminSemesterBackfillStatusRequestObject) (openapi.AdminSemesterBackfillStatusResponseObject, error) {
	return openapi.AdminSemesterBackfillStatus200JSONResponse{
		AdminSemesterBackfillOKJSONResponse: serialiseSemesterBackfill(a.semesterBackfill.Status(), 0),
	}, nil
}

func (a *Admin) AdminSemesterBackfillStart(ctx context.Context, request openapi.AdminSemesterBackfillStartRequestObject) (openapi.AdminSemesterBackfillStartResponseObject, error) {
	s, err := a.semesterBackfill.Start(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.AdminSemesterBackfillStart200JSONResponse{
		AdminSemesterBackfillOKJSONResponse: serialiseSemesterBackfill(s, 0),
	}, nil
}

func (a *Admin) AdminSemesterBackfillCancel(ctx context.Context, request openapi.AdminSemesterBackfillCancelRequestObject) (openapi.AdminSemesterBackfillCancelResponseObject, error) {
	return openapi.AdminSemesterBackfillCancel200JSONResponse{
		AdminSemesterBackfillOKJSONResponse: serialiseSemesterBackfill(a.semesterBackfill.Cancel(), 0),
	}, nil
}

func (a *Admin) AdminSemesterBackfillApply(ctx context.Context, request openapi.AdminSemesterBackfillApplyRequestObject) (openapi.AdminSemesterBackfillApplyResponseObject, error) {
	var items []semester_backfill.Item
	for _, it := range request.Body.Items {
		id := openapi.ParseID(it.ThreadId)
		if id == xid.NilID() {
			continue
		}
		term, err := semester.Parse(it.Term)
		if err != nil {
			return nil, fault.Wrap(err, fctx.With(ctx))
		}
		items = append(items, semester_backfill.Item{ThreadID: id, Term: term})
	}

	applied, err := a.semesterBackfill.Apply(ctx, items)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.AdminSemesterBackfillApply200JSONResponse{
		AdminSemesterBackfillOKJSONResponse: serialiseSemesterBackfill(a.semesterBackfill.Status(), applied),
	}, nil
}

func serialiseSemesterBackfill(s semester_backfill.Status, applied int) openapi.AdminSemesterBackfillOKJSONResponse {
	proposals := make([]openapi.SemesterBackfillProposal, len(s.Proposals))
	for i, p := range s.Proposals {
		proposals[i] = openapi.SemesterBackfillProposal{
			ThreadId:        p.ThreadID.String(),
			Slug:            p.Slug,
			Title:           p.Title,
			CreatedAt:       p.CreatedAt,
			Current:         p.Current.String(),
			CurrentExplicit: p.CurrentExplicit,
			Suggested:       p.Suggested.String(),
			Source:          openapi.SemesterBackfillProposalSource(p.Source),
			Evidence:        p.Evidence,
		}
	}

	out := openapi.AdminSemesterBackfillOKJSONResponse{
		Running:   s.Running,
		Checked:   s.Checked,
		Total:     s.Total,
		Failed:    s.Failed,
		Applied:   applied,
		Proposals: proposals,
	}
	if !s.StartedAt.IsZero() {
		out.StartedAt = opt.New(s.StartedAt).Ptr()
	}
	if !s.FinishedAt.IsZero() {
		out.FinishedAt = opt.New(s.FinishedAt).Ptr()
	}
	if s.Error != "" {
		out.Error = opt.New(s.Error).Ptr()
	}
	return out
}
