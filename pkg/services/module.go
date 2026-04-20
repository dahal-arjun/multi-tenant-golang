package services

import "go.uber.org/fx"

// Module is reserved for optional app-wide services (no cloud dependencies in the base template).
var Module = fx.Module("services")
