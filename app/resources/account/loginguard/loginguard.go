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

	"github.com/Southclaws/storyden/app/resources/settings"
	"github.com/Southclaws/storyden/internal/config"
	"github.com/Southclaws/storyden/internal/infrastructure/cache"
)

// KindLockedOut marks a failed-login lockout. The HTTP transport maps this to
// 429 Too Many Requests, distinct from an ordinary bad-credentials failure.
const KindLockedOut ftag.Kind = "LOGIN_LOCKED_OUT"

var ErrLockedOut = fault.New("too many failed login attempts, try again later", ftag.With(KindLockedOut))

const (
	countField            = "count"
	defaultMaxAttempts    = 5
	defaultLockoutTimeout = 15 * time.Minute
)

type Guard struct {
	cache       cache.Store
	settings    *settings.SettingsRepository
	maxAttempts int
	lockout     time.Duration
}

func New(cfg config.Config, cache cache.Store, settings *settings.SettingsRepository) *Guard {
	g := &Guard{
		cache:       cache,
		settings:    settings,
		maxAttempts: defaultMaxAttempts,
		lockout:     defaultLockoutTimeout,
	}
	if cfg.LoginMaxAttempts > 0 {
		g.maxAttempts = cfg.LoginMaxAttempts
	}
	if cfg.LoginLockoutDuration > 0 {
		g.lockout = cfg.LoginLockoutDuration
	}
	return g
}

func (g *Guard) limits(ctx context.Context) (int, time.Duration) {
	maxAttempts, lockout := g.maxAttempts, g.lockout

	s, err := g.settings.Get(ctx)
	if err != nil {
		return maxAttempts, lockout
	}

	rl := s.Services.OrZero().RateLimit.OrZero()
	if v, ok := rl.LoginMaxAttempts.Get(); ok && v > 0 {
		maxAttempts = v
	}
	if v, ok := rl.LoginLockoutDuration.Get(); ok && v > 0 {
		lockout = v
	}

	return maxAttempts, lockout
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

	maxAttempts, _ := g.limits(ctx)

	n, _ := strconv.Atoi(m[countField])
	if n >= maxAttempts {
		return fault.Wrap(ErrLockedOut, fctx.With(ctx))
	}

	return nil
}

// RecordFailure increments the failure counter and restarts the lockout
// window, so a credential stays locked until it has seen no failed attempt
// for the configured lockout duration.
func (g *Guard) RecordFailure(ctx context.Context, identifier string) {
	k := key(identifier)

	if _, err := g.cache.HIncrBy(ctx, k, countField, 1); err != nil {
		return
	}

	_, lockout := g.limits(ctx)

	_ = g.cache.Expire(ctx, k, lockout)
}

// Reset clears the failure counter, e.g. after a successful login.
func (g *Guard) Reset(ctx context.Context, identifier string) {
	_ = g.cache.HDel(ctx, key(identifier), countField)
}
