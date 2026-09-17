package settings_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Southclaws/opt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/datagraph"
	"github.com/Southclaws/storyden/app/resources/settings"
	"github.com/Southclaws/storyden/internal/config"
	"github.com/Southclaws/storyden/internal/ent"
	"github.com/Southclaws/storyden/internal/integration"
)

func TestSettingsRepository(t *testing.T) {
	t.Parallel()

	integration.Test(t, nil, fx.Invoke(func(lc fx.Lifecycle, sr *settings.SettingsRepository) {
		lc.Append(fx.StartHook(func(ctx context.Context) {
			t.Run("partial_update", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				content, err := datagraph.NewRichText("<body><p>Hello, Makeroom!</p></body>")
				r.NoError(err)

				set, err := sr.Set(ctx, settings.Settings{
					Title:   opt.New("Makeroom"),
					Content: opt.New(content),
				})
				r.NoError(err)
				r.NotNil(set)

				got, err := sr.Get(ctx)
				r.NoError(err)
				r.NotNil(got)

				a.Equal("Makeroom", got.Title.OrZero())
				a.Equal(content.HTML(), got.Content.OrZero().HTML())
				a.Equal(settings.DefaultDescription, got.Description.OrZero())
			})
		}))
	}))
}

// TestSettingsRepository_DriveKeyEncryption covers B21: the Google service
// account key was stored in the settings table as plaintext, which is
// exactly what turned a database dump leak into a leaked credential in the
// incident this whole remediation pass is responding to.
func TestSettingsRepository_DriveKeyEncryption(t *testing.T) {
	t.Parallel()

	rawKey := `{"type":"service_account","private_key":"very secret key material"}`

	integration.Test(t, &config.Config{
		JWTSecret: []byte("07d422e512b23a056ccc953994d1593f"),
	}, fx.Invoke(func(lc fx.Lifecycle, sr *settings.SettingsRepository, db *ent.Client) {
		lc.Append(fx.StartHook(func(ctx context.Context) {
			t.Run("stored value is encrypted, round trip returns plaintext", func(t *testing.T) {
				r := require.New(t)
				a := assert.New(t)

				_, err := sr.Set(ctx, settings.Settings{
					Services: opt.New(settings.ServiceSettings{
						Drive: opt.New(settings.DriveServiceSettings{
							ServiceAccountJSON: opt.New(rawKey),
						}),
					}),
				})
				r.NoError(err)

				got, err := sr.Get(ctx)
				r.NoError(err)
				a.Equal(rawKey, got.Services.OrZero().Drive.OrZero().ServiceAccountJSON.OrZero(),
					"the in-memory value returned to callers must be plaintext")

				row, err := db.Setting.Get(ctx, settings.StorydenPrimarySettingsKey)
				r.NoError(err)
				a.NotContains(row.Value, "very secret key material", "the raw database row must not contain the plaintext key")
				a.NotContains(row.Value, rawKey)
			})
		}))
	}))
}

// TestSettingsRepository_DriveKeyBackwardsCompatibleWithPlaintextRow covers a
// row written before this field was encrypted (or written by an older
// version): it must still be usable rather than treated as corrupt.
func TestSettingsRepository_DriveKeyBackwardsCompatibleWithPlaintextRow(t *testing.T) {
	t.Parallel()

	rawKey := `{"type":"service_account","private_key":"legacy plaintext key"}`

	integration.Test(t, &config.Config{
		JWTSecret: []byte("07d422e512b23a056ccc953994d1593f"),
	}, fx.Invoke(func(lc fx.Lifecycle, sr *settings.SettingsRepository, db *ent.Client) {
		lc.Append(fx.StartHook(func(ctx context.Context) {
			r := require.New(t)
			a := assert.New(t)

			// Force a legacy row into existence directly, bypassing the
			// repository's own (now encrypting) write path, to simulate
			// data written before this change shipped.
			current, err := sr.Get(ctx)
			r.NoError(err)

			current.Services = opt.New(settings.ServiceSettings{
				Drive: opt.New(settings.DriveServiceSettings{
					ServiceAccountJSON: opt.New(rawKey),
				}),
			})

			b, err := json.Marshal(current)
			r.NoError(err)

			_, err = db.Setting.UpdateOneID(settings.StorydenPrimarySettingsKey).SetValue(string(b)).Save(ctx)
			r.NoError(err)

			got, err := sr.Get(ctx)
			r.NoError(err)
			a.Equal(rawKey, got.Services.OrZero().Drive.OrZero().ServiceAccountJSON.OrZero(),
				"a legacy plaintext row must still be read back correctly")
		}))
	}))
}
