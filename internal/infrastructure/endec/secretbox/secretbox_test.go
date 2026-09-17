package secretbox

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testKey = []byte("07d422e512b23a056ccc953994d1593f")

func TestEncryptDecryptRoundTrip(t *testing.T) {
	t.Parallel()

	plaintext := `{"type":"service_account","private_key":"very secret"}`

	ciphertext, err := Encrypt(testKey, plaintext)
	require.NoError(t, err)
	assert.NotContains(t, ciphertext, "very secret", "the encoded form must not leak the plaintext")
	assert.NotContains(t, ciphertext, "private_key")

	got, err := Decrypt(testKey, ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, got)
}

func TestEncryptIsNotDeterministic(t *testing.T) {
	t.Parallel()

	a, err := Encrypt(testKey, "same input")
	require.NoError(t, err)
	b, err := Encrypt(testKey, "same input")
	require.NoError(t, err)

	assert.NotEqual(t, a, b, "a fresh random nonce must be used every time")
}

func TestDecryptRejectsWrongKey(t *testing.T) {
	t.Parallel()

	ciphertext, err := Encrypt(testKey, "secret")
	require.NoError(t, err)

	_, err = Decrypt([]byte("a different jwt secret entirely"), ciphertext)
	assert.ErrorIs(t, err, ErrInvalidCipher)
}

func TestDecryptRejectsTamperedCiphertext(t *testing.T) {
	t.Parallel()

	ciphertext, err := Encrypt(testKey, "secret")
	require.NoError(t, err)

	tampered := []byte(ciphertext)
	tampered[len(tampered)-1] ^= 0xFF

	_, err = Decrypt(testKey, string(tampered))
	assert.Error(t, err)
}

func TestDecryptRejectsPlaintextJSON(t *testing.T) {
	t.Parallel()

	// A pre-encryption row, or any other non-ciphertext value, must not be
	// silently accepted as valid - Decrypt fails closed.
	_, err := Decrypt(testKey, `{"type":"service_account"}`)
	assert.ErrorIs(t, err, ErrInvalidCipher)
}

func TestEncryptRequiresAKey(t *testing.T) {
	t.Parallel()

	_, err := Encrypt(nil, "secret")
	assert.ErrorIs(t, err, ErrNoKey)

	_, err = Decrypt(nil, "anything")
	assert.ErrorIs(t, err, ErrNoKey)
}
