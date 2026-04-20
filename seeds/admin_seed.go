package seeds

import (
	"clean-architecture/domain/constants"
	"clean-architecture/domain/models"
	"clean-architecture/pkg/framework"
	"clean-architecture/pkg/infrastructure"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AdminSeed creates a platform admin user and default tenant when configured.
type AdminSeed struct {
	logger framework.Logger
	env    *framework.Env
	db     infrastructure.Database
}

// NewAdminSeed constructs AdminSeed.
func NewAdminSeed(logger framework.Logger, env *framework.Env, db infrastructure.Database) AdminSeed {
	return AdminSeed{logger: logger, env: env, db: db}
}

// Setup inserts admin user + tenant + owner membership if missing.
func (s AdminSeed) Setup() {
	email := strings.TrimSpace(strings.ToLower(s.env.AdminEmail))
	password := s.env.AdminPassword

	s.logger.Info("🌱 seeding admin data...")

	if email == "" || password == "" {
		s.logger.Info("skip admin seed: ADMIN_EMAIL or ADMIN_PASSWORD empty")
		return
	}

	var count int64
	if err := s.db.Model(&models.User{}).Where("email = ?", email).Count(&count).Error; err != nil {
		s.logger.Error("admin seed count failed", err.Error())
		return
	}
	if count > 0 {
		s.logger.Info("admin user already exists")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("admin seed bcrypt failed", err.Error())
		return
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		u := &models.User{
			Email:           email,
			PasswordHash:    string(hash),
			Role:            constants.UserRoleSystemManager,
			IsActive:        true,
			IsEmailVerified: true,
		}
		if err := tx.Create(u).Error; err != nil {
			return err
		}
		actor := u.ID
		t := &models.Tenant{
			Name: "Platform",
			Slug: "platform",
			AuditFields: models.AuditFields{
				CreatedByID: &actor,
				UpdatedByID: &actor,
			},
		}
		if err := tx.Create(t).Error; err != nil {
			return err
		}
		m := &models.TenantMembership{
			UserID:   u.ID,
			TenantID: t.ID,
			Role:     constants.TenantRoleOwner,
			AuditFields: models.AuditFields{
				CreatedByID: &actor,
				UpdatedByID: &actor,
			},
		}
		if err := tx.Create(m).Error; err != nil {
			return err
		}
		tid := t.ID
		return tx.Model(u).Updates(map[string]any{
			"tenant_id":     tid,
			"created_by_id": actor,
			"updated_by_id": actor,
		}).Error
	})
	if err != nil {
		s.logger.Error("admin seed failed", err.Error())
		return
	}
	s.logger.Info("admin user and platform tenant created")
}
