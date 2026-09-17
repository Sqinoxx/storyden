package settings

import (
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
	"github.com/Southclaws/opt"

	"github.com/Southclaws/storyden/internal/infrastructure/endec/secretbox"
)

// encryptSecrets replaces plaintext secret fields (currently just the Google
// Drive service account key) with their encrypted form before a Settings
// value is persisted. The in-memory Settings value passed around the rest of
// the app always holds plaintext (see decryptSecrets); only what reaches the
// database is ciphertext, so a leaked database dump - the exact incident this
// closes - doesn't hand over a usable credential.
func (d *SettingsRepository) encryptSecrets(s *Settings) error {
	services, ok := s.Services.Get()
	if !ok {
		return nil
	}

	drive, ok := services.Drive.Get()
	if !ok {
		return nil
	}

	raw := drive.ServiceAccountJSON.Or("")
	if raw == "" {
		return nil
	}

	ciphertext, err := secretbox.Encrypt(d.config.JWTSecret, raw)
	if err != nil {
		return fault.Wrap(err,
			ftag.With(ftag.InvalidArgument),
			fmsg.WithDesc("cannot store Google Drive credentials",
				"A JWT_SECRET must be configured before a Google Drive service account key can be saved."),
		)
	}

	drive.ServiceAccountJSON = opt.New(ciphertext)
	services.Drive = opt.New(drive)
	s.Services = opt.New(services)

	return nil
}

// decryptSecrets reverses encryptSecrets after a Settings value is loaded
// from the database, so every other consumer of a Settings value (the Drive
// credentials resolver, the admin status endpoint, etc.) keeps working with
// plaintext exactly as before this field was encrypted.
func (d *SettingsRepository) decryptSecrets(s *Settings) {
	services, ok := s.Services.Get()
	if !ok {
		return
	}

	drive, ok := services.Drive.Get()
	if !ok {
		return
	}

	raw := drive.ServiceAccountJSON.Or("")
	if raw == "" {
		return
	}

	plaintext, err := secretbox.Decrypt(d.config.JWTSecret, raw)
	if err != nil {
		// A row written before this field was encrypted (or written while no
		// JWT_SECRET was configured) holds the raw key as-is. Use it
		// unchanged rather than breaking an existing Drive integration -
		// it's transparently encrypted the next time settings are saved.
		return
	}

	drive.ServiceAccountJSON = opt.New(plaintext)
	services.Drive = opt.New(drive)
	s.Services = opt.New(services)
}
