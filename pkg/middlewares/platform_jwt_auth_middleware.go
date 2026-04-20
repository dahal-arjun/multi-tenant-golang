package middlewares

import (
	"clean-architecture/domain/constants"
	"clean-architecture/pkg/audit"
	"clean-architecture/pkg/errorz"
	"clean-architecture/pkg/framework"
	"clean-architecture/pkg/jwtutil"
	"clean-architecture/pkg/responses"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// PlatformJWTAuthMiddleware validates Bearer platform access tokens (no tenant).
type PlatformJWTAuthMiddleware struct {
	env    *framework.Env
	logger framework.Logger
}

// NewPlatformJWTAuthMiddleware constructs platform JWT middleware.
func NewPlatformJWTAuthMiddleware(env *framework.Env, logger framework.Logger) PlatformJWTAuthMiddleware {
	return PlatformJWTAuthMiddleware{env: env, logger: logger}
}

// Handle returns middleware that loads platform claims into context (no TenantID).
func (m PlatformJWTAuthMiddleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		raw := strings.TrimSpace(strings.TrimPrefix(header, "Bearer"))
		if raw == "" {
			responses.HandleError(m.logger, c, errorz.ErrUnauthorizedAccess)
			c.Abort()
			return
		}
		claims, err := jwtutil.ParsePlatformAccess([]byte(m.env.JWTSecret), raw)
		if err != nil {
			responses.HandleError(m.logger, c, errorz.ErrUnauthorizedAccess)
			c.Abort()
			return
		}
		role := constants.UserRole(claims.PlatformRole)
		if !constants.IsPlatformStaff(role) {
			responses.ErrorJSON(c, http.StatusUnauthorized, "Invalid platform token role")
			c.Abort()
			return
		}
		actor := uint(claims.UserDBID)
		c.Set(framework.UID, claims.Subject)
		c.Set(framework.PlatformUserRole, claims.PlatformRole)
		c.Set(framework.UserDBID, actor)
		c.Request = c.Request.WithContext(audit.WithActorID(c.Request.Context(), actor))
		c.Next()
	}
}
