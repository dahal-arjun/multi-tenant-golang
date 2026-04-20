package auth

import (
	"clean-architecture/domain/models"
	"clean-architecture/pkg/errorz"
	"clean-architecture/pkg/jwtutil"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegister_Login_TenantSession_Refresh_Logout(t *testing.T) {
	_, svc, env := setupAuthSQLite(t)

	reg, err := svc.Register(RegisterRequest{
		Email:      "owner@example.com",
		Password:   "password123",
		TenantName: "Acme Corp",
	})
	require.NoError(t, err)
	require.NotEmpty(t, reg.AccessToken)
	require.NotEmpty(t, reg.RefreshToken)

	claims, err := jwtutil.ParseAccess([]byte(env.JWTSecret), reg.AccessToken)
	require.NoError(t, err)
	require.NotEmpty(t, claims.TenantID)

	disc, err := svc.Login(LoginRequest{Email: "owner@example.com", Password: "password123"})
	require.NoError(t, err)
	require.Len(t, disc.Tenants, 1)
	require.Equal(t, "Acme Corp", disc.Tenants[0].Name)
	require.NotEmpty(t, disc.PickTenantToken)

	tokens, err := svc.TenantSession(disc.PickTenantToken, disc.Tenants[0].TenantID)
	require.NoError(t, err)
	require.NotEmpty(t, tokens.AccessToken)

	refreshed, err := svc.Refresh(tokens.RefreshToken)
	require.NoError(t, err)
	require.NotEmpty(t, refreshed.AccessToken)

	require.NoError(t, svc.Logout(refreshed.RefreshToken))
	_, err = svc.Refresh(refreshed.RefreshToken)
	require.Error(t, err)
	require.True(t, errors.Is(err, errorz.ErrInvalidRefreshToken))
}

func TestLogin_invalidPassword(t *testing.T) {
	_, svc, _ := setupAuthSQLite(t)
	_, err := svc.Register(RegisterRequest{
		Email:      "u@example.com",
		Password:   "password123",
		TenantName: "T1",
	})
	require.NoError(t, err)

	_, err = svc.Login(LoginRequest{Email: "u@example.com", Password: "wrong"})
	require.Error(t, err)
	require.True(t, errors.Is(err, errorz.ErrInvalidCredentials))
}

func TestLogin_noTenants_noPickToken(t *testing.T) {
	repo, svc, _ := setupAuthSQLite(t)
	_, err := svc.Register(RegisterRequest{
		Email:      "orphan@example.com",
		Password:   "password123",
		TenantName: "WillStripMembership",
	})
	require.NoError(t, err)
	u, err := repo.FindUserByEmail("orphan@example.com")
	require.NoError(t, err)
	require.NotNil(t, u)
	require.NoError(t, repo.Where("user_id = ?", u.ID).Delete(&models.TenantMembership{}).Error)

	disc, err := svc.Login(LoginRequest{Email: "orphan@example.com", Password: "password123"})
	require.NoError(t, err)
	require.Empty(t, disc.Tenants)
	require.Empty(t, disc.PickTenantToken)
	require.Zero(t, disc.PickTenantExpiresIn)
}

func TestTenantSession_invalidPickToken(t *testing.T) {
	_, svc, _ := setupAuthSQLite(t)
	_, err := svc.TenantSession("not-a-jwt", "00000000-0000-0000-0000-000000000001")
	require.Error(t, err)
	require.True(t, errors.Is(err, errorz.ErrInvalidPickToken))
}

func TestTenantSession_wrongTenant(t *testing.T) {
	_, svc, _ := setupAuthSQLite(t)
	_, err := svc.Register(RegisterRequest{
		Email:      "a@example.com",
		Password:   "password123",
		TenantName: "A",
	})
	require.NoError(t, err)
	_, err = svc.Register(RegisterRequest{
		Email:      "b@example.com",
		Password:   "password123",
		TenantName: "B",
	})
	require.NoError(t, err)

	disc, err := svc.Login(LoginRequest{Email: "a@example.com", Password: "password123"})
	require.NoError(t, err)
	require.Len(t, disc.Tenants, 1)

	other := svc // same DB
	discB, err := other.Login(LoginRequest{Email: "b@example.com", Password: "password123"})
	require.NoError(t, err)
	require.Len(t, discB.Tenants, 1)

	_, err = svc.TenantSession(disc.PickTenantToken, discB.Tenants[0].TenantID)
	require.Error(t, err)
	require.True(t, errors.Is(err, errorz.ErrMembershipNotFound))
}

func TestRegister_weakPassword(t *testing.T) {
	_, svc, _ := setupAuthSQLite(t)
	_, err := svc.Register(RegisterRequest{
		Email:      "x@example.com",
		Password:   "short",
		TenantName: "T",
	})
	require.Error(t, err)
	require.True(t, errors.Is(err, errorz.ErrWeakPassword))
}

func TestRegister_emailTaken(t *testing.T) {
	_, svc, _ := setupAuthSQLite(t)
	req := RegisterRequest{
		Email:      "dup@example.com",
		Password:   "password123",
		TenantName: "First",
	}
	_, err := svc.Register(req)
	require.NoError(t, err)
	_, err = svc.Register(RegisterRequest{
		Email:      "dup@example.com",
		Password:   "password123",
		TenantName: "Second",
	})
	require.Error(t, err)
	require.True(t, errors.Is(err, errorz.ErrEmailTaken))
}

func TestMe_and_UpdateMe(t *testing.T) {
	repo, svc, env := setupAuthSQLite(t)
	_, err := svc.Register(RegisterRequest{
		Email:      "me@example.com",
		Password:   "password123",
		TenantName: "My Org",
	})
	require.NoError(t, err)
	u, err := repo.FindUserByEmail("me@example.com")
	require.NoError(t, err)
	require.NotNil(t, u)

	disc, err := svc.Login(LoginRequest{Email: "me@example.com", Password: "password123"})
	require.NoError(t, err)
	tok, err := svc.TenantSession(disc.PickTenantToken, disc.Tenants[0].TenantID)
	require.NoError(t, err)
	claims, err := jwtutil.ParseAccess([]byte(env.JWTSecret), tok.AccessToken)
	require.NoError(t, err)

	me, err := svc.Me(claims.Subject, claims.TenantID)
	require.NoError(t, err)
	require.Equal(t, "me@example.com", me.User.Email)
	require.Equal(t, claims.TenantID, me.TenantID)

	jane := "Jane"
	me2, err := svc.UpdateMe(claims.Subject, claims.TenantID, u.ID, UpdateMeRequest{FirstName: &jane})
	require.NoError(t, err)
	require.Equal(t, "Jane", me2.User.FirstName)
}

func TestForgotPassword_ResetPassword_local(t *testing.T) {
	_, svc, _ := setupAuthSQLite(t)
	_, err := svc.Register(RegisterRequest{
		Email:      "reset@example.com",
		Password:   "password123",
		TenantName: "R",
	})
	require.NoError(t, err)

	resp, err := svc.ForgotPassword("reset@example.com")
	require.NoError(t, err)
	require.NotNil(t, resp.ResetToken)

	require.NoError(t, svc.ResetPassword(*resp.ResetToken, "newpass9999"))

	_, err = svc.Login(LoginRequest{Email: "reset@example.com", Password: "password123"})
	require.Error(t, err)

	disc, err := svc.Login(LoginRequest{Email: "reset@example.com", Password: "newpass9999"})
	require.NoError(t, err)
	require.NotEmpty(t, disc.PickTenantToken)
}

func TestChangePassword(t *testing.T) {
	_, svc, env := setupAuthSQLite(t)
	_, err := svc.Register(RegisterRequest{
		Email:      "chg@example.com",
		Password:   "password123",
		TenantName: "C",
	})
	require.NoError(t, err)
	disc, err := svc.Login(LoginRequest{Email: "chg@example.com", Password: "password123"})
	require.NoError(t, err)
	tok, err := svc.TenantSession(disc.PickTenantToken, disc.Tenants[0].TenantID)
	require.NoError(t, err)
	claims, err := jwtutil.ParseAccess([]byte(env.JWTSecret), tok.AccessToken)
	require.NoError(t, err)

	require.NoError(t, svc.ChangePassword(uint(claims.UserDBID), "password123", "updated9999"))

	_, err = svc.Login(LoginRequest{Email: "chg@example.com", Password: "password123"})
	require.Error(t, err)

	_, err = svc.Login(LoginRequest{Email: "chg@example.com", Password: "updated9999"})
	require.NoError(t, err)
}
