package platform

import (
	"clean-architecture/pkg/framework"
	"clean-architecture/pkg/infrastructure"
	"clean-architecture/pkg/middlewares"
)

// Route wires platform HTTP routes.
type Route struct {
	logger         framework.Logger
	handler        infrastructure.Router
	controller     *Controller
	platformJWT    middlewares.PlatformJWTAuthMiddleware
}

// NewRoute constructs platform routes.
func NewRoute(
	logger framework.Logger,
	handler infrastructure.Router,
	controller *Controller,
	platformJWT middlewares.PlatformJWTAuthMiddleware,
) *Route {
	return &Route{
		logger:      logger,
		handler:     handler,
		controller:  controller,
		platformJWT: platformJWT,
	}
}

// RegisterRoute registers /api/platform/* (platform JWT only).
func RegisterRoute(r *Route) {
	r.logger.Info("Setting up platform routes")
	api := r.handler.Group("/api/platform")
	api.Use(r.platformJWT.Handle())
	api.GET("/health", r.controller.Health)
	api.GET("/tenants/summary", r.controller.TenantSummary)
}
