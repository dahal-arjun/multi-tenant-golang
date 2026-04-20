package jwtutil

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestSignParseAccess_roundTrip(t *testing.T) {
	secret := []byte("test-secret-at-least-32-bytes-long!")
	ttl := 15 * time.Minute
	tok, err := SignAccess(secret, "550e8400-e29b-41d4-a716-446655440000", 42, "660e8400-e29b-41d4-a716-446655440001", "owner", ttl)
	require.NoError(t, err)

	claims, err := ParseAccess(secret, tok)
	require.NoError(t, err)
	require.Equal(t, "550e8400-e29b-41d4-a716-446655440000", claims.Subject)
	require.Equal(t, int64(42), claims.UserDBID)
	require.Equal(t, "660e8400-e29b-41d4-a716-446655440001", claims.TenantID)
	require.Equal(t, "owner", claims.TenantRole)
	require.Equal(t, accessJWTIssuer, claims.Issuer)
}

func TestParseAccess_rejectsPickToken(t *testing.T) {
	secret := []byte("test-secret-at-least-32-bytes-long!")
	pick, err := SignTenantPick(secret, "550e8400-e29b-41d4-a716-446655440000", 7, time.Minute)
	require.NoError(t, err)

	_, err = ParseAccess(secret, pick)
	require.Error(t, err)
}

func TestParseAccess_rejectsEmptyTenantID(t *testing.T) {
	secret := []byte("test-secret-at-least-32-bytes-long!")
	now := time.Now()
	claims := AccessClaims{
		TenantID:   "",
		TenantRole: "owner",
		UserDBID:   1,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "550e8400-e29b-41d4-a716-446655440000",
			Issuer:    accessJWTIssuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, &claims)
	s, err := tok.SignedString(secret)
	require.NoError(t, err)

	_, err = ParseAccess(secret, s)
	require.Error(t, err)
}

func TestSignParseTenantPick_roundTrip(t *testing.T) {
	secret := []byte("test-secret-at-least-32-bytes-long!")
	ttl := 10 * time.Minute
	tok, err := SignTenantPick(secret, "550e8400-e29b-41d4-a716-446655440000", 99, ttl)
	require.NoError(t, err)

	claims, err := ParseTenantPick(secret, tok)
	require.NoError(t, err)
	require.Equal(t, tenantPickJWTIssuer, claims.Issuer)
	require.Equal(t, "550e8400-e29b-41d4-a716-446655440000", claims.Subject)
	require.Equal(t, int64(99), claims.UserDBID)
}

func TestParseTenantPick_rejectsAccessToken(t *testing.T) {
	secret := []byte("test-secret-at-least-32-bytes-long!")
	access, err := SignAccess(secret, "550e8400-e29b-41d4-a716-446655440000", 1, "660e8400-e29b-41d4-a716-446655440001", "member", time.Minute)
	require.NoError(t, err)

	_, err = ParseTenantPick(secret, access)
	require.Error(t, err)
}

func TestParseTenantPick_expired(t *testing.T) {
	secret := []byte("test-secret-at-least-32-bytes-long!")
	tok, err := SignTenantPick(secret, "550e8400-e29b-41d4-a716-446655440000", 1, -time.Hour)
	require.NoError(t, err)

	_, err = ParseTenantPick(secret, tok)
	require.Error(t, err)
}
