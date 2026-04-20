package widget

import (
	"clean-architecture/domain/models"
	"clean-architecture/pkg/dbscope"
	"clean-architecture/pkg/framework"
	"clean-architecture/pkg/infrastructure"
	"clean-architecture/pkg/tenancy"

	"gorm.io/gorm"
)

// Service lists tenant-scoped widgets under PostgreSQL RLS.
type Service struct {
	db     infrastructure.Database
	logger framework.Logger
}

// NewService constructs the widget service.
func NewService(db infrastructure.Database, logger framework.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// List returns widgets visible for the tenant (RLS enforced inside transaction).
func (s *Service) List(tenantID string) ([]models.Widget, error) {
	var out []models.Widget
	err := tenancy.WithTenant(s.db.DB, tenantID, func(tx *gorm.DB) error {
		return tx.Scopes(dbscope.TenantID(tenantID)).Order("created_at desc").Find(&out).Error
	})
	return out, err
}
