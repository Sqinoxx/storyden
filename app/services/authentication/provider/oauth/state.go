package oauth

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"

	"github.com/Southclaws/storyden/internal/infrastructure/endec"
)

const (
	stateLifespan = 10 * time.Minute
	redirectKey   = "redirect"
	nonceKey      = "nonce"
)

// ErrStateMismatch means the state's nonce didn't match the one from the
// cookie set when the flow started - either a stale/replayed link, or an
// attempt to complete one browser's OAuth flow (e.g. an attacker's) inside
// another (the victim's), a login CSRF.
var ErrStateMismatch = fault.New("oauth state does not match this browser session", ftag.With(ftag.Unauthenticated))

// NewNonce generates a fresh anti-CSRF nonce for a login attempt. The same
// value must be set in the storyden-oauth-state cookie and embedded in every
// OAuth provider's state via NewState.
func NewNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fault.Wrap(err)
	}

	return hex.EncodeToString(b), nil
}

// NewState builds the signed value handed to the OAuth provider as `state`,
// binding it to the nonce from the caller's storyden-oauth-state cookie.
func NewState(ed endec.EncrypterDecrypter, redirect, nonce string) (string, error) {
	state, err := ed.Encrypt(endec.Claims{
		redirectKey: redirect,
		nonceKey:    nonce,
	}, stateLifespan)
	if err != nil {
		return "", fault.Wrap(err)
	}

	return state, nil
}

// VerifyState decrypts a state value, checks its nonce against the one from
// the browser's cookie, and returns the original redirect path once both
// have been confirmed.
func VerifyState(ed endec.EncrypterDecrypter, state, cookieNonce string) (string, error) {
	claims, err := ed.Decrypt(state)
	if err != nil {
		return "", fault.Wrap(err, fmsg.WithDesc("failed to decrypt state value", "This link has expired, please try again."))
	}

	redirect, ok := claims[redirectKey].(string)
	if !ok {
		return "", fault.New("missing or invalid redirect in oauth state", ftag.With(ftag.InvalidArgument))
	}

	stateNonce, ok := claims[nonceKey].(string)
	if !ok || stateNonce == "" || cookieNonce == "" || stateNonce != cookieNonce {
		return "", ErrStateMismatch
	}

	return redirect, nil
}
