package middlewares

import (
	"clean-architecture/domain/constants"
	"clean-architecture/pkg/framework"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRequirePermission_allows(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := framework.CreateTestLogger(t)

	router := gin.New()
	router.GET("/x",
		func(c *gin.Context) {
			c.Set(framework.Permissions, []string{"widgets.read", "widgets.write"})
			c.Next()
		},
		RequirePermission(logger, constants.PermissionWidgetsRead),
		func(c *gin.Context) {
			c.Status(http.StatusOK)
		},
	)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestRequirePermission_deniesWhenMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := framework.CreateTestLogger(t)

	router := gin.New()
	router.GET("/x",
		func(c *gin.Context) {
			c.Set(framework.Permissions, []string{"widgets.read"})
			c.Next()
		},
		RequirePermission(logger, constants.PermissionWidgetsWrite),
		func(c *gin.Context) {
			c.Status(http.StatusOK)
		},
	)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRequirePermission_deniesWhenNoPermissionsContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := framework.CreateTestLogger(t)

	router := gin.New()
	router.GET("/x",
		RequirePermission(logger, constants.PermissionWidgetsRead),
		func(c *gin.Context) {
			c.Status(http.StatusOK)
		},
	)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	require.Equal(t, http.StatusForbidden, rec.Code)
}
