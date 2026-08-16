package bindings

import (
	"context"
	"errors"
	"net/mail"
	"strings"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
	"github.com/Southclaws/opt"

	"github.com/Southclaws/storyden/app/services/authentication/email_verify"
	"github.com/Southclaws/storyden/app/services/authentication/session"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
)

func (i *Authentication) AuthEmailVerify(ctx context.Context, request openapi.AuthEmailVerifyRequestObject) (openapi.AuthEmailVerifyResponseObject, error) {
	email, err := mail.ParseAddress(strings.ToLower(request.Body.Email))
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx), ftag.With(ftag.InvalidArgument))
	}

	acc, err := i.emailVerifier.Verify(ctx, *email, request.Body.Code)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	t, err := i.si.Issue(ctx, acc.ID, rememberMe(request.Body.RememberMe))
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.AuthEmailVerify200JSONResponse{
		AuthSuccessOKJSONResponse: openapi.AuthSuccessOKJSONResponse{
			Body: openapi.AuthSuccessOK{Id: acc.ID.String()},
			Headers: openapi.AuthSuccessOKResponseHeaders{
				SetCookie: i.cj.Create(*t).String(),
			},
		},
	}, nil
}

func (i *Authentication) AuthEmailVerifyResend(ctx context.Context, request openapi.AuthEmailVerifyResendRequestObject) (openapi.AuthEmailVerifyResendResponseObject, error) {
	address, err := i.resolveResendAddress(ctx, request.Body.Email)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	// Every outcome past this point reports success so that the endpoint cannot
	// be used to discover which addresses have accounts.
	if addr, ok := address.Get(); ok {
		if err := i.emailVerifier.ResendVerification(ctx, addr); err != nil {
			if !errors.Is(err, email_verify.ErrAccountNotFound) {
				return nil, fault.Wrap(err, fctx.With(ctx))
			}

			i.logger.InfoContext(ctx, "verification resend requested for unknown address")
		}
	}

	return openapi.AuthEmailVerifyResend200Response{}, nil
}

// resolveResendAddress prefers the address in the request body and otherwise
// falls back to the session account's first unverified address.
func (i *Authentication) resolveResendAddress(ctx context.Context, requested *openapi.EmailAddress) (opt.Optional[mail.Address], error) {
	if requested != nil && *requested != "" {
		address, err := mail.ParseAddress(strings.ToLower(string(*requested)))
		if err != nil {
			return nil, fault.Wrap(err, fctx.With(ctx), ftag.With(ftag.InvalidArgument))
		}

		return opt.New(*address), nil
	}

	accountID, err := session.GetAccountID(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx), ftag.With(ftag.InvalidArgument),
			fmsg.WithDesc("no address", "An email address is required when not signed in."))
	}

	acc, err := i.accountQuery.GetByID(ctx, accountID)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	for _, e := range acc.EmailAddresses {
		if !e.Verified {
			return opt.New(e.Email), nil
		}
	}

	return opt.NewEmpty[mail.Address](), nil
}
