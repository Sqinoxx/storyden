package token

import (
	"context"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/opt"
	"github.com/rs/xid"

	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/internal/ent"
	"github.com/Southclaws/storyden/internal/ent/predicate"
	"github.com/Southclaws/storyden/internal/ent/session"
)

// Expiry is the fallback session lifetime for callers that do not specify one.
var Expiry = time.Hour

// RefreshInterval bounds how often a sliding session is written back. Without
// it every authenticated request would issue an UPDATE.
const RefreshInterval = 5 * time.Minute

type issueOptions struct {
	lifetime   time.Duration
	persistent bool
}

type IssueOption func(*issueOptions)

// WithLifetime sets the sliding window for the session being issued.
func WithLifetime(d time.Duration) IssueOption {
	return func(o *issueOptions) {
		o.lifetime = d
	}
}

// WithPersistent marks the session as one whose cookie survives the browser
// being closed.
func WithPersistent(v bool) IssueOption {
	return func(o *issueOptions) {
		o.persistent = v
	}
}

type persistedRepository struct {
	db *ent.Client
}

func New(
	db *ent.Client,
) Repository {
	return &persistedRepository{
		db: db,
	}
}

func (r *persistedRepository) Issue(ctx context.Context, accountID account.AccountID, opts ...IssueOption) (*Session, error) {
	o := issueOptions{lifetime: Expiry}
	for _, fn := range opts {
		fn(&o)
	}

	if o.lifetime <= 0 {
		o.lifetime = Expiry
	}

	tok := Generate()
	now := time.Now()

	create := r.db.Session.Create().
		SetTokenHash(tok.Hash()).
		SetAccountID(xid.ID(accountID)).
		SetExpiresAt(now.Add(o.lifetime)).
		SetRefreshedAt(now).
		SetLifetimeSeconds(int(o.lifetime.Seconds())).
		SetPersistent(o.persistent)

	result, err := create.Save(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return Map(result, tok), nil
}

// Refresh slides the session's expiry forward by its own lifetime. Legacy
// sessions, which have no recorded lifetime, are returned untouched so that
// deploying this cannot shorten sessions issued before it.
func (r *persistedRepository) Refresh(ctx context.Context, t Token) (*Session, error) {
	current, err := r.db.Session.Query().Where(session.TokenHash(t.Hash())).Only(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	if current.LifetimeSeconds <= 0 {
		return Map(current, t), nil
	}

	now := time.Now()
	lifetime := time.Duration(current.LifetimeSeconds) * time.Second

	updated, err := current.Update().
		SetExpiresAt(now.Add(lifetime)).
		SetRefreshedAt(now).
		Save(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return Map(updated, t), nil
}

// RevokeAllForAccount revokes every currently-active session belonging to an
// account and returns the hashes of the rows it revoked.
func (r *persistedRepository) RevokeAllForAccount(ctx context.Context, accountID account.AccountID) ([]string, error) {
	pred := []predicate.Session{
		session.AccountID(xid.ID(accountID)),
		session.RevokedAtIsNil(),
	}

	var hashes []string
	if err := r.db.Session.Query().
		Where(pred...).
		Select(session.FieldTokenHash).
		Scan(ctx, &hashes); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	if err := r.db.Session.Update().
		Where(pred...).
		SetRevokedAt(time.Now()).
		Exec(ctx); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return hashes, nil
}

func (r *persistedRepository) Revoke(ctx context.Context, t Token) error {
	update := r.db.Session.Update().Where(session.TokenHash(t.Hash()))

	update.SetRevokedAt(time.Now())

	err := update.Exec(ctx)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	return nil
}

func (r *persistedRepository) Validate(ctx context.Context, t Token) (*Validated, error) {
	query := r.db.Session.Query().Where(session.TokenHash(t.Hash()))

	result, err := query.Only(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	v, err := Map(result, t).Validate()
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return v, nil
}

// Map builds a Session from a database row plus the raw token the caller
// already holds, since only the token's hash is ever persisted.
func Map(s *ent.Session, t Token) *Session {
	return &Session{
		Token:       t,
		AccountID:   account.AccountID(s.AccountID),
		ExpiresAt:   s.ExpiresAt,
		RevokedAt:   opt.NewPtr(s.RevokedAt),
		RefreshedAt: opt.NewPtr(s.RefreshedAt),
		Lifetime:    time.Duration(s.LifetimeSeconds) * time.Second,
		Persistent:  s.Persistent,
	}
}
