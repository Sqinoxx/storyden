package token

import (
	"encoding/json"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/opt"
	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/rs/xid"
)

var (
	ErrTokenExpired = fault.New("token expired")
	ErrTokenRevoked = fault.New("token revoked")
)

type Token struct{ xid.ID }

func FromString(b string) (Token, error) {
	id, err := xid.FromString(b)
	if err != nil {
		return Token{}, err
	}

	return Token{id}, nil
}

func Generate() Token {
	return Token{xid.New()}
}

func (t Token) Bytes() []byte {
	return t.ID.Bytes()
}

func (t Token) String() string {
	return t.ID.String()
}

// MarshalJSON encodes the token as a bare string (`"c5n4mpk1..."`).
func (t Token) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

// UnmarshalJSON decodes the token from a bare string.
func (t *Token) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	id, err := xid.FromString(s)
	if err != nil {
		return err
	}
	*t = Token{id}
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
