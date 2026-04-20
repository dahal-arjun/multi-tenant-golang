package auth

import (
	"clean-architecture/pkg/errorz"
	"clean-architecture/pkg/framework"
	"clean-architecture/pkg/responses"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Controller exposes HTTP handlers for authentication.
type Controller struct {
	service *Service
	logger  framework.Logger
}

// NewController constructs the auth controller.
func NewController(service *Service, logger framework.Logger) *Controller {
	return &Controller{service: service, logger: logger}
}

// Register godoc
// @Summary Register user and tenant
// @Tags auth
// @Accept json
// @Produce json
// @Param body body RegisterRequest true "Registration"
// @Success 201 {object} TokenResponse
// @Router /api/auth/register [post]
func (a *Controller) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.HandleValidationError(a.logger, c, err)
		return
	}
	out, err := a.service.Register(req)
	if err != nil {
		responses.HandleError(a.logger, c, err)
		return
	}
	responses.JSON(c, 201, out)
}

// Login godoc
// @Summary Login (discover tenants)
// @Description Returns tenant list and pick_tenant_token. Call POST /auth/tenant-session with Bearer pick_tenant_token and JSON tenant_id to obtain access_token and refresh_token.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body LoginRequest true "Credentials"
// @Success 200 {object} LoginDiscoveryResponse
// @Router /api/auth/login [post]
func (a *Controller) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.HandleValidationError(a.logger, c, err)
		return
	}
	out, err := a.service.Login(req)
	if err != nil {
		responses.HandleError(a.logger, c, err)
		return
	}
	responses.JSON(c, 200, out)
}

// TenantSession godoc
// @Summary Open session for a tenant
// @Description Authorization: Bearer pick_tenant_token from POST /auth/login (not the final access token).
// @Tags auth
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer {pick_tenant_token}"
// @Param body body TenantSessionRequest true "Tenant to use"
// @Success 200 {object} TokenResponse
// @Router /api/auth/tenant-session [post]
func (a *Controller) TenantSession(c *gin.Context) {
	header := c.GetHeader("Authorization")
	raw := strings.TrimSpace(strings.TrimPrefix(header, "Bearer"))
	if raw == "" {
		responses.ErrorJSON(c, http.StatusUnauthorized, "Authorization Bearer pick_tenant_token required")
		return
	}
	var req TenantSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.HandleValidationError(a.logger, c, err)
		return
	}
	out, err := a.service.TenantSession(raw, req.TenantID)
	if err != nil {
		responses.HandleError(a.logger, c, err)
		return
	}
	responses.JSON(c, 200, out)
}

// Refresh godoc
// @Summary Refresh tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param body body RefreshRequest true "Refresh token"
// @Success 200 {object} TokenResponse
// @Router /api/auth/refresh [post]
func (a *Controller) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.HandleValidationError(a.logger, c, err)
		return
	}
	out, err := a.service.Refresh(req.RefreshToken)
	if err != nil {
		responses.HandleError(a.logger, c, err)
		return
	}
	responses.JSON(c, 200, out)
}

// Logout godoc
// @Summary Logout
// @Tags auth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body LogoutRequest true "Refresh token"
// @Success 204 "No Content"
// @Router /api/auth/logout [post]
func (a *Controller) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.HandleValidationError(a.logger, c, err)
		return
	}
	if err := a.service.Logout(req.RefreshToken); err != nil {
		responses.HandleError(a.logger, c, err)
		return
	}
	c.Status(204)
}

// Me godoc
// @Summary Current user
// @Tags auth
// @Security BearerAuth
// @Produce json
// @Success 200 {object} MeResponse
// @Router /api/auth/me [get]
func (a *Controller) Me(c *gin.Context) {
	userUUIDStr, tenantIDStr, _, ok := authContext(c)
	if !ok {
		responses.HandleError(a.logger, c, errorz.ErrUnauthorizedAccess)
		return
	}
	out, err := a.service.Me(userUUIDStr, tenantIDStr)
	if err != nil {
		responses.HandleError(a.logger, c, err)
		return
	}
	responses.JSON(c, 200, out)
}

// ForgotPassword godoc
// @Summary Request password reset
// @Description In ENVIRONMENT=local the response may include reset_token for testing. In production, send the token via your own channel (e.g. email).
// @Tags auth
// @Accept json
// @Produce json
// @Param body body ForgotPasswordRequest true "Email"
// @Success 200 {object} ForgotPasswordResponse
// @Router /api/auth/forgot-password [post]
func (a *Controller) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.HandleValidationError(a.logger, c, err)
		return
	}
	out, err := a.service.ForgotPassword(req.Email)
	if err != nil {
		responses.HandleError(a.logger, c, err)
		return
	}
	responses.JSON(c, 200, out)
}

// ResetPassword godoc
// @Summary Complete password reset
// @Tags auth
// @Accept json
// @Produce json
// @Param body body ResetPasswordRequest true "Token and new password"
// @Success 204 "No Content"
// @Router /api/auth/reset-password [post]
func (a *Controller) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.HandleValidationError(a.logger, c, err)
		return
	}
	if err := a.service.ResetPassword(req.Token, req.NewPassword); err != nil {
		responses.HandleError(a.logger, c, err)
		return
	}
	c.Status(204)
}

// ChangePassword godoc
// @Summary Change password (authenticated)
// @Tags auth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body ChangePasswordRequest true "Passwords"
// @Success 204 "No Content"
// @Router /api/auth/change-password [post]
func (a *Controller) ChangePassword(c *gin.Context) {
	_, _, userDBID, ok := authContext(c)
	if !ok {
		responses.HandleError(a.logger, c, errorz.ErrUnauthorizedAccess)
		return
	}
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.HandleValidationError(a.logger, c, err)
		return
	}
	if err := a.service.ChangePassword(userDBID, req.CurrentPassword, req.NewPassword); err != nil {
		responses.HandleError(a.logger, c, err)
		return
	}
	c.Status(204)
}

// UpdateMe godoc
// @Summary Update current user profile
// @Tags auth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body UpdateMeRequest true "Fields to update"
// @Success 200 {object} MeResponse
// @Router /api/auth/me [patch]
func (a *Controller) UpdateMe(c *gin.Context) {
	userUUIDStr, tenantIDStr, userDBID, ok := authContext(c)
	if !ok {
		responses.HandleError(a.logger, c, errorz.ErrUnauthorizedAccess)
		return
	}
	var req UpdateMeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.HandleValidationError(a.logger, c, err)
		return
	}
	out, err := a.service.UpdateMe(userUUIDStr, tenantIDStr, userDBID, req)
	if err != nil {
		responses.HandleError(a.logger, c, err)
		return
	}
	responses.JSON(c, 200, out)
}

func authContext(c *gin.Context) (userUUID string, tenantID string, userDBID uint, ok bool) {
	u, ok1 := c.Get(framework.UID)
	t, ok2 := c.Get(framework.TenantID)
	dbID, ok3 := c.Get(framework.UserDBID)
	userUUID, okU := u.(string)
	tenantID, okT := t.(string)
	uid, okD := dbID.(uint)
	if !ok1 || !ok2 || !ok3 || !okU || !okT || !okD || userUUID == "" || tenantID == "" || uid == 0 {
		return "", "", 0, false
	}
	return userUUID, tenantID, uid, true
}
