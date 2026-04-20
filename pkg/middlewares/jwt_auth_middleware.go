package middlewares

import (
	"clean-architecture/pkg/audit"
	"clean-architecture/pkg/errorz"
	"clean-architecture/pkg/framework"
	"clean-architecture/pkg/jwtutil"
	"clean-architecture/pkg/responses"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// JWTAuthMiddleware validates Bearer JWT access tokens.
type JWTAuthMiddleware struct {
	env    *framework.Env
	logger framework.Logger
}

// NewJWTAuthMiddleware constructs JWT auth middleware.
func NewJWTAuthMiddleware(env *framework.Env, logger framework.Logger) JWTAuthMiddleware {
	return JWTAuthMiddleware{env: env, logger: logger}
}

// Handle returns a gin middleware that loads claims into the context.
func (m JWTAuthMiddleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		raw := strings.TrimSpace(strings.TrimPrefix(header, "Bearer"))
		if raw == "" {
			responses.HandleError(m.logger, c, errorz.ErrUnauthorizedAccess)
			c.Abort()
			return
		}
		claims, err := jwtutil.ParseAccess([]byte(m.env.JWTSecret), raw)
		if err != nil {
			responses.HandleError(m.logger, c, errorz.ErrUnauthorizedAccess)
			c.Abort()
			return
		}
		if claims.Subject == "" || claims.TenantID == "" || claims.UserDBID <= 0 {
			responses.ErrorJSON(c, http.StatusUnauthorized, "Invalid token claims")
			c.Abort()
			return
		}
		actor := uint(claims.UserDBID)
		c.Set(framework.UID, claims.Subject)
		c.Set(framework.TenantID, claims.TenantID)
		c.Set(framework.Role, claims.TenantRole)
		c.Set(framework.UserDBID, actor)
		c.Request = c.Request.WithContext(audit.WithActorID(c.Request.Context(), actor))
		c.Next()
	}
}
