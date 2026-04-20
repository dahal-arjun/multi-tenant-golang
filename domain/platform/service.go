package platform

import (
	"clean-architecture/domain/models"
	"clean-architecture/pkg/infrastructure"
)

// Service provides cross-tenant read models for platform operators (no RLS tenant context).
type Service struct {
	db infrastructure.Database
}

// NewService constructs the service.
func NewService(db infrastructure.Database) *Service {
	return &Service{db: db}
}

// TenantSummary holds coarse counts for dashboards.
type TenantSummary struct {
	TenantCount int64 `json:"tenant_count"`
	UserCount   int64 `json:"user_count"`
}

// GetTenantSummary returns approximate counts across all rows (platform DB session).
func (s *Service) GetTenantSummary() (*TenantSummary, error) {
	var out TenantSummary
	if err := s.db.Model(&models.Tenant{}).Count(&out.TenantCount).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&models.User{}).Count(&out.UserCount).Error; err != nil {
		return nil, err
	}
	return &out, nil
}
