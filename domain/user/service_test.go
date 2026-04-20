package user

import (
	"clean-architecture/domain/models"
	"clean-architecture/pkg/framework"
	"clean-architecture/pkg/infrastructure"
	"clean-architecture/pkg/types"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func setupUserSQLite(t *testing.T) (*Service, Repository) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}))

	repo := NewRepository(infrastructure.Database{DB: db}, framework.CreateTestLogger(t))
	svc := NewService(framework.CreateTestLogger(t), repo)
	return svc, repo
}

func TestCreate_and_GetUserByUUID(t *testing.T) {
	svc, _ := setupUserSQLite(t)
	id := uuid.New()
	u := &models.User{
		UUID:         types.BinaryUUID(id),
		Email:        "user-svc@example.com",
		PasswordHash: "x",
		IsActive:     true,
	}
	require.NoError(t, svc.Create(u))

	got, err := svc.GetUserByUUID(types.BinaryUUID(id))
	require.NoError(t, err)
	require.Equal(t, "user-svc@example.com", got.Email)
}
