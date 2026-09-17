package loginguard

import (
	"go.uber.org/fx"
)

func Build() fx.Option {
	return fx.Provide(New)
}
