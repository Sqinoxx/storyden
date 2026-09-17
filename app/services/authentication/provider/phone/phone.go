package phone

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
	"github.com/Southclaws/opt"
	"github.com/rs/xid"
	"github.com/samber/lo"

	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/app/resources/account/account_querier"
	"github.com/Southclaws/storyden/app/resources/account/authentication"
	"github.com/Southclaws/storyden/app/resources/settings"
	"github.com/Southclaws/storyden/app/services/account/register"
	"github.com/Southclaws/storyden/app/services/authentication/provider"
	"github.com/Southclaws/storyden/internal/infrastructure/sms"
	"github.com/Southclaws/storyden/internal/otp"
)

var (
	errHandleMismatch      = fault.New("phone already linked to different account")
	errNoPhoneAuth         = fault.New("no phone auth method linked to account")
	errNotFound            = fault.New("account not found")
	errOneTimeCodeMismatch = fault.New("one time code mismatch")
)

const (
	// codeLifespan is how long an SMS one-time code stays valid.
	codeLifespan = 10 * time.Minute

	// maxCodeAttempts caps wrong guesses against a single issued code before
	// it is locked out and a new one must be requested via Register.
	maxCodeAttempts = 5
)

// hashCode is what's actually persisted, so a database read can't be turned
// back into a usable code.
func hashCode(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

// codeAttempts reads the failed-attempt counter out of the auth record's
// free-form metadata.
func codeAttempts(metadata interface{}) int {
	m, ok := metadata.(map[string]interface{})
	if !ok {
		return 0
	}

	n, ok := m["attempts"].(float64)
	if !ok {
		return 0
	}

	return int(n)
}

// invalidated returns an unguessable value to overwrite a consumed code with,
// so it can never be replayed.
func invalidated() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

var (
	requiredMode = authentication.ModePhone
	service      = authentication.ServicePhoneVerify
	tokenType    = authentication.TokenTypeNone
)

const template = `Your unique one-time login code is: %s`

type Provider struct {
	logger   *slog.Logger
	settings *settings.SettingsRepository
	auth     authentication.Repository
	account  *account_querier.Querier
	register *register.Registrar

	sms sms.Sender
}

func New(
	logger *slog.Logger,
	settings *settings.SettingsRepository,
	auth authentication.Repository,
	account *account_querier.Querier,
	register *register.Registrar,
	sms sms.Sender,
) *Provider {
	return &Provider{
		logger:   logger,
		settings: settings,
		auth:     auth,
		account:  account,
		register: register,
		sms:      sms,
	}
}

func (p *Provider) Service() authentication.Service { return service }
func (p *Provider) Token() authentication.TokenType { return tokenType }

func (p *Provider) Enabled(ctx context.Context) (bool, error) {
	settings, err := p.settings.Get(ctx)
	if err != nil {
		return false, fault.Wrap(err, fctx.With(ctx))
	}

	return settings.AuthenticationMode.Or(authentication.ModeHandle) == requiredMode, nil
}

func (p *Provider) Register(ctx context.Context, handle string, phone string, inviteCode opt.Optional[xid.ID]) (*account.Account, error) {
	if err := provider.CheckMode(ctx, p.logger, p.settings, requiredMode); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}
	//
	// STEP 1.
	//
	// Using the provided phone number, look up an authentication record which
	// points to an account already registered with the system. We need to do
	// this because there's no separation between registration and login via the
	// phone login system so if there's an account already, we start auth again.
	//

	authrecord, exists, err := p.auth.LookupByIdentifier(ctx, service, phone)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx), fmsg.With("failed to get account"))
	}

	var acc *account.Account
	if exists {
		acc = &authrecord.Account
		if err := acc.RejectSuspended(); err != nil {
			return nil, fault.Wrap(err, fctx.With(ctx))
		}
		if acc.Handle != handle {
			return nil, fault.Wrap(errHandleMismatch,
				fctx.With(ctx),
				ftag.With(ftag.PermissionDenied),
				fmsg.WithDesc("handle mismatch", "Phone number already registered to a different account."),
			)
		}

		//
		// STEP 1.5:
		//
		// If an account already exists, there's a chance the account also has a
		// phone authentication record associated with it. Currently, we only
		// support a single phone associated with an account so if there is one,
		// it needs to be deleted so it can be created again with a new code.
		//

		auths, err := p.auth.GetAuthMethods(ctx, acc.ID)
		if err != nil {
			return nil, fault.Wrap(err, fctx.With(ctx))
		}

		// If there's already a phone auth associated with the account, deleted it
		// and start fresh with the new request.
		// NOTE: This could result in a DoS for the account holder...
		if _, exists = lo.Find(auths, func(a *authentication.Authentication) bool {
			return a.Service == service
		}); exists {
			_, err = p.auth.Delete(ctx, acc.ID, phone, service)
			if err != nil {
				return nil, fault.Wrap(err, fctx.With(ctx))
			}
		}

	} else {
		//
		// If there isn't an account already with this phone number, we create
		// a new one using the @handle specified in the request.
		//

		acc, err = p.register.Register(ctx, opt.New(handle), inviteCode)
		if err != nil {
			if ftag.Get(err) == ftag.AlreadyExists {
				return nil, fault.Wrap(err,
					fctx.With(ctx),
					fmsg.With("failed to create account"),
					fmsg.WithDesc("already exists", "Handle already registered with a different authentication method."))
			}
			return nil, fault.Wrap(err, fctx.With(ctx), fmsg.With("failed to create account"))
		}
	}

	//
	// STEP 2:
	//
	// Generate a one-time-password which is a 6 digit number and send this to
	// the phone number specified in the request.
	//

	code, err := otp.Generate()
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx), fmsg.With("failed to generate code"))
	}

	_, err = p.auth.Create(ctx, acc.ID, service, authentication.TokenTypeNone, phone, hashCode(code), nil,
		authentication.WithExpiresAt(time.Now().Add(codeLifespan)))
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx), fmsg.With("failed to create account authentication instance"))
	}

	// TODO: For whitelabling, allow the instance brand name to be specified in
	// the message template. So the message says "Log in to Acme with xyz..."
	message := fmt.Sprintf(template, code)
	err = p.sms.Send(ctx, phone, message)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return acc, nil
}

func (b *Provider) Link(_ string) (string, error) {
	// Phone provider does not use external links.
	return "", nil
}

func (p *Provider) Login(ctx context.Context, handle string, onetimecode string) (*account.Account, error) {
	acc, exists, err := p.account.LookupByHandle(ctx, handle)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}
	if !exists {
		return nil, fault.Wrap(errNotFound,
			fctx.With(ctx),
			ftag.With(ftag.NotFound),
			fmsg.WithDesc("not found", "No account was found with the provided handle."))
	}

	if err := acc.RejectSuspended(); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	auths, err := p.auth.GetAuthMethods(ctx, acc.ID)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	phoneauth, exists := lo.Find(auths, func(a *authentication.Authentication) bool {
		return a.Service == service
	})
	if !exists {
		return nil, fault.Wrap(errNoPhoneAuth)
	}

	attempts := codeAttempts(phoneauth.Metadata)
	locked := attempts >= maxCodeAttempts
	match := subtle.ConstantTimeCompare([]byte(phoneauth.Token), []byte(hashCode(onetimecode))) == 1

	if locked || phoneauth.IsExpired() || !match {
		if !locked {
			_, _ = p.auth.Update(ctx, phoneauth.ID,
				authentication.WithMetadata(map[string]any{"attempts": attempts + 1}))
		}

		return nil, fault.Wrap(errOneTimeCodeMismatch,
			fctx.With(ctx),
			ftag.With(ftag.PermissionDenied),
			fmsg.WithDesc("mismatch", "The code did not match."),
		)
	}

	// Consume the code so it cannot be replayed; the next login request goes
	// through Register again, which always issues a fresh one.
	if _, err := p.auth.Update(ctx, phoneauth.ID, authentication.WithToken(invalidated())); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return &acc.Account, nil
}
