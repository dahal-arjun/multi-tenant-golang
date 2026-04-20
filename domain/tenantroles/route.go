package tenantroles

import (
	"clean-architecture/pkg/framework"
	"clean-architecture/pkg/infrastructure"
	"clean-architecture/pkg/middlewares"
)

// Route wires tenant role HTTP routes.
type Route struct {
	logger        framework.Logger
	handler       infrastructure.Router
	controller    *Controller
	jwtMiddleware middlewares.JWTAuthMiddleware
}

// NewRoute constructs routes.
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

// RegisterRoute registers /api/tenant-roles and related paths.
func RegisterRoute(r *Route) {
	r.logger.Info("Setting up tenant role routes")
	api := r.handler.Group("/api")
	protected := api.Group("")
	protected.Use(r.jwtMiddleware.Handle())
	protected.GET("/tenant-roles", r.controller.List)
	protected.POST("/tenant-roles", r.controller.Create)
	protected.PATCH("/tenant-roles/:id", r.controller.Update)
	protected.DELETE("/tenant-roles/:id", r.controller.Delete)
	protected.PUT("/tenant-roles/:id/permissions", r.controller.SetPermissions)
	protected.PATCH("/tenant-members/role", r.controller.AssignMemberRole)
}
