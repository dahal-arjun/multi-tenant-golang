package tenantroles

import "go.uber.org/fx"

// Module provides tenant role dependencies.
var Module = fx.Module("tenantroles",
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
