package middlewares

import (
	"clean-architecture/pkg/audit"
	"clean-architecture/pkg/framework"
	"clean-architecture/pkg/jwtutil"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestJWTAuthMiddleware_missingAuthorization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	env := &framework.Env{JWTSecret: "middleware-test-secret-32chars!!"}
	mw := NewJWTAuthMiddleware(env, framework.CreateTestLogger(t))

	router := gin.New()
	router.GET("/x", mw.Handle(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Contains(t, body["error"].(string), "Unauthorized")
}

func TestJWTAuthMiddleware_invalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	env := &framework.Env{JWTSecret: "middleware-test-secret-32chars!!"}
	mw := NewJWTAuthMiddleware(env, framework.CreateTestLogger(t))

	router := gin.New()
	router.GET("/x", mw.Handle(), func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer not-a-valid-jwt")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestJWTAuthMiddleware_rejectsTenantPickToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := []byte("middleware-test-secret-32chars!!")
	env := &framework.Env{JWTSecret: string(secret)}
	mw := NewJWTAuthMiddleware(env, framework.CreateTestLogger(t))

	pick, err := jwtutil.SignTenantPick(secret, "550e8400-e29b-41d4-a716-446655440000", 1, time.Minute)
	require.NoError(t, err)

	router := gin.New()
	router.GET("/x", mw.Handle(), func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+pick)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestJWTAuthMiddleware_invalidClaims_emptySubject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := []byte("middleware-test-secret-32chars!!")
	env := &framework.Env{JWTSecret: string(secret)}
	mw := NewJWTAuthMiddleware(env, framework.CreateTestLogger(t))

	now := time.Now()
	claims := jwtutil.AccessClaims{
		TenantID:   "660e8400-e29b-41d4-a716-446655440001",
		TenantRole: "owner",
		UserDBID:   42,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "",
			Issuer:    "access",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, &claims)
	s, err := tok.SignedString(secret)
	require.NoError(t, err)

	router := gin.New()
	router.GET("/x", mw.Handle(), func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+s)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "Invalid token claims", body["error"])
}

func TestJWTAuthMiddleware_success_setsContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := []byte("middleware-test-secret-32chars!!")
	env := &framework.Env{JWTSecret: string(secret)}
	mw := NewJWTAuthMiddleware(env, framework.CreateTestLogger(t))

	access, err := jwtutil.SignAccess(
		secret,
		"550e8400-e29b-41d4-a716-446655440000",
		99,
		"660e8400-e29b-41d4-a716-446655440001",
		"admin",
		"",
		[]string{"widgets.read"},
		time.Hour,
	)
	require.NoError(t, err)

	var sawUID any
	var sawTenant any
	var sawRole any
	var sawPerms any
	var sawDBID any
	var actorOK bool
	var actor uint

	router := gin.New()
	router.GET("/x", mw.Handle(), func(c *gin.Context) {
		sawUID, _ = c.Get(framework.UID)
		sawTenant, _ = c.Get(framework.TenantID)
		sawRole, _ = c.Get(framework.Role)
		sawPerms, _ = c.Get(framework.Permissions)
		sawDBID, _ = c.Get(framework.UserDBID)
		actor, actorOK = audit.ActorID(c.Request.Context())
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "550e8400-e29b-41d4-a716-446655440000", sawUID)
	require.Equal(t, "660e8400-e29b-41d4-a716-446655440001", sawTenant)
	require.Equal(t, "admin", sawRole)
	require.Equal(t, []string{"widgets.read"}, sawPerms)
	require.Equal(t, uint(99), sawDBID)
	require.True(t, actorOK)
	require.Equal(t, uint(99), actor)
}
