package middlewares

import (
	"clean-architecture/domain/constants"
	"clean-architecture/pkg/errorz"
	"clean-architecture/pkg/framework"
	"clean-architecture/pkg/responses"
	"slices"

	"github.com/gin-gonic/gin"
)

// RequirePermission returns middleware that requires a curated permission in the tenant JWT.
func RequirePermission(logger framework.Logger, perm constants.Permission) gin.HandlerFunc {
	key := string(perm)
	return func(c *gin.Context) {
		raw, ok := c.Get(framework.Permissions)
		if !ok {
			responses.HandleError(logger, c, errorz.ErrForbiddenAccess)
			c.Abort()
			return
		}
		perms, ok := raw.([]string)
		if !ok || !slices.Contains(perms, key) {
			responses.HandleError(logger, c, errorz.ErrForbiddenAccess)
			c.Abort()
			return
		}
		c.Next()
	}
}
