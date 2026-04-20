package platform

import "go.uber.org/fx"

// Module provides platform (system operator) HTTP surface.
var Module = fx.Module("platform",
	fx.Options(
		fx.Provide(
			NewService,
			NewController,
			NewRoute,
		),
		fx.Invoke(RegisterRoute),
	),
)
