package bindings

import (
	"context"

	"github.com/Southclaws/dt"
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"

	"github.com/Southclaws/storyden/app/resources/access"
	"github.com/Southclaws/storyden/app/resources/access/access_repo"
	"github.com/Southclaws/storyden/app/resources/post/category"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
)

type Access struct {
	repo *access_repo.Repository
}

func NewAccess(repo *access_repo.Repository) Access {
	return Access{repo: repo}
}

func (h *Access) AccessPassList(ctx context.Context, request openapi.AccessPassListRequestObject) (openapi.AccessPassListResponseObject, error) {
	passes, err := h.repo.ListPasses(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.AccessPassList200JSONResponse{
		AccessPassListOKJSONResponse: openapi.AccessPassListOKJSONResponse{
			Passes: dt.Map(passes, serialiseAccessPass),
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

func serialiseContentAccess(in *access.Lock) *openapi.ContentAccess {
	if in == nil {
		return nil
	}

	return &openapi.ContentAccess{
		Locked: true,
		Passes: dt.Map(in.Passes, serialiseAccessPassReference),
	}
}
