package authentication

import (
	"context"

	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/app/resources/account/authentication"
)

// Provider describes a type that can be used to authenticate people.
type Provider interface {
	// Enabled tells the auth method manager whether this method is enabled.
	Enabled(ctx context.Context) (bool, error)

	// Service returns the unique identifier for the service. This is used for
	// the repository layer to record which auth methods a member has used. It
	// may also be used by clients to show a user-friendly label for the method.
	Service() authentication.Service

	// Token returns the type of token/secret that this provider uses.
	Token() authentication.TokenType
}

type OAuthProvider interface {
	// Link will, for providers that support it, provide a URL to a third-party
	// authenticator. OAuth providers use this to start the authentication flow.
	// nonce is an anti-CSRF value the caller has also set in a short-lived
	// cookie, bound into the returned state so Login can reject a state/code
	// pair completed in a different browser (login CSRF).
	Link(redirect string, nonce string) (string, error)

	// Login is a function that will validate and authenticate a user given that
	// the provider is happy with the input. The input format differs depending
	// on the provider. For example, an OAuth provider will use `state` and
	// `secret` as the `state` and `code` components of the OAuth2 specification
	// and a simple password-based provider may simply want the account handle
	// in the `state` and their password in the `secret`. nonce is the value
	// from the caller's storyden-oauth-state cookie, checked against the one
	// embedded in state by Link.
	Login(ctx context.Context, state, nonce, secret string) (*account.Account, error)
}
