package settings

import (
	"testing"

	"github.com/Southclaws/opt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Southclaws/storyden/internal/config"
)

// TestEncryptSecretsRequiresJWTSecret covers the write side of B21: a
// deployment with no JWT_SECRET configured must not be able to persist a
// Drive key that would only ever be stored in plaintext.
func TestEncryptSecretsRequiresJWTSecret(t *testing.T) {
	t.Parallel()

	d := &SettingsRepository{config: config.Config{}}

	s := &Settings{
		Services: opt.New(ServiceSettings{
			Drive: opt.New(DriveServiceSettings{
				ServiceAccountJSON: opt.New(`{"type":"service_account"}`),
			}),
		}),
	}

	err := d.encryptSecrets(s)
	require.Error(t, err)
}

// TestEncryptSecretsNoopWithoutDriveKey covers the common case: most Settings
// values have no Drive key at all, and must not require a JWT_SECRET just to
// save unrelated settings.
func TestEncryptSecretsNoopWithoutDriveKey(t *testing.T) {
	t.Parallel()

	d := &SettingsRepository{config: config.Config{}}

	s := &Settings{Title: opt.New("My Community")}

	err := d.encryptSecrets(s)
	require.NoError(t, err)
	assert.Equal(t, "My Community", s.Title.OrZero())
}

// TestEncryptDecryptSecretsRoundTrip covers the pure transform in isolation,
// without going through the database.
func TestEncryptDecryptSecretsRoundTrip(t *testing.T) {
	t.Parallel()

	d := &SettingsRepository{config: config.Config{JWTSecret: []byte("07d422e512b23a056ccc953994d1593f")}}

	raw := `{"type":"service_account","private_key":"secret"}`

	s := &Settings{
		Services: opt.New(ServiceSettings{
			Drive: opt.New(DriveServiceSettings{
				ServiceAccountJSON: opt.New(raw),
			}),
		}),
	}

	require.NoError(t, d.encryptSecrets(s))

	encrypted := s.Services.OrZero().Drive.OrZero().ServiceAccountJSON.OrZero()
	assert.NotEqual(t, raw, encrypted)
	assert.NotContains(t, encrypted, "secret")

	d.decryptSecrets(s)

	assert.Equal(t, raw, s.Services.OrZero().Drive.OrZero().ServiceAccountJSON.OrZero())
}
