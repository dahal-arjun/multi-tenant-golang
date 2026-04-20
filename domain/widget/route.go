package widget

import (
	"clean-architecture/pkg/framework"
	"clean-architecture/pkg/infrastructure"
	"clean-architecture/pkg/middlewares"
)

// Route registers widget HTTP routes.
type Route struct {
	logger        framework.Logger
	handler       infrastructure.Router
	controller    *Controller
	jwtMiddleware middlewares.JWTAuthMiddleware
}

// NewRoute constructs widget routes.
func NewRoute(
	logger framework.Logger,
	handler infrastructure.Router,
	controller *Controller,
	jwtMiddleware middlewares.JWTAuthMiddleware,
) *Route {
	return &Route{
		logger:        logger,
		handler:       handler,
		controller:    controller,
		jwtMiddleware: jwtMiddleware,
	}
}

// RegisterRoute wires /api/widgets.
func RegisterRoute(r *Route) {
	r.logger.Info("Setting up widget routes")
	api := r.handler.Group("/api")
	protected := api.Group("")
	protected.Use(r.jwtMiddleware.Handle())
	protected.GET("/widgets", r.controller.List)
}
