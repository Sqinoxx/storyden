// Package password_reset_token tracks the jti of every password-reset token
// issued, so that a token - although it is a stateless, self-verifying JWT -
// can be consumed exactly once and rejected on any later attempt.
package password_reset_token

import (
	"context"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/ftag"
	"github.com/rs/xid"

	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/internal/ent"
	"github.com/Southclaws/storyden/internal/ent/passwordresettoken"
)

var ErrAlreadyUsed = fault.New("password reset token already used or expired", ftag.With(ftag.Unauthenticated))

type Repository interface {
	// Create records a newly issued reset token's jti so it can later be
	// consumed exactly once.
	Create(ctx context.Context, accountID account.AccountID, jti string, expiresAt time.Time) error

	// Consume marks a token used, failing if it does not exist, has already
	// been used, or has expired.
	Consume(ctx context.Context, jti string) error

	// RevokeAllForAccount marks every still-open reset token for an account
	// as used, e.g. once the account's password has actually been changed.
	RevokeAllForAccount(ctx context.Context, accountID account.AccountID) error
}

type repository struct {
	db *ent.Client
}

func New(db *ent.Client) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, accountID account.AccountID, jti string, expiresAt time.Time) error {
	err := r.db.PasswordResetToken.Create().
		SetJti(jti).
		SetAccountID(xid.ID(accountID)).
		SetExpiresAt(expiresAt).
		Exec(ctx)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	return nil
}

func (r *repository) Consume(ctx context.Context, jti string) error {
	affected, err := r.db.PasswordResetToken.Update().
		Where(
			passwordresettoken.Jti(jti),
			passwordresettoken.UsedAtIsNil(),
			passwordresettoken.ExpiresAtGT(time.Now()),
		).
		SetUsedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	if affected == 0 {
		return fault.Wrap(ErrAlreadyUsed, fctx.With(ctx))
	}

	return nil
}

func (r *repository) RevokeAllForAccount(ctx context.Context, accountID account.AccountID) error {
	err := r.db.PasswordResetToken.Update().
		Where(
			passwordresettoken.AccountID(xid.ID(accountID)),
			passwordresettoken.UsedAtIsNil(),
		).
		SetUsedAt(time.Now()).
		Exec(ctx)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	return nil
}
