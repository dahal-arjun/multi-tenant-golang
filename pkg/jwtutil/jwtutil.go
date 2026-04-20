package jwtutil

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	accessJWTIssuer     = "access"
	tenantPickJWTIssuer = "login-tenant-pick"
)

// AccessClaims is encoded in JWT access tokens.
type AccessClaims struct {
	TenantID   string `json:"tid"`
	TenantRole string `json:"trole"`
	UserDBID   int64  `json:"idb"`
	jwt.RegisteredClaims
}

// TenantPickClaims is a short-lived token after password login; use with POST /auth/tenant-session only.
type TenantPickClaims struct {
	UserDBID int64 `json:"idb"`
	jwt.RegisteredClaims
}

// SignAccess creates a signed HS256 access token.
func SignAccess(secret []byte, userUUID string, userDBID int64, tenantID string, tenantRole string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := AccessClaims{
		TenantID:   tenantID,
		TenantRole: tenantRole,
		UserDBID:   userDBID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userUUID,
			Issuer:    accessJWTIssuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, &claims)
	return t.SignedString(secret)
}

// SignTenantPick creates a short-lived token used to choose a tenant after login.
func SignTenantPick(secret []byte, userUUID string, userDBID int64, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := TenantPickClaims{
		UserDBID: userDBID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userUUID,
			Issuer:    tenantPickJWTIssuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, &claims)
	return t.SignedString(secret)
}

// ParseAccess validates an access token and returns claims.
func ParseAccess(secret []byte, tokenStr string) (*AccessClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &AccessClaims{}, func(t *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*AccessClaims)
	if !ok || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	if claims.Issuer == tenantPickJWTIssuer || claims.TenantID == "" || claims.UserDBID <= 0 {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}

// ParseTenantPick validates a tenant-pick token from the login step.
func ParseTenantPick(secret []byte, tokenStr string) (*TenantPickClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &TenantPickClaims{}, func(t *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*TenantPickClaims)
	if !ok || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	if claims.Issuer != tenantPickJWTIssuer || claims.Subject == "" || claims.UserDBID <= 0 {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}
