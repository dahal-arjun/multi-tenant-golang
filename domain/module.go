package domain

import (
	"clean-architecture/domain/auth"
	"clean-architecture/domain/platform"
	"clean-architecture/domain/tenantroles"
	"clean-architecture/domain/user"
	"clean-architecture/domain/widget"

	"go.uber.org/fx"
)

var Module = fx.Module("domain",
	fx.Options(
		tenantroles.Module,
		auth.Module,
		platform.Module,
		user.Module,
		widget.Module,
	),
)
