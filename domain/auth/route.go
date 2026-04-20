package auth

import (
	"clean-architecture/pkg/framework"
	"clean-architecture/pkg/infrastructure"
	"clean-architecture/pkg/middlewares"
)

// Route registers auth HTTP routes.
type Route struct {
	logger         framework.Logger
	handler        infrastructure.Router
	controller     *Controller
	jwtMiddleware  middlewares.JWTAuthMiddleware
}

// NewRoute constructs auth routes.
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

// RegisterRoute wires /api/auth/*.
func RegisterRoute(r *Route) {
	r.logger.Info("Setting up auth routes")

	api := r.handler.Group("/api")

	api.POST("/auth/register", r.controller.Register)
	api.POST("/auth/login", r.controller.Login)
	api.POST("/auth/tenant-session", r.controller.TenantSession)
	api.POST("/auth/refresh", r.controller.Refresh)
	api.POST("/auth/forgot-password", r.controller.ForgotPassword)
	api.POST("/auth/reset-password", r.controller.ResetPassword)

	protected := api.Group("/auth")
	protected.Use(r.jwtMiddleware.Handle())
	protected.GET("/me", r.controller.Me)
	protected.PATCH("/me", r.controller.UpdateMe)
	protected.POST("/logout", r.controller.Logout)
	protected.POST("/change-password", r.controller.ChangePassword)
}
