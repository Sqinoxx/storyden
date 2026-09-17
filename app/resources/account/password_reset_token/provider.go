package password_reset_token

import (
	"go.uber.org/fx"
)

func Build() fx.Option {
	return fx.Provide(New)
}
