package domain

import (
	"clean-architecture/domain/auth"
	"clean-architecture/domain/user"
	"clean-architecture/domain/widget"

	"go.uber.org/fx"
)

var Module = fx.Module("domain",
	fx.Options(
		auth.Module,
		user.Module,
		widget.Module,
	),
)
