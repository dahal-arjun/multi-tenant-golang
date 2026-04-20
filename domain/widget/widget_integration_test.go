//go:build integration

package widget

import (
	"clean-architecture/domain/auth"
	"clean-architecture/domain/models"
	"clean-architecture/pkg/framework"
	"clean-architecture/pkg/infrastructure"
	"clean-architecture/pkg/jwtutil"
	"clean-architecture/pkg/types"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func setupWidgetPostgres(t *testing.T) (*gorm.DB, *framework.Env) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set (PostgreSQL DSN for integration tests)")
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
		&models.Widget{},
	)
	require.NoError(t, err)

	env := &framework.Env{
		JWTSecret:                 "integration-test-jwt-secret-32chars",
		JWTAccessTTLMinutes:       15,
		JWTRefreshTTLDays:         30,
		PasswordResetTTLMinutes:   60,
		LoginPickTenantTTLMinutes: 10,
		Environment:               "local",
	}
	return db, env
}

func TestIntegration_Postgres_WidgetList(t *testing.T) {
	db, env := setupWidgetPostgres(t)
	log := framework.CreateTestLogger(t)
	repo := auth.NewRepository(infrastructure.Database{DB: db}, log)
	svcAuth := auth.NewService(repo, env, log)

	email := "widget-user@example.com"
	_, err := svcAuth.Register(auth.RegisterRequest{
		Email:      email,
		Password:   "password123",
		TenantName: "Widget Co",
	})
	require.NoError(t, err)

	disc, err := svcAuth.Login(auth.LoginRequest{Email: email, Password: "password123"})
	require.NoError(t, err)
	tok, err := svcAuth.TenantSession(disc.PickTenantToken, disc.Tenants[0].TenantID)
	require.NoError(t, err)
	claims, err := jwtutil.ParseAccess([]byte(env.JWTSecret), tok.AccessToken)
	require.NoError(t, err)
	tid, err := types.ShouldParseUUID(claims.TenantID)
	require.NoError(t, err)

	var u models.User
	require.NoError(t, db.Where("email = ?", email).First(&u).Error)
	actor := u.ID
	w := &models.Widget{
		TenantBound: models.TenantBound{TenantID: tid},
		AuditFields: models.AuditFields{CreatedByID: &actor, UpdatedByID: &actor},
		Title:       "Hello Widget",
	}
	require.NoError(t, db.Create(w).Error)

	wsvc := NewService(infrastructure.Database{DB: db}, log)
	list, err := wsvc.List(tid.String())
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "Hello Widget", list[0].Title)
}
