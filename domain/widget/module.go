package widget

import "go.uber.org/fx"

// Module provides widget domain dependencies.
var Module = fx.Module("widget",
	fx.Options(
		fx.Provide(
			NewService,
			NewController,
			NewRoute,
		),
		fx.Invoke(RegisterRoute),
	),
)
