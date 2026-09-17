// Package loginguard rate-limits repeated failed login attempts against a
// single credential (an email address or handle), independently of the
// per-IP HTTP rate limiter, so that a distributed brute-force attempt against
// one account is still slowed down.
package loginguard

import (
	"context"
	"strconv"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/ftag"

	"github.com/Southclaws/storyden/internal/infrastructure/cache"
)

// KindLockedOut marks a failed-login lockout. The HTTP transport maps this to
// 429 Too Many Requests, distinct from an ordinary bad-credentials failure.
const KindLockedOut ftag.Kind = "LOGIN_LOCKED_OUT"

var ErrLockedOut = fault.New("too many failed login attempts, try again later", ftag.With(KindLockedOut))

const (
	maxAttempts   = 5
	countField    = "count"
	failureWindow = 15 * time.Minute
	baseBackoff   = time.Minute
	maxBackoff    = time.Hour
	maxBackoffExp = 6 // caps baseBackoff*2^6 = 64m, clamped to maxBackoff anyway
)

type Guard struct {
	cache cache.Store
}

func New(cache cache.Store) *Guard {
	return &Guard{cache: cache}
}

func key(identifier string) string {
	return "loginguard:" + identifier
}

// Check returns ErrLockedOut if identifier has failed to authenticate too
// many times recently. identifier should be the credential being attempted
// (e.g. a normalised email address or handle) rather than an account ID, so
// the lockout also covers attempts against identifiers with no account.
func (g *Guard) Check(ctx context.Context, identifier string) error {
	m, err := g.cache.HGetAll(ctx, key(identifier))
	if err != nil || len(m) == 0 {
		return nil
	}

	n, _ := strconv.Atoi(m[countField])
	if n >= maxAttempts {
		return fault.Wrap(ErrLockedOut, fctx.With(ctx))
	}

	return nil
}

// RecordFailure increments the failure counter and, once the threshold is
// exceeded, extends the lockout window exponentially with each further
// failure.
func (g *Guard) RecordFailure(ctx context.Context, identifier string) {
	k := key(identifier)

	n, err := g.cache.HIncrBy(ctx, k, countField, 1)
	if err != nil {
		return
	}

	ttl := failureWindow
	if n > maxAttempts {
		shift := n - maxAttempts
		if shift > maxBackoffExp {
			shift = maxBackoffExp
		}

		if backoff := baseBackoff * time.Duration(int64(1)<<uint(shift)); backoff < maxBackoff {
			ttl = backoff
		} else {
			ttl = maxBackoff
		}
	}

	_ = g.cache.Expire(ctx, k, ttl)
}

// Reset clears the failure counter, e.g. after a successful login.
func (g *Guard) Reset(ctx context.Context, identifier string) {
	_ = g.cache.HDel(ctx, key(identifier), countField)
}
