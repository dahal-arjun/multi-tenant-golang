package user

import (
	"clean-architecture/pkg/framework"
	"clean-architecture/pkg/infrastructure"
	"clean-architecture/pkg/middlewares"
)

// Route wires user HTTP routes.
type Route struct {
	logger        framework.Logger
	handler       infrastructure.Router
	controller    *Controller
	jwtMiddleware middlewares.JWTAuthMiddleware
}

// NewRoute initializes a new Route instance
func NewRoute(
	logger framework.Logger,
	handler infrastructure.Router,
	controller *Controller,
	jwtMiddleware middlewares.JWTAuthMiddleware,
) *Route {
	return &Route{
		handler:       handler,
		logger:        logger,
		controller:    controller,
		jwtMiddleware: jwtMiddleware,
	}
}

// RegisterRoute sets up user routes (JWT required).
func RegisterRoute(r *Route) {
	r.logger.Info("Setting up user routes")

	api := r.handler.Group("/api")
	protected := api.Group("")
	protected.Use(r.jwtMiddleware.Handle())
	protected.GET("/user/:id", r.controller.GetUserByID)
}
