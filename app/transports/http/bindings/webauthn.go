package bindings

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/Southclaws/dt"
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/labstack/echo/v4"
	"github.com/rs/xid"

	"github.com/Southclaws/storyden/app/resources/account/account_querier"
	waprovider "github.com/Southclaws/storyden/app/services/authentication/provider/webauthn"
	"github.com/Southclaws/storyden/app/services/authentication/session"
	"github.com/Southclaws/storyden/app/transports/http/middleware/session_cookie"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/config"
	"github.com/Southclaws/storyden/internal/infrastructure/cache"
)

const (
	cookieName          = "storyden-webauthn-session"
	webauthnSessionTTL  = 5 * time.Minute
	webauthnCachePrefix = "webauthn-session:"
)

var errNoCookie = fault.New("no webauthn session cookie")

type webauthnSessionContextKey struct{}

// webauthnSessionRef carries the cache key alongside the decoded session data
// so the binding that consumes it can delete it (one-time use) afterwards.
type webauthnSessionRef struct {
	cacheKey string
	data     *webauthn.SessionData
}

type WebAuthn struct {
	cj           *session_cookie.Jar
	si           *session.Issuer
	accountQuery *account_querier.Querier
	wa           *waprovider.Provider
	address      url.URL
	cache        cache.Store
}

func NewWebAuthn(
	cfg config.Config,
	si *session.Issuer,
	accountQuery *account_querier.Querier,
	cj *session_cookie.Jar,
	wa *waprovider.Provider,
	cacheStore cache.Store,
	router *echo.Echo,
) WebAuthn {
	// In order to retain context across the credential request and creation,
	// a cookie carries a random reference to the actual session data, which
	// is kept server-side. The cookie previously held the session data
	// itself (base64 JSON), which a client could freely edit - e.g. to
	// downgrade UserVerification or swap the allowed credential list, since
	// nothing was signed. Only the server can write and read this cache.
	router.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if ck, err := c.Cookie(cookieName); err == nil {
				key := webauthnCachePrefix + ck.Value

				if raw, err := cacheStore.Get(c.Request().Context(), key); err == nil {
					session := &webauthn.SessionData{}
					if err := json.Unmarshal([]byte(raw), session); err == nil {
						r := c.Request()
						ref := webauthnSessionRef{cacheKey: key, data: session}
						ctx := context.WithValue(r.Context(), webauthnSessionContextKey{}, ref)
						c.SetRequest(r.WithContext(ctx))
					}
				}
			}

			return next(c)
		}
	})

	return WebAuthn{cj, si, accountQuery, wa, cfg.PublicAPIAddress, cacheStore}
}

// startWebAuthnSession stores session data server-side under a fresh random
// ID and returns the cookie that carries only that ID to the client.
func (a *WebAuthn) startWebAuthnSession(ctx context.Context, sessionData *webauthn.SessionData) (*http.Cookie, error) {
	id := make([]byte, 32)
	if _, err := rand.Read(id); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}
	value := hex.EncodeToString(id)

	j, err := json.Marshal(sessionData)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	if err := a.cache.Set(ctx, webauthnCachePrefix+value, string(j), webauthnSessionTTL); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return &http.Cookie{
		Name:     cookieName,
		Value:    value,
		Expires:  time.Now().Add(webauthnSessionTTL),
		SameSite: http.SameSiteDefaultMode,
		Path:     "/",
		Domain:   a.address.Hostname(),
		Secure:   true,
		HttpOnly: true,
	}, nil
}

// consumeWebAuthnSession reads the session data stashed by the middleware
// and deletes it from the cache so it cannot be used a second time.
func consumeWebAuthnSession(ctx context.Context, a *WebAuthn) (*webauthn.SessionData, error) {
	ref, ok := ctx.Value(webauthnSessionContextKey{}).(webauthnSessionRef)
	if !ok {
		return nil, fault.Wrap(errNoCookie, fctx.With(ctx), ftag.With(ftag.InvalidArgument))
	}

	_ = a.cache.Delete(ctx, ref.cacheKey)

	return ref.data, nil
}

func (a *WebAuthn) WebAuthnRequestCredential(ctx context.Context, request openapi.WebAuthnRequestCredentialRequestObject) (openapi.WebAuthnRequestCredentialResponseObject, error) {
	cred, sessionData, err := a.wa.BeginRegistration(ctx, string(request.AccountHandle))
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	cookie, err := a.startWebAuthnSession(ctx, sessionData)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.WebAuthnRequestCredential200JSONResponse{
		WebAuthnRequestCredentialOKJSONResponse: openapi.WebAuthnRequestCredentialOKJSONResponse{
			Headers: openapi.WebAuthnRequestCredentialOKResponseHeaders{
				SetCookie: cookie.String(),
			},
			Body: serialiseWebAuthnCredentialCreationOptions(*cred),
		},
	}, nil
}

func (a *WebAuthn) WebAuthnMakeCredential(ctx context.Context, request openapi.WebAuthnMakeCredentialRequestObject) (openapi.WebAuthnMakeCredentialResponseObject, error) {
	session, err := consumeWebAuthnSession(ctx, a)
	if err != nil {
		return nil, err
	}

	// NOTE: This is a hack due to oapi-codegen not giving us raw JSON.

	b, err := json.Marshal(request.Body)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	reader := bytes.NewReader(b)

	cr, err := protocol.ParseCredentialCreationResponseBody(reader)
	if err != nil {
		pe := err.(*protocol.Error)
		ctx = fctx.WithMeta(ctx,
			"type", pe.Type,
			"details", pe.Details,
			"info", pe.DevInfo,
		)
		return nil, fault.Wrap(err, fctx.With(ctx), ftag.With(ftag.InvalidArgument), fmsg.With(pe.DevInfo))
	}

	invitedBy, err := deserialiseInvitationID(request.Params.InvitationId)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	_, accountID, err := a.wa.FinishRegistration(ctx, string(session.UserID), *session, cr, invitedBy)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	t, err := a.si.Issue(ctx, accountID, alwaysRemember())
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.WebAuthnMakeCredential200JSONResponse{
		AuthSuccessOKJSONResponse: openapi.AuthSuccessOKJSONResponse{
			Body: openapi.AuthSuccess{
				Id: xid.NilID().String(),
			},
			Headers: openapi.AuthSuccessOKResponseHeaders{
				SetCookie: a.cj.Create(*t).String(),
			},
		},
	}, nil
}

func (a *WebAuthn) WebAuthnGetAssertion(ctx context.Context, request openapi.WebAuthnGetAssertionRequestObject) (openapi.WebAuthnGetAssertionResponseObject, error) {
	cred, sessionData, err := a.wa.BeginLogin(ctx, string(request.AccountHandle))
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	cookie, err := a.startWebAuthnSession(ctx, sessionData)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.WebAuthnGetAssertion200JSONResponse{
		WebAuthnGetAssertionOKJSONResponse: openapi.WebAuthnGetAssertionOKJSONResponse{
			Body: serialiseWebAuthnCredentialRequestOptions(cred.Response),
			Headers: openapi.WebAuthnGetAssertionOKResponseHeaders{
				SetCookie: cookie.String(),
			},
		},
	}, nil
}

func (a *WebAuthn) WebAuthnMakeAssertion(ctx context.Context, request openapi.WebAuthnMakeAssertionRequestObject) (openapi.WebAuthnMakeAssertionResponseObject, error) {
	session, err := consumeWebAuthnSession(ctx, a)
	if err != nil {
		return nil, err
	}

	// something here is messing up userHandle
	b, err := json.Marshal(request.Body)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	reader := bytes.NewReader(b)

	cr, err := protocol.ParseCredentialRequestResponseBody(reader)
	if err != nil {
		pe := err.(*protocol.Error)
		ctx = fctx.WithMeta(ctx,
			"type", pe.Type,
			"details", pe.Details,
			"info", pe.DevInfo,
		)
		return nil, fault.Wrap(err, fctx.With(ctx), ftag.With(ftag.InvalidArgument), fmsg.With(pe.DevInfo))
	}

	_, acc, err := a.wa.FinishLogin(ctx, string(session.UserID), *session, cr)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	t, err := a.si.Issue(ctx, acc.ID, alwaysRemember())
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.WebAuthnMakeAssertion200JSONResponse{
		AuthSuccessOKJSONResponse: openapi.AuthSuccessOKJSONResponse{
			Body: openapi.AuthSuccess{
				Id: xid.NilID().String(),
			},
			Headers: openapi.AuthSuccessOKResponseHeaders{
				SetCookie: a.cj.Create(*t).String(),
			},
		},
	}, nil
}

func serialiseWebAuthnCredentialCreationOptions(cred protocol.CredentialCreation) openapi.WebAuthnPublicKeyCreationOptions {
	rp := openapi.PublicKeyCredentialRpEntity{
		Id:   cred.Response.RelyingParty.ID,
		Name: cred.Response.RelyingParty.Name,
	}

	user := openapi.PublicKeyCredentialUserEntity{
		DisplayName: cred.Response.User.DisplayName,
		Id:          fmt.Sprint(cred.Response.User.ID),
		Name:        cred.Response.User.Name,
	}

	pubKeyCredParams := dt.Map(cred.Response.Parameters, func(p protocol.CredentialParameter) openapi.PublicKeyCredentialParameters {
		alg := float32(p.Algorithm)
		return openapi.PublicKeyCredentialParameters{
			Type: openapi.PublicKeyCredentialType(p.Type),
			Alg:  alg,
		}
	})

	excludeCredentials := dt.Map(cred.Response.CredentialExcludeList, func(d protocol.CredentialDescriptor) openapi.PublicKeyCredentialDescriptor {
		transports := dt.Map(d.Transport, func(t protocol.AuthenticatorTransport) openapi.PublicKeyCredentialDescriptorTransports {
			return openapi.PublicKeyCredentialDescriptorTransports(t)
		})
		return openapi.PublicKeyCredentialDescriptor{
			Type:       openapi.PublicKeyCredentialType(d.Type),
			Id:         string(d.CredentialID),
			Transports: &transports,
		}
	})

	authenticatorSelection := &openapi.AuthenticatorSelectionCriteria{
		AuthenticatorAttachment: openapi.AuthenticatorAttachment(cred.Response.AuthenticatorSelection.AuthenticatorAttachment),
		RequireResidentKey:      cred.Response.AuthenticatorSelection.RequireResidentKey,
		ResidentKey:             openapi.ResidentKeyRequirement(cred.Response.AuthenticatorSelection.ResidentKey),
		UserVerification:        (*openapi.UserVerificationRequirement)(&cred.Response.AuthenticatorSelection.UserVerification),
	}

	return openapi.WebAuthnPublicKeyCreationOptions{
		PublicKey: openapi.PublicKeyCredentialCreationOptions{
			Rp:   rp,
			User: user,

			Challenge:        cred.Response.Challenge.String(),
			PubKeyCredParams: pubKeyCredParams,

			Timeout:                &cred.Response.Timeout,
			ExcludeCredentials:     excludeCredentials,
			AuthenticatorSelection: authenticatorSelection,
			Attestation:            (*openapi.AttestationConveyancePreference)(&cred.Response.Attestation),
			Extensions:             (*openapi.AuthenticationExtensionsClientInputs)(&cred.Response.Extensions),
		},
	}
}

func serialiseWebAuthnCredentialRequestOptions(cred protocol.PublicKeyCredentialRequestOptions) openapi.CredentialRequestOptions {
	allowedCredentials := dt.Map(cred.AllowedCredentials, func(cd protocol.CredentialDescriptor) openapi.PublicKeyCredentialDescriptor {
		transports := dt.Map(cd.Transport, func(t protocol.AuthenticatorTransport) openapi.PublicKeyCredentialDescriptorTransports {
			return openapi.PublicKeyCredentialDescriptorTransports(t)
		})
		id := make([]byte, base64.RawStdEncoding.EncodedLen(len(cd.CredentialID)))
		base64.RawURLEncoding.Encode(id, cd.CredentialID)
		return openapi.PublicKeyCredentialDescriptor{
			Id:         string(id),
			Transports: &transports,
			Type:       openapi.PublicKeyCredentialType(cd.Type),
		}
	})

	return openapi.CredentialRequestOptions{
		PublicKey: openapi.PublicKeyCredentialRequestOptions{
			AllowCredentials: &allowedCredentials,
			Challenge:        cred.Challenge.String(),
			RpId:             &cred.RelyingPartyID,
			Timeout:          &cred.Timeout,
			UserVerification: (*openapi.PublicKeyCredentialRequestOptionsUserVerification)(&cred.UserVerification),
		},
	}
}
