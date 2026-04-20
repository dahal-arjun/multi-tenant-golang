package auth

import (
	"clean-architecture/domain/models"
	"clean-architecture/pkg/framework"
	"clean-architecture/pkg/infrastructure"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func setupAuthSQLite(t *testing.T) (Repository, *Service, *framework.Env) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
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
	)
	require.NoError(t, err)

	env := &framework.Env{
		JWTSecret:                 "unit-test-jwt-secret-32chars!!",
		JWTAccessTTLMinutes:       15,
		JWTRefreshTTLDays:         30,
		PasswordResetTTLMinutes:   60,
		LoginPickTenantTTLMinutes: 10,
		Environment:               "local",
	}
	repo := NewRepository(infrastructure.Database{DB: db}, framework.CreateTestLogger(t))
	svc := NewService(repo, env, framework.CreateTestLogger(t))
	return repo, svc, env
}
