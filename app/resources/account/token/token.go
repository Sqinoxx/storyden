package token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/opt"
	"github.com/Southclaws/storyden/app/resources/account"
)

var (
	ErrTokenExpired = fault.New("token expired")
	ErrTokenRevoked = fault.New("token revoked")
	ErrInvalidToken = fault.New("invalid token")
)

// secretLength is the size, in bytes, of a session token's random secret.
// 32 bytes (256 bits) from crypto/rand is unguessable well beyond any
// practical brute-force budget.
const secretLength = 32

// Token is an opaque, unguessable session credential. Only its Hash is ever
// persisted; the raw secret exists solely in the value handed to the client
// (cookie or bearer header), so reading the database cannot produce a valid
// token.
type Token struct{ secret []byte }

// Generate creates a new random session token.
func Generate() Token {
	b := make([]byte, secretLength)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}

	return Token{secret: b}
}

// FromString parses a token from its client-facing (base64url) form.
func FromString(s string) (Token, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return Token{}, fault.Wrap(err)
	}

	if len(b) != secretLength {
		return Token{}, ErrInvalidToken
	}

	return Token{secret: b}, nil
}

// Hash is the value actually stored in and looked up from the database, so
// that the raw token never needs to be persisted anywhere.
func (t Token) Hash() string {
	sum := sha256.Sum256(t.secret)
	return hex.EncodeToString(sum[:])
}

func (t Token) String() string {
	return base64.RawURLEncoding.EncodeToString(t.secret)
}

// MarshalJSON encodes the token as a bare string.
func (t Token) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

// UnmarshalJSON decodes the token from a bare string.
func (t *Token) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	tok, err := FromString(s)
	if err != nil {
		return err
	}

	*t = tok
	return nil
}

type Session struct {
	Token     Token                   `json:"t"`
	AccountID account.AccountID       `json:"a"`
	ExpiresAt time.Time               `json:"e"`
	RevokedAt opt.Optional[time.Time] `json:"r"`

	// RefreshedAt is when the sliding window was last extended, used to keep
	// the extension to at most one write per RefreshInterval.
	RefreshedAt opt.Optional[time.Time] `json:"rf"`

	// Lifetime is the sliding window size chosen when this session was issued.
	// Zero marks a session issued before lifetimes were configurable; those
	// keep their original expiry rather than being cut short.
	Lifetime time.Duration `json:"l"`

	// Persistent records whether the browser was given a cookie that survives
	// being closed, so refreshes re-issue the same kind.
	Persistent bool `json:"p"`
}

// IsLegacy reports a session issued before session lifetimes were configurable.
func (s Session) IsLegacy() bool {
	return s.Lifetime <= 0
}

// NeedsRefresh reports whether the sliding window is stale enough to be worth a
// write. Legacy sessions are left alone so an upgrade cannot shorten them.
func (s Session) NeedsRefresh(now time.Time, interval time.Duration) bool {
	if s.IsLegacy() {
		return false
	}

	refreshedAt, ok := s.RefreshedAt.Get()
	if !ok {
		return true
	}

	return now.Sub(refreshedAt) > interval
}

type Validated Session

func (s Session) Validate() (*Validated, error) {
	if s.RevokedAt.Ok() {
		return nil, ErrTokenRevoked
	}

	if s.ExpiresAt.Before(time.Now()) {
		return nil, ErrTokenExpired
	}

	return (*Validated)(&s), nil
}

func (t Session) Serialise() ([]byte, error) {
	jsonData, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}

	return jsonData, nil
}

func Deserialise(data []byte) (*Session, error) {
	t := Session{}
	err := json.Unmarshal(data, &t)
	if err != nil {
		return nil, err
	}

	return &t, nil
}
