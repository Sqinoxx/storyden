package settings

import (
	"time"

	"github.com/Southclaws/opt"
	"github.com/Southclaws/storyden/app/resources/account/authentication"
	"github.com/Southclaws/storyden/app/resources/datagraph"
)

const (
	DefaultTitle       = "Storyden"
	DefaultDescription = "A forum for the modern age"
	DefaultContent     = `<body>
<p>Welcome to your new community!</p>
<p>You can edit this content by clicking Edit below.</p>
<p>This is a <em>rich text section</em> for telling visitors what your community is about.</p>
<p>Add a link to your <a href="https://discord.gg/XF6ZBGF9XF">Discord</a> or other sites.</p>
<p>Enjoy!</p>
</body>`
)
const DefaultColour = "hsl(157, 65%, 44%)"

// skip error check, we know it's correct, it's literally above ^^
var defaultContent, _ = datagraph.NewRichText(DefaultContent)

var DefaultSettings = Settings{
	Title:              opt.New(DefaultTitle),
	Description:        opt.New(DefaultDescription),
	Content:            opt.New(defaultContent),
	AccentColour:       opt.New(DefaultColour),
	AuthenticationMode: opt.New(authentication.ModeHandle),
	RegistrationMode:   opt.New(RegistrationModePublic),
	Services: opt.New(ServiceSettings{
		Moderation: opt.New(ModerationServiceSettings{
			ThreadBodyLengthMax: opt.New(60000),
			ReplyBodyLengthMax:  opt.New(10000),
			SignatureLengthMax:  opt.New(500),
		}),
		Content: opt.New(ContentServiceSettings{
			RepliesPerPage: opt.New(DefaultRepliesPerPage),
			ThreadsPerPage: opt.New(DefaultThreadsPerPage),
		}),
		Session: opt.New(SessionServiceSettings{
			IdleTimeout:       opt.New(DefaultSessionIdleTimeout),
			RememberMeDefault: opt.New(DefaultRememberMeDuration),
			RememberMeMax:     opt.New(DefaultRememberMeMax),
		}),
	}),
}

const (
	DefaultRepliesPerPage = 15
	DefaultThreadsPerPage = 50

	DefaultSessionIdleTimeout = time.Hour
	DefaultRememberMeDuration = time.Hour * 24 * 30
	DefaultRememberMeMax      = time.Hour * 24 * 365
)

// RepliesPerPage resolves the configured thread reply page size.
func (s *Settings) RepliesPerPage() int {
	return clampPageSize(contentSettings(s).RepliesPerPage, DefaultRepliesPerPage, 200)
}

// ThreadsPerPage resolves the configured thread list page size.
func (s *Settings) ThreadsPerPage() int {
	return clampPageSize(contentSettings(s).ThreadsPerPage, DefaultThreadsPerPage, 100)
}

// SessionIdleTimeout is the sliding window applied to sessions issued without
// "remember me".
func (s *Settings) SessionIdleTimeout() time.Duration {
	return sessionSettings(s).IdleTimeout.Or(DefaultSessionIdleTimeout)
}

// RememberMeDuration resolves a member's requested session length against the
// instance default and maximum.
func (s *Settings) RememberMeDuration(requested opt.Optional[time.Duration]) time.Duration {
	session := sessionSettings(s)
	max := session.RememberMeMax.Or(DefaultRememberMeMax)

	duration := requested.Or(session.RememberMeDefault.Or(DefaultRememberMeDuration))

	if duration > max {
		return max
	}
	if duration < time.Hour {
		return time.Hour
	}

	return duration
}

func contentSettings(s *Settings) ContentServiceSettings {
	return s.Services.OrZero().Content.OrZero()
}

func sessionSettings(s *Settings) SessionServiceSettings {
	return s.Services.OrZero().Session.OrZero()
}

func clampPageSize(configured opt.Optional[int], fallback int, max int) int {
	size := configured.Or(fallback)

	if size < 1 {
		return fallback
	}
	if size > max {
		return max
	}

	return size
}
