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
	api.POST("/auth/signup", r.controller.Signup)
	api.POST("/auth/login", r.controller.Login)
	api.POST("/auth/tenant-session", r.controller.TenantSession)
	api.POST("/auth/create-tenant", r.controller.CreateTenant)
	api.POST("/auth/verify-email", r.controller.VerifyEmail)
	api.POST("/auth/resend-verification", r.controller.ResendVerification)
	api.POST("/auth/accept-invite", r.controller.AcceptInvite)
	api.POST("/auth/refresh", r.controller.Refresh)
	api.POST("/auth/forgot-password", r.controller.ForgotPassword)
	api.POST("/auth/reset-password", r.controller.ResetPassword)

	protected := api.Group("")
	protected.Use(r.jwtMiddleware.Handle())
	protected.GET("/auth/me", r.controller.Me)
	protected.PATCH("/auth/me", r.controller.UpdateMe)
	protected.POST("/auth/logout", r.controller.Logout)
	protected.POST("/auth/change-password", r.controller.ChangePassword)
	protected.POST("/invites", r.controller.CreateTenantInvite)
}
