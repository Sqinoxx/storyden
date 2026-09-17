package discord

import (
	"context"
	"fmt"
	"net/mail"
	"strings"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
	"github.com/bwmarrin/discordgo"
	"golang.org/x/oauth2"

	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/app/resources/account/authentication"
	"github.com/Southclaws/storyden/app/services/account/register"
	"github.com/Southclaws/storyden/app/services/authentication/provider/oauth"

	"github.com/Southclaws/storyden/internal/config"
	"github.com/Southclaws/storyden/internal/infrastructure/endec"
)

var (
	service   = authentication.ServiceOAuthDiscord
	tokenType = authentication.TokenTypeOAuth
	endpoint  = oauth2.Endpoint{
		AuthURL:   "https://discord.com/oauth2/authorize",
		TokenURL:  "https://discord.com/api/oauth2/token",
		AuthStyle: oauth2.AuthStyleInParams,
	}
)

type Provider struct {
	config   oauth.Configuration
	register *register.Registrar
	ed       endec.EncrypterDecrypter
}

func New(
	cfg config.Config,
	register *register.Registrar,
	ed endec.EncrypterDecrypter,
) (*Provider, error) {
	if cfg.DiscordEnabled && ed == nil {
		return nil, fault.New("JWT provider must be enabled by setting JWT_SECRET for Discord OAuth provider")
	}

	return &Provider{
		config: oauth.Configuration{
			Enabled:      cfg.DiscordEnabled,
			ClientID:     cfg.DiscordClientID,
			ClientSecret: cfg.DiscordClientSecret,
		},
		register: register,
		ed:       ed,
	}, nil
}

func (p *Provider) Service() authentication.Service { return service }
func (p *Provider) Token() authentication.TokenType { return tokenType }

func (p *Provider) Enabled(ctx context.Context) (bool, error) {
	return p.config.Enabled, nil
}

func (p *Provider) oauthConfig(redirect string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     p.config.ClientID,
		ClientSecret: p.config.ClientSecret,
		Endpoint:     endpoint,
		RedirectURL:  redirect,
		Scopes: []string{
			"identify",
			"email",
		},
	}
}

func (p *Provider) Link(redirectPath string, nonce string) (string, error) {
	state, err := oauth.NewState(p.ed, redirectPath, nonce)
	if err != nil {
		return "", fault.Wrap(err)
	}

	oac := p.oauthConfig(redirectPath)

	return oac.AuthCodeURL(state, oauth2.AccessTypeOffline), nil
}

func (p *Provider) Login(ctx context.Context, state, nonce, code string) (*account.Account, error) {
	redirect, err := oauth.VerifyState(p.ed, state, nonce)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	oac := p.oauthConfig(redirect)

	token, err := oac.Exchange(ctx, code, oauth2.AccessTypeOffline)
	if err != nil {
		return nil, fault.Wrap(err,
			fctx.With(ctx),
			ftag.With(ftag.InvalidArgument),
			fmsg.WithDesc("failed to exchange code for token", "This login token may have expired, please try again from the start."),
		)
	}

	client, err := discordgo.New(fmt.Sprintf("Bearer %s", token.AccessToken))
	if err != nil {
		return nil, fault.Wrap(err,
			fctx.With(ctx),
			fmsg.WithDesc("failed to create Discord client", "Unable to connect to Discord. Please try again."))
	}

	u, err := client.User("@me", discordgo.WithContext(ctx))
	if err != nil {
		return nil, fault.Wrap(err,
			fctx.With(ctx),
			fmsg.WithDesc("failed to fetch Discord user info", "Unable to retrieve your Discord profile. Please try again."))
	}

	handle := strings.ToLower(u.Username)
	name := u.GlobalName

	email, err := mail.ParseAddress(u.Email)
	if err != nil {
		return nil, fault.Wrap(err,
			fctx.With(ctx),
			fmsg.WithDesc("failed to parse Discord email address", "The email address from Discord is invalid. Please check your Discord account settings."))
	}

	authName := fmt.Sprintf("Discord (@%s)", handle)

	return p.register.GetOrCreateViaEmail(ctx,
		service,
		authName,
		u.ID,
		token.AccessToken,
		handle,
		name,
		*email,
	)
}
