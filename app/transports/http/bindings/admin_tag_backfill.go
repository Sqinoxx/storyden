package bindings

import (
	"context"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/opt"
	"github.com/rs/xid"

	"github.com/Southclaws/storyden/app/services/tag/tag_backfill"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
)

func (a *Admin) AdminTagBackfillStatus(ctx context.Context, request openapi.AdminTagBackfillStatusRequestObject) (openapi.AdminTagBackfillStatusResponseObject, error) {
	return openapi.AdminTagBackfillStatus200JSONResponse{
		AdminTagBackfillOKJSONResponse: serialiseTagBackfill(a.tagBackfill.Status(), 0),
	}, nil
}

func (a *Admin) AdminTagBackfillStart(ctx context.Context, request openapi.AdminTagBackfillStartRequestObject) (openapi.AdminTagBackfillStartResponseObject, error) {
	s, err := a.tagBackfill.Start(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.AdminTagBackfillStart200JSONResponse{
		AdminTagBackfillOKJSONResponse: serialiseTagBackfill(s, 0),
	}, nil
}

func (a *Admin) AdminTagBackfillCancel(ctx context.Context, request openapi.AdminTagBackfillCancelRequestObject) (openapi.AdminTagBackfillCancelResponseObject, error) {
	return openapi.AdminTagBackfillCancel200JSONResponse{
		AdminTagBackfillOKJSONResponse: serialiseTagBackfill(a.tagBackfill.Cancel(), 0),
	}, nil
}

func (a *Admin) AdminTagBackfillApply(ctx context.Context, request openapi.AdminTagBackfillApplyRequestObject) (openapi.AdminTagBackfillApplyResponseObject, error) {
	var items []tag_backfill.Item
	for _, it := range request.Body.Items {
		id := openapi.ParseID(it.ThreadId)
		if id == xid.NilID() {
			continue
		}
		items = append(items, tag_backfill.Item{ThreadID: id, Tags: it.Tags})
	}

	applied, err := a.tagBackfill.Apply(ctx, items)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.AdminTagBackfillApply200JSONResponse{
		AdminTagBackfillOKJSONResponse: serialiseTagBackfill(a.tagBackfill.Status(), applied),
	}, nil
}

func serialiseTagBackfill(s tag_backfill.Status, applied int) openapi.AdminTagBackfillOKJSONResponse {
	proposals := make([]openapi.TagBackfillProposal, len(s.Proposals))
	for i, p := range s.Proposals {
		current := p.Current
		if current == nil {
			current = []string{}
		}
		proposals[i] = openapi.TagBackfillProposal{
			ThreadId: p.ThreadID.String(),
			Slug:     p.Slug,
			Title:    p.Title,
			Current:  current,
			Add:      p.Add,
		}
	}

	out := openapi.AdminTagBackfillOKJSONResponse{
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
