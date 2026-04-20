package auth

import "go.uber.org/fx"

// Module provides auth domain dependencies.
var Module = fx.Module("auth",
	fx.Options(
		fx.Provide(
			NewRepository,
			NewService,
			NewController,
			NewRoute,
		),
		fx.Invoke(RegisterRoute),
	),
)
