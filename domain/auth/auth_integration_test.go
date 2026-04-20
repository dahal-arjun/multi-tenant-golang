//go:build integration

package auth

import (
	"clean-architecture/domain/models"
	"clean-architecture/domain/tenantroles"
	"clean-architecture/pkg/errorz"
	"clean-architecture/pkg/framework"
	"clean-architecture/pkg/infrastructure"
	"clean-architecture/pkg/jwtutil"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func setupPostgres(t *testing.T) (Repository, *Service, *framework.Env) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set (PostgreSQL DSN, e.g. postgres://user:pass@localhost:5432/dbname?sslmode=disable)")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	err = db.AutoMigrate(
		&models.User{},
		&models.Tenant{},
		&models.TenantMembership{},
		&models.RefreshToken{},
		&models.PasswordResetToken{},
		&models.EmailVerificationToken{},
		&models.TenantInvitation{},
		&models.TenantRole{},
		&models.TenantRolePermission{},
		&models.Widget{},
	)
	require.NoError(t, err)

	env := &framework.Env{
		JWTSecret:                 "integration-test-jwt-secret-32chars",
		JWTAccessTTLMinutes:       15,
		JWTRefreshTTLDays:       30,
		PasswordResetTTLMinutes:   60,
		LoginPickTenantTTLMinutes: 10,
		Environment:               "local",
	}
	log := framework.CreateTestLogger(t)
	repo := NewRepository(infrastructure.Database{DB: db}, log)
	trepo := tenantroles.NewRepository(infrastructure.Database{DB: db}, log)
	tcalc := tenantroles.NewService(trepo)
	svc := NewService(repo, env, log, tcalc)
	return repo, svc, env
}

func TestIntegration_Postgres_RegisterLoginTenantSessionRefresh(t *testing.T) {
	_, svc, env := setupPostgres(t)

	email := "integration-auth@example.com"
	reg, err := svc.Register(RegisterRequest{
		Email:      email,
		Password:   "password123",
		TenantName: "Integration Org",
	})
	require.NoError(t, err)
	claims, err := jwtutil.ParseAccess([]byte(env.JWTSecret), reg.AccessToken)
	require.NoError(t, err)

	disc, err := svc.Login(LoginRequest{Email: email, Password: "password123"})
	require.NoError(t, err)
	require.Len(t, disc.Tenants, 1)
	require.Equal(t, claims.TenantID, disc.Tenants[0].TenantID)

	tokens, err := svc.TenantSession(disc.PickTenantToken, disc.Tenants[0].TenantID)
	require.NoError(t, err)
	refreshed, err := svc.Refresh(tokens.RefreshToken)
	require.NoError(t, err)
	require.NoError(t, svc.Logout(refreshed.RefreshToken))
	_, err = svc.Refresh(refreshed.RefreshToken)
	require.Error(t, err)
	require.True(t, errors.Is(err, errorz.ErrInvalidRefreshToken))
}
