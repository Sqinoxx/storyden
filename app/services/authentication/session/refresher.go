package session

import (
	"context"
	"time"

	"github.com/Southclaws/opt"

	"github.com/Southclaws/storyden/app/resources/account/token"
)

// Refresher extends sliding sessions.
type Refresher struct {
	tokenRepo token.Repository
}

func NewRefresher(tokenRepo token.Repository) *Refresher {
	return &Refresher{tokenRepo: tokenRepo}
}

// RefreshIfStale slides a session's expiry forward, but only once per
// token.RefreshInterval so that a busy client does not cause a database write
// per request. Returns the refreshed session only when a write happened, so
// callers know whether a new cookie needs sending.
func (r *Refresher) RefreshIfStale(ctx context.Context, s token.Session) opt.Optional[token.Session] {
	if !s.NeedsRefresh(time.Now(), token.RefreshInterval) {
		return opt.NewEmpty[token.Session]()
	}

	refreshed, err := r.tokenRepo.Refresh(ctx, s.Token)
	if err != nil {
		// A failed refresh is not worth failing the request over; the session
		// stays valid until its current expiry.
		return opt.NewEmpty[token.Session]()
	}

	return opt.New(*refreshed)
}
