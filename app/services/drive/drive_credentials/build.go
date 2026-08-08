package drive_credentials

import (
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/internal/infrastructure/gdrive"
)

// Build provides both the concrete *Resolver, for the admin credentials
// endpoints, and the gdrive.Client interface it implements, for the rest of
// the Drive feature (folder management and browsing).
func Build() fx.Option {
	return fx.Provide(
		New,
		func(r *Resolver) gdrive.Client { return r },
	)
}
