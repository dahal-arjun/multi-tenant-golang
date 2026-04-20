package tenantroles

import (
	"clean-architecture/domain/constants"
	"clean-architecture/domain/models"
	"clean-architecture/pkg/framework"
	"clean-architecture/pkg/infrastructure"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func setupTenantRolesSQLite(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.Tenant{},
		&models.TenantMembership{},
		&models.TenantRole{},
		&models.TenantRolePermission{},
	))
	log := framework.CreateTestLogger(t)
	repo := NewRepository(infrastructure.Database{DB: db}, log)
	return NewService(repo), db
}

func TestEffectivePermissionKeys_memberUnionCustomGrants(t *testing.T) {
	svc, db := setupTenantRolesSQLite(t)
	hash, err := bcrypt.GenerateFromPassword([]byte("x"), bcrypt.DefaultCost)
	require.NoError(t, err)
	u := &models.User{
		Email:           "m@example.com",
		PasswordHash:    string(hash),
		IsActive:        true,
		IsEmailVerified: true,
	}
	require.NoError(t, db.Create(u).Error)

	tenant := &models.Tenant{Name: "Co", Slug: "co"}
	require.NoError(t, db.Create(tenant).Error)

	custom := &models.TenantRole{
		TenantID: tenant.ID,
		Name:     "Writer",
		Slug:     "writer",
	}
	require.NoError(t, db.Create(custom).Error)

	require.NoError(t, db.Create(&models.TenantRolePermission{
		TenantRoleID:  custom.ID,
		PermissionKey: string(constants.PermissionWidgetsWrite),
	}).Error)

	rid := custom.ID
	m := &models.TenantMembership{
		UserID:       u.ID,
		TenantID:     tenant.ID,
		Role:         constants.TenantRoleMember,
		TenantRoleID: &rid,
	}
	require.NoError(t, db.Create(m).Error)

	keys, err := svc.EffectivePermissionKeys(m)
	require.NoError(t, err)
	require.True(t, slices.Contains(keys, string(constants.PermissionWidgetsRead)))
	require.True(t, slices.Contains(keys, string(constants.PermissionWidgetsWrite)))
}

func TestEffectivePermissionKeys_ownerIgnoresEmptyCustomRoleID(t *testing.T) {
	svc, _ := setupTenantRolesSQLite(t)
	m := &models.TenantMembership{
		Role: constants.TenantRoleOwner,
	}
	keys, err := svc.EffectivePermissionKeys(m)
	require.NoError(t, err)
	require.Contains(t, keys, string(constants.PermissionWidgetsRead))
	require.Contains(t, keys, string(constants.PermissionRolesManage))
}
