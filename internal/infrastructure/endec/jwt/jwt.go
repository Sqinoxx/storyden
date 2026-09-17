package jwt

import (
	"crypto/rand"
	"io"
	"time"

	"github.com/Southclaws/fault"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/internal/config"
	"github.com/Southclaws/storyden/internal/infrastructure/endec"
)

type jwtEncrypterDecrypter struct {
	key []byte
}

func Build() fx.Option {
	return fx.Provide(
		fx.Annotate(New,
			fx.As(new(endec.EncrypterDecrypter)),
			fx.As(new(endec.Encrypter)),
			fx.As(new(endec.Decrypter)),
		),
	)
}

// errNoJWTSecret is returned by both Encrypt and Decrypt when JWT_SECRET is
// unset. JWT_SECRET is only required when email or OAuth features are
// enabled (see config.yaml), so an empty one is a legitimate configuration
// for a handle-only, no-OAuth deployment - New must not fail outright. It
// previously returned a nil interface in that case, which any caller that
// did try to use it (e.g. OAuth state signing, password reset) would panic
// on with a bare nil pointer dereference instead of a clear error. Decrypt
// needs the same guard for a second reason: without it, an empty key is
// still a valid HMAC key, so Parse would happily verify - and anyone could
// forge - a token signed with that well-known empty key.
var errNoJWTSecret = fault.New("no JWT secret configured")

func New(cfg config.Config) (endec.EncrypterDecrypter, error) {
	return &jwtEncrypterDecrypter{key: cfg.JWTSecret}, nil
}

func (e *jwtEncrypterDecrypter) Encrypt(data endec.Claims, lifespan time.Duration) (string, error) {
	if len(e.key) == 0 {
		return "", errNoJWTSecret
	}

	var nonce [24]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return "", fault.Wrap(err)
	}

	expires := time.Now().UTC().Add(lifespan)

	claims := jwt.MapClaims{
		"jti": nonce,
		"exp": jwt.NewNumericDate(expires),
	}

	for k, v := range data {
		claims[k] = v
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	s, err := t.SignedString(e.key)
	if err != nil {
		return "", fault.Wrap(err)
	}

	return s, nil
}

func (e *jwtEncrypterDecrypter) Decrypt(message string) (endec.Claims, error) {
	if len(e.key) == 0 {
		return nil, errNoJWTSecret
	}

	t, err := jwt.Parse(message, e.keyfunc)
	if err != nil {
		return nil, fault.Wrap(err)
	}

	if !t.Valid {
		return nil, fault.New("token flagged as invalid but no error was reported")
	}

	claims, ok := t.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fault.New("invalid token")
	}

	return endec.Claims(claims), nil
}

func (e *jwtEncrypterDecrypter) keyfunc(token *jwt.Token) (any, error) {
	if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
		return nil, fault.Newf("invalid jwt algorithm %e", token.Header["alg"])
	}

	return e.key, nil
}
