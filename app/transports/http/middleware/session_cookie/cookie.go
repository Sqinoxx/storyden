// Package session provides session handling primitives and middleware for the
// API. Sessions work by encrypting an account's ID inside a cookie value. This
// is read via a middleware and dropped into the request context for later use.
package session_cookie

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/opt"
	"github.com/Southclaws/storyden/app/resources/account/token"
	"github.com/Southclaws/storyden/app/services/authentication/session"
	"github.com/Southclaws/storyden/internal/config"
)

// TODO: Allow changing this via config.
const (
	secureCookieName = "storyden-session"
	sameSiteMode     = http.SameSiteLaxMode
)

type Jar struct {
	validator        *session.Validator
	refresher        *session.Refresher
	domain           string
	secureCookieName string
}

func New(cfg config.Config, v *session.Validator, refresher *session.Refresher) (*Jar, error) {
	domain, err := getCookieDomain(cfg.PublicAPIAddress, cfg.PublicWebAddress)
	if err != nil {
		return nil, fault.Wrap(err, fmsg.With("failed to parse domain from public API address"))
	}

	return &Jar{
		domain:           domain,
		validator:        v,
		refresher:        refresher,
		secureCookieName: secureCookieName,
	}, nil
}

// createWithValue builds the session cookie. An empty expiry produces a browser
// session cookie, which is deliberately not persisted to disk so that closing
// the browser signs the member out.
func (j *Jar) createWithValue(value string, expire opt.Optional[time.Time]) *http.Cookie {
	cookie := &http.Cookie{
		Name:     secureCookieName,
		Value:    value,
		SameSite: sameSiteMode,
		Path:     "/",
		Domain:   j.domain,

		// Always secure, localhost is automatically excluded by browsers.
		Secure: true,

		// JS never needs to access these cookies.
		HttpOnly: true,
	}

	if expires, ok := expire.Get(); ok {
		cookie.Expires = expires
		cookie.MaxAge = max(1, int(time.Until(expires).Seconds()))
	}

	return cookie
}

// Create builds the cookie for a freshly issued or refreshed session.
func (j *Jar) Create(s token.Session) *http.Cookie {
	expiry := opt.NewEmpty[time.Time]()
	if s.Persistent {
		expiry = opt.New(s.ExpiresAt)
	}

	return j.createWithValue(s.Token.String(), expiry)
}

func (j *Jar) Destroy() *http.Cookie {
	cookie := j.createWithValue("", opt.New(time.Unix(0, 0)))

	// MaxAge<0 is what actually removes a browser session cookie; a past
	// Expires alone is not reliable for one.
	cookie.MaxAge = -1

	return cookie
}

// withSession checks the request for a session via either a cookie (for browser
// requests) or a bearer token access key (for API requests).
//
// The writer is needed because a cookie session slides its expiry forward on
// activity, which means re-issuing the cookie.
func (j *Jar) withSession(w http.ResponseWriter, r *http.Request) context.Context {
	if ctx, ok := j.tryFromCookie(w, r); ok {
		return ctx
	}

	if ctx, ok := j.tryFromHeader(r); ok {
		return ctx
	}

	return j.withDefaultRoles(r)
}

func (j *Jar) tryFromCookie(w http.ResponseWriter, r *http.Request) (context.Context, bool) {
	cookie, err := r.Cookie(secureCookieName)
	if err != nil {
		return r.Context(), false
	}

	ctx, validated, err := j.validator.ValidateSessionToken(r.Context(), cookie.Value)
	if err != nil {
		return r.Context(), false
	}

	// Extending the window is what makes this an inactivity timeout rather than
	// a hard cap from sign-in. Headers must be written before the handler runs.
	if refreshed, ok := j.refresher.RefreshIfStale(ctx, token.Session(*validated)).Get(); ok {
		http.SetCookie(w, j.Create(refreshed))
	}

	return ctx, true
}

func (j *Jar) tryFromHeader(r *http.Request) (context.Context, bool) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return r.Context(), false
	}

	// The header should be in the format "Bearer <token>".
	raw, ok := strings.CutPrefix(authHeader, "Bearer ")
	if !ok {
		return r.Context(), false
	}

	var (
		ctx context.Context
		err error
	)
	if looksLikeJWT(raw) {
		ctx, err = j.validator.ValidateOAuthToken(r.Context(), raw)
	} else {
		ctx, err = j.validator.ValidateAccessKeyToken(r.Context(), raw)
	}
	if err != nil {
		return r.Context(), false
	}

	return ctx, true
}

func looksLikeJWT(raw string) bool {
	return strings.HasPrefix(raw, "ey")
}

func (j *Jar) withDefaultRoles(r *http.Request) context.Context {
	ctx, err := j.validator.WithUnauthenticatedRoles(r.Context())
	if err != nil {
		// TODO: Handle this somehow - needs logging but if this fails then
		// we can't do anything, the request would have no roles and thus fail.
		return r.Context()
	}

	return ctx
}

// WithAuth simply pulls out the session from the cookie and propagates it.
func (j *Jar) WithAuth() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := j.withSession(w, r)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (j *Jar) GetCookieName() string {
	return j.secureCookieName
}
