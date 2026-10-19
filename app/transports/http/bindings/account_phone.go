package bindings

import (
	"context"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"

	"github.com/Southclaws/storyden/app/resources/account/account_phone"
	"github.com/Southclaws/storyden/app/services/authentication/session"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
)

type AccountPhone struct {
	phones *account_phone.Repository
}

func NewAccountPhone(phones *account_phone.Repository) AccountPhone {
	return AccountPhone{phones: phones}
}

func (h *AccountPhone) AccountPhoneGet(ctx context.Context, request openapi.AccountPhoneGetRequestObject) (openapi.AccountPhoneGetResponseObject, error) {
	accountID, err := session.GetAccountID(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	phone, err := h.phones.Get(ctx, accountID)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.AccountPhoneGet200JSONResponse{
		AccountPhoneOKJSONResponse: openapi.AccountPhoneOKJSONResponse{Phone: phone.Ptr()},
	}, nil
}

func (h *AccountPhone) AccountPhoneSet(ctx context.Context, request openapi.AccountPhoneSetRequestObject) (openapi.AccountPhoneSetResponseObject, error) {
	accountID, err := session.GetAccountID(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	phone, err := h.phones.Set(ctx, accountID, request.Body.Phone)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.AccountPhoneSet200JSONResponse{
		AccountPhoneOKJSONResponse: openapi.AccountPhoneOKJSONResponse{Phone: &phone},
	}, nil
}

func (h *AccountPhone) AccountPhoneRemove(ctx context.Context, request openapi.AccountPhoneRemoveRequestObject) (openapi.AccountPhoneRemoveResponseObject, error) {
	accountID, err := session.GetAccountID(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	if err := h.phones.Clear(ctx, accountID); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.AccountPhoneRemove200JSONResponse{}, nil
}
