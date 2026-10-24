package listing

import (
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/services/listing/listing_expiry"
	"github.com/Southclaws/storyden/app/services/listing/listing_manager"
)

func Build() fx.Option {
	return fx.Options(
		fx.Provide(listing_manager.New),
		listing_expiry.Build(),
	)
}
