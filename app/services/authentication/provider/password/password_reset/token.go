package password_reset

import (
	"context"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
	"github.com/rs/xid"

	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/app/resources/account/password_reset_token"
	"github.com/Southclaws/storyden/internal/infrastructure/endec"
)

var errMalformedToken = fault.New("missing account_id in token")

type TokenProvider struct {
	endec      endec.EncrypterDecrypter
	resetToken password_reset_token.Repository
}

func NewTokenProvider(endec endec.EncrypterDecrypter, resetToken password_reset_token.Repository) *TokenProvider {
	return &TokenProvider{
		endec:      endec,
		resetToken: resetToken,
	}
}

const (
	accountIDKey       = "account_id"
	resetIDKey         = "rid"
	resetTokenLifespan = time.Hour
)

func (r *TokenProvider) GetResetToken(ctx context.Context, accountID account.AccountID) (string, error) {
	rid := xid.New().String()

	claims := endec.Claims{
		accountIDKey: accountID.String(),
		resetIDKey:   rid,
	}

	token, err := r.endec.Encrypt(claims, resetTokenLifespan)
	if err != nil {
		return "", fault.Wrap(err, fctx.With(ctx))
	}

	if err := r.resetToken.Create(ctx, accountID, rid, time.Now().Add(resetTokenLifespan)); err != nil {
		return "", fault.Wrap(err, fctx.With(ctx))
	}

	return token, nil
}

func (r *TokenProvider) Validate(ctx context.Context, tokenString string) (account.AccountID, error) {
	token, err := r.endec.Decrypt(tokenString)
	if err != nil {
		return account.AccountID{}, fault.Wrap(err,
			fctx.With(ctx),
			ftag.With(ftag.Unauthenticated),
			fmsg.WithDesc("failed to decrypt token", "Your password reset link has expired or is invalid. Please request a new one."),
		)
	}

	rawID, ok := token[accountIDKey]
	if !ok {
		return account.AccountID{}, fault.Wrap(errMalformedToken, fctx.With(ctx), fmsg.With("failed to find account_id in token"))
	}

	stringID, ok := rawID.(string)
	if !ok {
		return account.AccountID{}, fault.Wrap(errMalformedToken, fctx.With(ctx), fmsg.With("failed to convert account_id in token"))
	}

	accountID, err := xid.FromString(stringID)
	if err != nil {
		return account.AccountID{}, fault.Wrap(err, fctx.With(ctx), fmsg.With("failed to parse account_id in token"))
	}

	rawRID, ok := token[resetIDKey]
	if !ok {
		return account.AccountID{}, fault.Wrap(errMalformedToken, fctx.With(ctx), fmsg.With("failed to find rid in token"))
	}

	rid, ok := rawRID.(string)
	if !ok {
		return account.AccountID{}, fault.Wrap(errMalformedToken, fctx.With(ctx), fmsg.With("failed to convert rid in token"))
	}

	// This is what stops the same reset link from being used more than once:
	// the JWT itself is stateless and would otherwise verify successfully
	// forever until its expiry.
	if err := r.resetToken.Consume(ctx, rid); err != nil {
		return account.AccountID{}, fault.Wrap(err,
			fctx.With(ctx),
			ftag.With(ftag.Unauthenticated),
			fmsg.WithDesc("token already used", "Your password reset link has already been used or has expired. Please request a new one."),
		)
	}

	return account.AccountID(accountID), nil
}
