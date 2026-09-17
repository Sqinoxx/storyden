package jwt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Southclaws/storyden/internal/config"
	"github.com/Southclaws/storyden/internal/infrastructure/endec"
)

func TestEncryptDecrypt(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	claims := endec.Claims{
		"sub": "test-subject",
		"exp": float64(time.Now().Add(1 * time.Hour).Unix()),
	}

	ed, err := New(config.Config{
		JWTSecret: []byte("07d422e512b23a056ccc953994d1593f"),
	})
	r.NoError(err)

	t.Run("encrypt and decrypt payload", func(t *testing.T) {
		token, err := ed.Encrypt(claims, time.Hour)
		a.NoError(err)

		gotClaims, err := ed.Decrypt(token)
		a.NoError(err)
		a.Equal(claims["sub"], gotClaims["sub"])
		a.Equal(claims["exp"], gotClaims["exp"])
	})

	// B21: an empty secret is a legitimate configuration (no email/OAuth
	// features enabled), so New must still succeed - but every operation on
	// the result must fail clearly rather than panicking on a nil interface
	// or, worse, silently signing/verifying with a well-known empty key.
	t.Run("empty_secret_fails_clearly_instead_of_a_nil_interface", func(t *testing.T) {
		ed, err := New(config.Config{
			JWTSecret: []byte{},
		})
		r.NoError(err)
		r.NotNil(ed)

		_, err = ed.Encrypt(claims, time.Hour)
		a.ErrorIs(err, errNoJWTSecret)

		_, err = ed.Decrypt("anything")
		a.ErrorIs(err, errNoJWTSecret)
	})
}
