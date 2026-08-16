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

	token := Token{xid.New()}
	now := time.Now()

	create := r.db.Session.Create().
		SetID(token.ID).
		SetAccountID(xid.ID(accountID)).
		SetExpiresAt(now.Add(o.lifetime)).
		SetRefreshedAt(now).
		SetLifetimeSeconds(int(o.lifetime.Seconds())).
		SetPersistent(o.persistent)

	result, err := create.Save(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return Map(result), nil
}

// Refresh slides the session's expiry forward by its own lifetime. Legacy
// sessions, which have no recorded lifetime, are returned untouched so that
// deploying this cannot shorten sessions issued before it.
func (r *persistedRepository) Refresh(ctx context.Context, t Token) (*Session, error) {
	current, err := r.db.Session.Query().Where(session.ID(t.ID)).Only(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	if current.LifetimeSeconds <= 0 {
		return Map(current), nil
	}

	now := time.Now()
	lifetime := time.Duration(current.LifetimeSeconds) * time.Second

	updated, err := r.db.Session.UpdateOneID(t.ID).
		SetExpiresAt(now.Add(lifetime)).
		SetRefreshedAt(now).
		Save(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return Map(updated), nil
}

func (r *persistedRepository) Revoke(ctx context.Context, id Token) error {
	update := r.db.Session.Update().Where(session.ID(id.ID))

	update.SetRevokedAt(time.Now())

	err := update.Exec(ctx)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	return nil
}

func (r *persistedRepository) Validate(ctx context.Context, t Token) (*Validated, error) {
	query := r.db.Session.Query().Where(session.ID(t.ID))

	result, err := query.Only(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	v, err := Map(result).Validate()
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return v, nil
}

func Map(s *ent.Session) *Session {
	return &Session{
		Token:       Token{s.ID},
		AccountID:   account.AccountID(s.AccountID),
		ExpiresAt:   s.ExpiresAt,
		RevokedAt:   opt.NewPtr(s.RevokedAt),
		RefreshedAt: opt.NewPtr(s.RefreshedAt),
		Lifetime:    time.Duration(s.LifetimeSeconds) * time.Second,
		Persistent:  s.Persistent,
	}
}
