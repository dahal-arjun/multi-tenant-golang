package tenantroles

import "go.uber.org/fx"

// providePermCalculator registers the same *Service instance as PermCalculator for cross-domain fx wiring.
func providePermCalculator(s *Service) PermCalculator {
	return s
}

// Module provides tenant role dependencies.
var Module = fx.Module("tenantroles",
	fx.Options(
		fx.Provide(
			NewRepository,
			NewService,
			providePermCalculator,
			NewController,
			NewRoute,
		),
		fx.Invoke(RegisterRoute),
	),
)
