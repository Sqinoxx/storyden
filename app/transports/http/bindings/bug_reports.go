package bindings

import (
	"context"

	"github.com/Southclaws/dt"
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/ftag"
	"github.com/Southclaws/opt"
	"github.com/rs/xid"

	"github.com/Southclaws/storyden/app/resources/asset"
	"github.com/Southclaws/storyden/app/resources/bug_report"
	"github.com/Southclaws/storyden/app/resources/bug_report/bug_report_querier"
	"github.com/Southclaws/storyden/app/resources/pagination"
	"github.com/Southclaws/storyden/app/resources/rbac"
	"github.com/Southclaws/storyden/app/services/authentication/session"
	bug_report_service "github.com/Southclaws/storyden/app/services/bug_report"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
)

type BugReports struct {
	manager *bug_report_service.Manager
}

func NewBugReports(manager *bug_report_service.Manager) BugReports {
	return BugReports{manager: manager}
}

func (h *BugReports) BugReportCreate(ctx context.Context, request openapi.BugReportCreateRequestObject) (openapi.BugReportCreateResponseObject, error) {
	category, err := bug_report.NewCategory(string(request.Body.Category))
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx), ftag.With(ftag.InvalidArgument))
	}

	screenshot, err := opt.MapErr(opt.NewPtr(request.Body.AssetId), func(id openapi.Identifier) (asset.AssetID, error) {
		return xid.FromString(id)
	})
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx), ftag.With(ftag.InvalidArgument))
	}

	r, err := h.manager.Submit(ctx, bug_report_service.SubmitParams{
		Category:    category,
		Description: request.Body.Description,
		PageURL:     opt.NewPtr(request.Body.PageUrl),
		UserAgent:   opt.NewPtr(request.Body.UserAgent),
		Screenshot:  screenshot,
	})
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.BugReportCreate200JSONResponse{
		BugReportCreateOKJSONResponse: openapi.BugReportCreateOKJSONResponse(serialiseBugReport(r)),
	}, nil
}

func (h *BugReports) BugReportListOwn(ctx context.Context, request openapi.BugReportListOwnRequestObject) (openapi.BugReportListOwnResponseObject, error) {
	category, err := opt.MapErr(opt.NewPtr(request.Params.Category), deserialiseBugReportCategory)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx), ftag.With(ftag.InvalidArgument))
	}

	result, err := h.manager.ListOwn(ctx, deserialisePageParams(request.Params.Page, 20), category)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.BugReportListOwn200JSONResponse{
		BugReportListOKJSONResponse: openapi.BugReportListOKJSONResponse(serialiseBugReportList(result)),
	}, nil
}

func (h *BugReports) AdminBugReportList(ctx context.Context, request openapi.AdminBugReportListRequestObject) (openapi.AdminBugReportListResponseObject, error) {
	if err := session.Authorise(ctx, nil, rbac.PermissionAdministrator); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	status, err := opt.MapErr(opt.NewPtr(request.Params.Status), func(s openapi.BugReportStatusQuery) (bug_report.Status, error) {
		return bug_report.NewStatus(string(s))
	})
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx), ftag.With(ftag.InvalidArgument))
	}

	category, err := opt.MapErr(opt.NewPtr(request.Params.Category), deserialiseBugReportCategory)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx), ftag.With(ftag.InvalidArgument))
	}

	result, err := h.manager.List(ctx, deserialisePageParams(request.Params.Page, 20), bug_report_querier.Filter{
		Status:   status,
		Category: category,
	})
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.AdminBugReportList200JSONResponse{
		BugReportListOKJSONResponse: openapi.BugReportListOKJSONResponse(serialiseBugReportList(result)),
	}, nil
}

func (h *BugReports) AdminBugReportUpdate(ctx context.Context, request openapi.AdminBugReportUpdateRequestObject) (openapi.AdminBugReportUpdateResponseObject, error) {
	if err := session.Authorise(ctx, nil, rbac.PermissionAdministrator); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	id, err := xid.FromString(request.BugReportId)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx), ftag.With(ftag.InvalidArgument))
	}

	status, err := bug_report.NewStatus(string(request.Body.Status))
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx), ftag.With(ftag.InvalidArgument))
	}

	r, err := h.manager.UpdateStatus(ctx, bug_report.ID(id), status)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.AdminBugReportUpdate200JSONResponse{
		BugReportUpdateOKJSONResponse: openapi.BugReportUpdateOKJSONResponse(serialiseBugReport(r)),
	}, nil
}

func (h *BugReports) AdminBugReportDelete(ctx context.Context, request openapi.AdminBugReportDeleteRequestObject) (openapi.AdminBugReportDeleteResponseObject, error) {
	if err := session.Authorise(ctx, nil, rbac.PermissionAdministrator); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	id, err := xid.FromString(request.BugReportId)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx), ftag.With(ftag.InvalidArgument))
	}

	if err := h.manager.Delete(ctx, bug_report.ID(id)); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.AdminBugReportDelete204Response{}, nil
}

func deserialiseBugReportCategory(c openapi.BugReportCategoryQuery) (bug_report.Category, error) {
	return bug_report.NewCategory(string(c))
}

func serialiseBugReport(in *bug_report.BugReport) openapi.BugReport {
	return openapi.BugReport{
		Id:          in.ID.String(),
		CreatedAt:   in.CreatedAt,
		UpdatedAt:   in.UpdatedAt,
		Author:      serialiseProfileReferenceFromAccount(in.Author),
		Category:    openapi.BugReportCategory(in.Category.String()),
		Description: in.Description,
		PageUrl:     in.PageURL.Ptr(),
		UserAgent:   in.UserAgent.Ptr(),
		Status:      openapi.BugReportStatus(in.Status.String()),
		Screenshot:  opt.Map(in.Screenshot, serialiseAsset).Ptr(),
	}
}

func serialiseBugReportList(in pagination.Result[*bug_report.BugReport]) openapi.BugReportListResult {
	return openapi.BugReportListResult{
		BugReports:  dt.Map(in.Items, serialiseBugReport),
		CurrentPage: in.CurrentPage,
		NextPage:    in.NextPage.Ptr(),
		PageSize:    in.Size,
		Results:     in.Results,
		TotalPages:  in.TotalPages,
	}
}
