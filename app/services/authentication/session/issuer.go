package session

import (
	"context"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/opt"

	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/app/resources/account/account_querier"
	"github.com/Southclaws/storyden/app/resources/account/token"
	"github.com/Southclaws/storyden/app/resources/settings"
)

// MetadataKey is the account metadata entry holding session preferences.
const MetadataKey = "session"

const metadataRememberMeDays = "remember_me_days"

type Issuer struct {
	tokenRepo    token.Repository
	accountQuery *account_querier.Querier
	settings     *settings.SettingsRepository
}

func NewIssuer(
	tokenRepo token.Repository,
	accountQuery *account_querier.Querier,
	settings *settings.SettingsRepository,
) *Issuer {
	return &Issuer{
		tokenRepo:    tokenRepo,
		accountQuery: accountQuery,
		settings:     settings,
	}
}

// IssueParams controls how long the new session lives.
type IssueParams struct {
	// RememberMe keeps the member signed in beyond the browser session, using
	// their configured duration rather than the instance inactivity timeout.
	RememberMe bool
}

// Issue creates a session for the account.
//
// Without RememberMe the session is a short sliding window that starts again on
// every request, so members are signed out after a period of inactivity rather
// than at a fixed time after signing in.
func (s *Issuer) Issue(ctx context.Context, accountID account.AccountID, params IssueParams) (*token.Session, error) {
	set, err := s.settings.Get(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	lifetime := set.SessionIdleTimeout()

	if params.RememberMe {
		lifetime = set.RememberMeDuration(s.memberPreference(ctx, accountID))
	}

	issued, err := s.tokenRepo.Issue(ctx, accountID,
		token.WithLifetime(lifetime),
		token.WithPersistent(params.RememberMe),
	)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return issued, nil
}

// IssueLike creates a session that inherits the persistence of an existing one.
// Used when re-issuing after a credential change so that a password update does
// not silently downgrade a remembered session.
func (s *Issuer) IssueLike(ctx context.Context, accountID account.AccountID, previous opt.Optional[token.Session]) (*token.Session, error) {
	rememberMe := false
	if p, ok := previous.Get(); ok {
		rememberMe = p.Persistent
	}

	return s.Issue(ctx, accountID, IssueParams{RememberMe: rememberMe})
}

// memberPreference reads the member's chosen session duration. A missing or
// unreadable value simply falls back to the instance default.
func (s *Issuer) memberPreference(ctx context.Context, accountID account.AccountID) opt.Optional[time.Duration] {
	acc, err := s.accountQuery.GetRefByID(ctx, accountID)
	if err != nil {
		return opt.NewEmpty[time.Duration]()
	}

	raw, ok := acc.Metadata[MetadataKey].(map[string]any)
	if !ok {
		return opt.NewEmpty[time.Duration]()
	}

	days, ok := readNumber(raw[metadataRememberMeDays])
	if !ok || days <= 0 {
		return opt.NewEmpty[time.Duration]()
	}

	return opt.New(time.Duration(days) * 24 * time.Hour)
}

// readNumber copes with metadata that has round-tripped through JSON, where
// every number decodes as a float64.
func readNumber(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}
