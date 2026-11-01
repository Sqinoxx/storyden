package bindings

import (
	"context"
	"time"

	"github.com/Southclaws/dt"
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/opt"

	"github.com/Southclaws/storyden/app/resources/access"
	"github.com/Southclaws/storyden/app/resources/access/access_repo"
	"github.com/Southclaws/storyden/app/resources/post/category"
	"github.com/Southclaws/storyden/app/services/access/access_manage"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
)

type Access struct {
	repo    *access_repo.Repository
	manager *access_manage.Manager
}

func NewAccess(repo *access_repo.Repository, manager *access_manage.Manager) Access {
	return Access{repo: repo, manager: manager}
}

func (h *Access) AccessPassList(ctx context.Context, request openapi.AccessPassListRequestObject) (openapi.AccessPassListResponseObject, error) {
	passes, err := h.manager.ListPasses(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.AccessPassList200JSONResponse{
		AccessPassListOKJSONResponse: openapi.AccessPassListOKJSONResponse{
			Passes: dt.Map(passes, serialiseAccessPassWithMembership),
		},
	}, nil
}

func (h *Access) AccessPassRequest(ctx context.Context, request openapi.AccessPassRequestRequestObject) (openapi.AccessPassRequestResponseObject, error) {
	p, err := h.manager.Request(ctx, access.PassID(openapi.ParseID(request.AccessPassId)))
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.AccessPassRequest200JSONResponse{
		AccessPassGetOKJSONResponse: openapi.AccessPassGetOKJSONResponse(serialiseAccessPassWithMembership(p)),
	}, nil
}

func (h *Access) AccessPassRequestWithdraw(ctx context.Context, request openapi.AccessPassRequestWithdrawRequestObject) (openapi.AccessPassRequestWithdrawResponseObject, error) {
	p, err := h.manager.Withdraw(ctx, access.PassID(openapi.ParseID(request.AccessPassId)))
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.AccessPassRequestWithdraw200JSONResponse{
		AccessPassGetOKJSONResponse: openapi.AccessPassGetOKJSONResponse(serialiseAccessPassWithMembership(p)),
	}, nil
}

func (h *Access) AccessGrantSet(ctx context.Context, request openapi.AccessGrantSetRequestObject) (openapi.AccessGrantSetResponseObject, error) {
	params := access_manage.GrantParams{}
	if request.Body != nil {
		params.ExpiresAt = opt.NewPtr(request.Body.ExpiresAt)
		params.Note = opt.NewPtr(request.Body.Note)
	}

	g, err := h.manager.Grant(ctx, string(request.AccountHandle), access.PassID(openapi.ParseID(request.AccessPassId)), params)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.AccessGrantSet200JSONResponse{
		AccessGrantOKJSONResponse: openapi.AccessGrantOKJSONResponse(serialiseAccessGrant(g)),
	}, nil
}

func (h *Access) AccessGrantRemove(ctx context.Context, request openapi.AccessGrantRemoveRequestObject) (openapi.AccessGrantRemoveResponseObject, error) {
	g, err := h.manager.Remove(ctx,
		string(request.AccountHandle),
		access.PassID(openapi.ParseID(request.AccessPassId)),
		access_manage.RemoveAction(request.Params.Action),
	)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.AccessGrantRemove200JSONResponse{
		AccessGrantOKJSONResponse: openapi.AccessGrantOKJSONResponse(serialiseAccessGrant(g)),
	}, nil
}

func (h *Access) AccessGrantList(ctx context.Context, request openapi.AccessGrantListRequestObject) (openapi.AccessGrantListResponseObject, error) {
	params := access_manage.ListParams{
		Status: opt.Map(opt.NewPtr(request.Params.Status), func(s openapi.AccessGrantStatus) string { return string(s) }),
		PassID: opt.Map(opt.NewPtr(request.Params.PassId), func(id openapi.Identifier) access.PassID {
			return access.PassID(openapi.ParseID(id))
		}),
		Handle: opt.Map(opt.NewPtr(request.Params.AccountHandle), func(h openapi.AccountHandle) string { return string(h) }),
	}

	grants, err := h.manager.ListGrants(ctx, params)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.AccessGrantList200JSONResponse{
		AccessGrantListOKJSONResponse: openapi.AccessGrantListOKJSONResponse{
			Grants: dt.Map(grants, serialiseAccessGrant),
		},
	}, nil
}

func (h *Access) AccessPassCreate(ctx context.Context, request openapi.AccessPassCreateRequestObject) (openapi.AccessPassCreateResponseObject, error) {
	b := request.Body

	opts := accessPassOptions(b.Description, b.PriceLabel, b.PaymentInfo, b.Colour, b.Sort, b.Requestable, b.CategoryIds)

	p, err := h.repo.CreatePass(ctx, b.Name, opts...)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.AccessPassCreate200JSONResponse{
		AccessPassGetOKJSONResponse: openapi.AccessPassGetOKJSONResponse(serialiseAccessPass(p)),
	}, nil
}

func (h *Access) AccessPassUpdate(ctx context.Context, request openapi.AccessPassUpdateRequestObject) (openapi.AccessPassUpdateResponseObject, error) {
	b := request.Body

	opts := accessPassOptions(b.Description, b.PriceLabel, b.PaymentInfo, b.Colour, b.Sort, b.Requestable, b.CategoryIds)
	if b.Name != nil {
		opts = append(opts, access_repo.WithName(*b.Name))
	}

	p, err := h.repo.UpdatePass(ctx, access.PassID(openapi.ParseID(request.AccessPassId)), opts...)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.AccessPassUpdate200JSONResponse{
		AccessPassGetOKJSONResponse: openapi.AccessPassGetOKJSONResponse(serialiseAccessPass(p)),
	}, nil
}

func (h *Access) AccessPassDelete(ctx context.Context, request openapi.AccessPassDeleteRequestObject) (openapi.AccessPassDeleteResponseObject, error) {
	if err := h.repo.DeletePass(ctx, access.PassID(openapi.ParseID(request.AccessPassId))); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.AccessPassDelete200Response{}, nil
}

func accessPassOptions(
	description, priceLabel, paymentInfo, colour *string,
	sort *int,
	requestable *bool,
	categoryIDs *[]openapi.Identifier,
) []access_repo.PassOption {
	opts := []access_repo.PassOption{}

	if description != nil {
		opts = append(opts, access_repo.WithDescription(*description))
	}
	if priceLabel != nil {
		opts = append(opts, access_repo.WithPriceLabel(*priceLabel))
	}
	if paymentInfo != nil {
		opts = append(opts, access_repo.WithPaymentInfo(*paymentInfo))
	}
	if colour != nil {
		opts = append(opts, access_repo.WithColour(*colour))
	}
	if sort != nil {
		opts = append(opts, access_repo.WithSort(*sort))
	}
	if requestable != nil {
		opts = append(opts, access_repo.WithRequestable(*requestable))
	}
	if categoryIDs != nil {
		opts = append(opts, access_repo.WithCategories(dt.Map(*categoryIDs, func(id openapi.Identifier) category.CategoryID {
			return category.CategoryID(openapi.ParseID(id))
		})))
	}

	return opts
}

func serialiseAccessPassReference(in access.PassRef) openapi.AccessPassReference {
	return openapi.AccessPassReference{
		Id:         openapi.Identifier(in.ID.String()),
		Name:       in.Name,
		PriceLabel: in.PriceLabel,
		Colour:     in.Colour,
	}
}

func serialiseAccessPass(in *access.Pass) openapi.AccessPass {
	return openapi.AccessPass{
		Id:          openapi.Identifier(in.ID.String()),
		Name:        in.Name,
		PriceLabel:  in.PriceLabel,
		Colour:      in.Colour,
		Description: in.Description,
		PaymentInfo: in.PaymentInfo,
		Sort:        in.Sort,
		Requestable: in.Requestable,
		CategoryIds: dt.Map(in.Categories, func(id category.CategoryID) openapi.Identifier {
			return openapi.Identifier(id.String())
		}),
		CreatedAt: in.CreatedAt,
		UpdatedAt: in.UpdatedAt,
	}
}

func serialiseAccessPassWithMembership(in *access_manage.PassWithMembership) openapi.AccessPass {
	out := serialiseAccessPass(in.Pass)
	out.Membership = opt.Map(in.Membership, func(g access.Grant) openapi.AccessPassMembership {
		return openapi.AccessPassMembership{
			Status:        openapi.AccessGrantStatus(g.DisplayStatus(time.Now())),
			ReferenceCode: g.ReferenceCode,
			ExpiresAt:     g.ExpiresAt.Ptr(),
			RequestedAt:   g.RequestedAt.Ptr(),
			DecidedAt:     g.DecidedAt.Ptr(),
		}
	}).Ptr()
	return out
}

func serialiseAccessGrant(in *access.Grant) openapi.AccessGrant {
	return openapi.AccessGrant{
		Id:            openapi.Identifier(in.ID.String()),
		Account:       serialiseProfileReference(in.Account.OrZero()),
		Pass:          serialiseAccessPassReference(in.Pass.OrZero()),
		Status:        openapi.AccessGrantStatus(in.DisplayStatus(time.Now())),
		ReferenceCode: in.ReferenceCode,
		ExpiresAt:     in.ExpiresAt.Ptr(),
		RequestedAt:   in.RequestedAt.Ptr(),
		DecidedAt:     in.DecidedAt.Ptr(),
		DecidedBy:     opt.Map(in.DecidedByMember, serialiseProfileReference).Ptr(),
		Note:          in.Note,
		CreatedAt:     in.CreatedAt,
		UpdatedAt:     in.UpdatedAt,
	}
}

func serialiseContentAccess(in *access.Lock) *openapi.ContentAccess {
	if in == nil {
		return nil
	}

	return &openapi.ContentAccess{
		Locked: true,
		Passes: dt.Map(in.Passes, serialiseAccessPassReference),
	}
}
