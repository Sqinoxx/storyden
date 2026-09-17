package bindings

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Southclaws/storyden/internal/infrastructure/cache/cachetest"
)

func TestWebAuthnSessionCache(t *testing.T) {
	t.Parallel()

	a := &WebAuthn{
		address: url.URL{Host: "api.storyden.test"},
		cache:   cachetest.New(),
	}

	original := &webauthn.SessionData{
		Challenge:        "a-challenge",
		UserID:           []byte("user-id"),
		UserVerification: "required",
	}

	t.Run("cookie_carries_only_an_opaque_reference", func(t *testing.T) {
		r := require.New(t)

		cookie, err := a.startWebAuthnSession(context.Background(), original)
		r.NoError(err)

		// B10: the cookie must not contain the session data (challenge,
		// user verification requirement, etc) in any decodable form,
		// otherwise a client could edit it - e.g. downgrade
		// UserVerification from "required" to "discouraged".
		var probe map[string]any
		r.Error(json.Unmarshal([]byte(cookie.Value), &probe),
			"cookie value decoded as JSON; it should be an opaque reference")
	})

	t.Run("session_round_trips_and_is_single_use", func(t *testing.T) {
		r := require.New(t)
		as := assert.New(t)

		store := cachetest.New()
		a := &WebAuthn{address: url.URL{Host: "api.storyden.test"}, cache: store}

		cookie, err := a.startWebAuthnSession(context.Background(), original)
		r.NoError(err)

		ctx := context.WithValue(context.Background(), webauthnSessionContextKey{}, webauthnSessionRef{
			cacheKey: webauthnCachePrefix + cookie.Value,
			data:     original,
		})

		got, err := consumeWebAuthnSession(ctx, a)
		r.NoError(err)
		as.Equal(original.Challenge, got.Challenge)
		as.Equal(original.UserVerification, got.UserVerification)

		// A second attempt to read the same reference from the underlying
		// store must fail now that it's been consumed.
		_, err = store.Get(context.Background(), webauthnCachePrefix+cookie.Value)
		as.Error(err, "session data was not deleted after being consumed")
	})

	t.Run("missing_cookie_reference_is_rejected", func(t *testing.T) {
		r := require.New(t)

		_, err := consumeWebAuthnSession(context.Background(), a)
		r.Error(err)
	})
}
