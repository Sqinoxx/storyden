package bindings

import (
	"context"
	"strconv"

	"github.com/Southclaws/dt"
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/ftag"
	"github.com/Southclaws/opt"

	"github.com/Southclaws/storyden/app/resources/listing"
	"github.com/Southclaws/storyden/app/resources/listing/listing_repo"
	"github.com/Southclaws/storyden/app/services/listing/listing_manager"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
)

const listingsPageSize = 24

type Listings struct {
	manager *listing_manager.Manager
}

func NewListings(manager *listing_manager.Manager) Listings {
	return Listings{manager: manager}
}

func (h *Listings) ListingList(ctx context.Context, request openapi.ListingListRequestObject) (openapi.ListingListResponseObject, error) {
	p := request.Params

	params := listing_manager.ListParams{
		Page:     deserialisePageParams(p.Page, listingsPageSize),
		Sort:     listing_repo.NewSort(string(opt.NewPtr(p.Sort).OrZero())),
		Scope:    listing_manager.Scope(opt.NewPtr(p.Scope).Or(openapi.ListingListParamsScopeAll)),
		Query: opt.NewPtr(p.Q),
	}

	var err error
	if params.PriceMin, err = deserialiseOptionalCents(p.PriceMin); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}
	if params.PriceMax, err = deserialiseOptionalCents(p.PriceMax); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}
	params.Category = deserialiseOptionalCategory(p.Category)
	if params.Condition, err = deserialiseOptionalEnum(p.Condition, listing.NewCondition); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}
	if params.PriceType, err = deserialiseOptionalEnum(p.PriceType, listing.NewPriceType); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	result, err := h.manager.List(ctx, params)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.ListingList200JSONResponse{
		ListingListOKJSONResponse: openapi.ListingListOKJSONResponse{
			CurrentPage: result.CurrentPage,
			NextPage:    result.NextPage.Ptr(),
			PageSize:    result.Size,
			Results:     result.Results,
			TotalPages:  result.TotalPages,
			Listings:    dt.Map(result.Items, serialiseListing),
			PriceMin:    result.PriceMin,
			PriceMax:    result.PriceMax,
		},
	}, nil
}

func (h *Listings) ListingCreate(ctx context.Context, request openapi.ListingCreateRequestObject) (openapi.ListingCreateResponseObject, error) {
	b := request.Body

	partial, err := deserialiseListingPartial(openapi.ListingMutableProps{
		AssetIds:    b.AssetIds,
		Category:    &b.Category,
		Condition:   &b.Condition,
		Description: b.Description,
		Location:    b.Location,
		Price:       &b.Price,
		PriceType:   &b.PriceType,
	})
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	v, err := h.manager.Create(ctx, b.Title, partial)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.ListingCreate200JSONResponse{
		ListingGetOKJSONResponse: openapi.ListingGetOKJSONResponse(serialiseListing(v)),
	}, nil
}

func (h *Listings) ListingGet(ctx context.Context, request openapi.ListingGetRequestObject) (openapi.ListingGetResponseObject, error) {
	v, err := h.manager.Get(ctx, listing.NewKey(request.ListingMark))
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.ListingGet200JSONResponse{
		ListingGetOKJSONResponse: openapi.ListingGetOKJSONResponse(serialiseListing(v)),
	}, nil
}

func (h *Listings) ListingUpdate(ctx context.Context, request openapi.ListingUpdateRequestObject) (openapi.ListingUpdateResponseObject, error) {
	partial, err := deserialiseListingPartial(*request.Body)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	v, err := h.manager.Update(ctx, listing.NewKey(request.ListingMark), partial)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.ListingUpdate200JSONResponse{
		ListingGetOKJSONResponse: openapi.ListingGetOKJSONResponse(serialiseListing(v)),
	}, nil
}

func (h *Listings) ListingDelete(ctx context.Context, request openapi.ListingDeleteRequestObject) (openapi.ListingDeleteResponseObject, error) {
	if err := h.manager.Delete(ctx, listing.NewKey(request.ListingMark)); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.ListingDelete200Response{}, nil
}

func (h *Listings) ListingRenew(ctx context.Context, request openapi.ListingRenewRequestObject) (openapi.ListingRenewResponseObject, error) {
	v, err := h.manager.Renew(ctx, listing.NewKey(request.ListingMark))
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.ListingRenew200JSONResponse{
		ListingGetOKJSONResponse: openapi.ListingGetOKJSONResponse(serialiseListing(v)),
	}, nil
}

func (h *Listings) ListingOfferSubmit(ctx context.Context, request openapi.ListingOfferSubmitRequestObject) (openapi.ListingOfferSubmitResponseObject, error) {
	offer, err := h.manager.MakeOffer(ctx, listing.NewKey(request.ListingMark), request.Body.Amount, opt.NewPtr(request.Body.Message).OrZero())
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.ListingOfferSubmit200JSONResponse{
		ListingOfferOKJSONResponse: openapi.ListingOfferOKJSONResponse(serialiseListingOffer(offer, opt.NewEmpty[string]())),
	}, nil
}

func (h *Listings) ListingContactGet(ctx context.Context, request openapi.ListingContactGetRequestObject) (openapi.ListingContactGetResponseObject, error) {
	c, err := h.manager.Contact(ctx, listing.NewKey(request.ListingMark))
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.ListingContactGet200JSONResponse{
		ListingContactOKJSONResponse: openapi.ListingContactOKJSONResponse{
			Phone:  c.Phone,
			Seller: serialiseProfileReference(c.Seller),
		},
	}, nil
}

func (h *Listings) ListingOfferList(ctx context.Context, request openapi.ListingOfferListRequestObject) (openapi.ListingOfferListResponseObject, error) {
	offers, err := h.manager.Offers(ctx, listing.NewKey(request.ListingMark))
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.ListingOfferList200JSONResponse{
		ListingOfferListOKJSONResponse: openapi.ListingOfferListOKJSONResponse{
			Offers: dt.Map(offers, func(o *listing_manager.OfferWithPhone) openapi.ListingOffer {
				return serialiseListingOffer(o.Offer, o.Phone)
			}),
		},
	}, nil
}

func (h *Listings) ListingFavouriteAdd(ctx context.Context, request openapi.ListingFavouriteAddRequestObject) (openapi.ListingFavouriteAddResponseObject, error) {
	if err := h.manager.SetFavourite(ctx, listing.NewKey(request.ListingMark), true); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.ListingFavouriteAdd200Response{}, nil
}

func (h *Listings) ListingFavouriteRemove(ctx context.Context, request openapi.ListingFavouriteRemoveRequestObject) (openapi.ListingFavouriteRemoveResponseObject, error) {
	if err := h.manager.SetFavourite(ctx, listing.NewKey(request.ListingMark), false); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.ListingFavouriteRemove200Response{}, nil
}

func deserialiseOptionalCents(in *string) (opt.Optional[int], error) {
	if in == nil || *in == "" {
		return opt.NewEmpty[int](), nil
	}

	v, err := strconv.Atoi(*in)
	if err != nil || v < 0 {
		return opt.NewEmpty[int](), fault.New("invalid price", ftag.With(ftag.InvalidArgument))
	}

	return opt.New(v), nil
}

func deserialiseOptionalEnum[In ~string, Out any](in *In, parse func(string) (Out, error)) (opt.Optional[Out], error) {
	if in == nil || *in == "" {
		return opt.NewEmpty[Out](), nil
	}

	v, err := parse(string(*in))
	if err != nil {
		return opt.NewEmpty[Out](), fault.Wrap(err, ftag.With(ftag.InvalidArgument))
	}

	return opt.New(v), nil
}

func deserialiseOptionalCategory[In ~string](in *In) opt.Optional[string] {
	if in == nil || *in == "" {
		return opt.NewEmpty[string]()
	}
	return opt.New(string(*in))
}

func deserialiseListingPartial(in openapi.ListingMutableProps) (listing_manager.Partial, error) {
	var (
		p   listing_manager.Partial
		err error
	)

	p.Category = deserialiseOptionalCategory(in.Category)
	if p.Condition, err = deserialiseOptionalEnum(in.Condition, listing.NewCondition); err != nil {
		return p, err
	}
	if p.PriceType, err = deserialiseOptionalEnum(in.PriceType, listing.NewPriceType); err != nil {
		return p, err
	}
	if p.Status, err = deserialiseOptionalEnum(in.Status, listing.NewStatus); err != nil {
		return p, err
	}

	p.Title = opt.NewPtr(in.Title)
	p.Description = opt.NewPtr(in.Description)
	p.PriceCents = opt.NewPtr(in.Price)
	p.Location = opt.NewPtr(in.Location)
	p.Assets = opt.NewPtrMap(in.AssetIds, deserialiseAssetIDs)

	if pt, ok := p.PriceType.Get(); ok && pt == listing.PriceTypeFree {
		p.PriceCents = opt.New(0)
	}

	return p, nil
}

func serialiseListing(v *listing_manager.View) openapi.Listing {
	return openapi.Listing{
		Id:          v.ID.String(),
		CreatedAt:   v.CreatedAt,
		UpdatedAt:   v.UpdatedAt,
		ExpiresAt:   v.ExpiresAt,
		Title:       v.Title,
		Slug:        v.Slug,
		Description: v.Description,
		Price:       v.PriceCents,
		PriceType:   openapi.ListingPriceType(v.PriceType.String()),
		Category:    v.Category,
		Condition:   openapi.ListingCondition(v.Condition.String()),
		Status:      openapi.ListingStatus(v.Status.String()),
		Location:    v.Location.Ptr(),
		Assets:      dt.Map(v.Assets, serialiseAssetPtr),
		Owner:       serialiseProfileReference(v.Owner),
		IsOwn:       v.IsOwn,
		IsExpired:   v.IsExpired,
		HasOffered:  v.HasOffered,
		IsFavourite: v.IsFavourite,
		OfferCount:  v.OfferCount.Ptr(),
	}
}

func serialiseListingOffer(o *listing.Offer, phone opt.Optional[string]) openapi.ListingOffer {
	return openapi.ListingOffer{
		Id:         o.ID.String(),
		CreatedAt:  o.CreatedAt,
		UpdatedAt:  o.UpdatedAt,
		Amount:     o.AmountCents,
		Message:    o.Message,
		Buyer:      serialiseProfileReference(o.Buyer),
		BuyerPhone: phone.Ptr(),
	}
}
