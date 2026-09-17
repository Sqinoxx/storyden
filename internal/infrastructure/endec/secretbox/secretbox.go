// Package secretbox encrypts small secrets (service account keys, API keys)
// at rest, deriving its key from JWT_SECRET rather than requiring a second
// dedicated secret to configure and rotate.
package secretbox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
)

// keyInfo domain-separates this derived key from JWT_SECRET's other use
// signing session/state JWTs, so the two purposes never share key material
// even though they're derived from the same underlying secret.
const keyInfo = "storyden:settings:secretbox:v1"

var (
	ErrNoKey           = errors.New("no encryption key configured")
	ErrInvalidCipher   = errors.New("invalid or corrupt ciphertext")
	ErrCiphertextEmpty = errors.New("empty ciphertext")
)

func deriveKey(jwtSecret []byte) []byte {
	mac := hmac.New(sha256.New, jwtSecret)
	mac.Write([]byte(keyInfo))
	return mac.Sum(nil)
}

// Encrypt returns a base64-encoded, self-contained (nonce-prefixed)
// AES-256-GCM ciphertext for plaintext, suitable for storing as a string.
func Encrypt(jwtSecret []byte, plaintext string) (string, error) {
	if len(jwtSecret) == 0 {
		return "", ErrNoKey
	}

	block, err := aes.NewCipher(deriveKey(jwtSecret))
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	return base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt reverses Encrypt. It fails closed: any error decoding, sizing or
// authenticating the ciphertext is reported rather than returning partial or
// zero-value data.
func Decrypt(jwtSecret []byte, encoded string) (string, error) {
	if len(jwtSecret) == 0 {
		return "", ErrNoKey
	}

	if encoded == "" {
		return "", ErrCiphertextEmpty
	}

	sealed, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", ErrInvalidCipher
	}

	block, err := aes.NewCipher(deriveKey(jwtSecret))
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(sealed) < nonceSize {
		return "", ErrInvalidCipher
	}

	nonce, ciphertext := sealed[:nonceSize], sealed[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", ErrInvalidCipher
	}

	return string(plaintext), nil
}
