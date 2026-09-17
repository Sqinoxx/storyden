package oauth_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Southclaws/storyden/app/services/authentication/provider/oauth"
	"github.com/Southclaws/storyden/internal/config"
	"github.com/Southclaws/storyden/internal/infrastructure/endec"
	"github.com/Southclaws/storyden/internal/infrastructure/endec/jwt"
)

func newEncrypterDecrypter(t *testing.T) endec.EncrypterDecrypter {
	t.Helper()

	ed, err := jwt.New(config.Config{JWTSecret: []byte("00000000000000000000000000000000")})
	require.NoError(t, err)

	return ed
}

func TestOAuthState(t *testing.T) {
	t.Parallel()

	ed := newEncrypterDecrypter(t)

	t.Run("matching_nonce_succeeds", func(t *testing.T) {
		a := assert.New(t)
		r := require.New(t)

		nonce, err := oauth.NewNonce()
		r.NoError(err)

		state, err := oauth.NewState(ed, "/dashboard", nonce)
		r.NoError(err)

		redirect, err := oauth.VerifyState(ed, state, nonce)
		r.NoError(err)
		a.Equal("/dashboard", redirect)
	})

	t.Run("mismatched_nonce_is_rejected", func(t *testing.T) {
		r := require.New(t)

		attackerNonce, err := oauth.NewNonce()
		r.NoError(err)
		state, err := oauth.NewState(ed, "/dashboard", attackerNonce)
		r.NoError(err)

		// The victim's browser has a different (or no) nonce cookie, so a
		// state/code pair obtained from the attacker's own login attempt
		// must not be accepted when replayed into the victim's session
		// (login CSRF) - B9.
		victimNonce, err := oauth.NewNonce()
		r.NoError(err)

		_, err = oauth.VerifyState(ed, state, victimNonce)
		r.ErrorIs(err, oauth.ErrStateMismatch)
	})

	t.Run("empty_cookie_nonce_is_rejected", func(t *testing.T) {
		r := require.New(t)

		nonce, err := oauth.NewNonce()
		r.NoError(err)
		state, err := oauth.NewState(ed, "/dashboard", nonce)
		r.NoError(err)

		_, err = oauth.VerifyState(ed, state, "")
		r.ErrorIs(err, oauth.ErrStateMismatch)
	})

	t.Run("state_missing_nonce_does_not_panic", func(t *testing.T) {
		r := require.New(t)

		// A state built the old way (no nonce claim at all) must be
		// rejected cleanly rather than crashing on a bad type assertion.
		state, err := ed.Encrypt(endec.Claims{"redirect": "/dashboard"}, time.Minute)
		r.NoError(err)

		r.NotPanics(func() {
			_, err := oauth.VerifyState(ed, state, "anything")
			r.Error(err)
		})
	})

	t.Run("state_missing_redirect_is_rejected_not_panicked", func(t *testing.T) {
		r := require.New(t)

		nonce, err := oauth.NewNonce()
		r.NoError(err)

		state, err := ed.Encrypt(endec.Claims{"nonce": nonce}, time.Minute)
		r.NoError(err)

		r.NotPanics(func() {
			_, err := oauth.VerifyState(ed, state, nonce)
			r.Error(err)
		})
	})
}
