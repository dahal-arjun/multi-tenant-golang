package widget

import (
	"clean-architecture/domain/models"
	"clean-architecture/pkg/dbscope"
	"clean-architecture/pkg/framework"
	"clean-architecture/pkg/infrastructure"
	"clean-architecture/pkg/tenancy"
	"clean-architecture/pkg/utils"

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

// List returns a page of widgets visible for the tenant (RLS enforced inside transaction) and the total count for the tenant.
func (s *Service) List(tenantID string, p utils.Pagination) ([]models.Widget, int64, error) {
	var out []models.Widget
	var total int64
	err := tenancy.WithTenant(s.db.DB, tenantID, func(tx *gorm.DB) error {
		q := tx.Model(&models.Widget{}).Scopes(dbscope.TenantID(tenantID))
		if err := q.Count(&total).Error; err != nil {
			return err
		}
		return tx.Scopes(dbscope.TenantID(tenantID)).
			Order("created_at desc").
			Offset(p.Offset).
			Limit(p.Limit).
			Find(&out).Error
	})
	return out, total, err
}
