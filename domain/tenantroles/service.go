package tenantroles

import (
	"clean-architecture/domain/constants"
	"clean-architecture/domain/models"
	"clean-architecture/pkg/errorz"
	"clean-architecture/pkg/types"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// PermCalculator resolves effective permission keys for JWT claims.
type PermCalculator interface {
	EffectivePermissionKeys(m *models.TenantMembership) ([]string, error)
}

// Service implements PermCalculator and tenant role administration.
type Service struct {
	repo Repository
}

// NewService constructs the service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// EffectivePermissionKeys returns sorted unique keys: builtin(role) ∪ custom role grants.
func (s *Service) EffectivePermissionKeys(m *models.TenantMembership) ([]string, error) {
	base := constants.BuiltinPermissionKeys(m.Role)
	if m.TenantRoleID == nil {
		return base, nil
	}
	custom, err := s.repo.ListPermissionKeysForRole(*m.TenantRoleID)
	if err != nil {
		return nil, err
	}
	return constants.MergePermissionKeys(base, custom), nil
}

// ListRoles returns a page of roles for a tenant and the total count.
func (s *Service) ListRoles(tenantID types.BinaryUUID, offset, limit int) ([]models.TenantRole, int64, error) {
	return s.repo.ListRolesForTenant(tenantID, offset, limit)
}

// CreateRole creates a custom tenant role.
func (s *Service) CreateRole(actorID uint, tenantID types.BinaryUUID, name, description string, slugIn *string) (*models.TenantRole, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errorz.ErrBadRequest.JoinError("name is required")
	}
	slug := slugifyRole(name)
	if slugIn != nil && strings.TrimSpace(*slugIn) != "" {
		slug = slugifyRole(*slugIn)
	}
	var out *models.TenantRole
	err := s.repo.Transaction(func(tx *gorm.DB) error {
		for i := 0; i < 50; i++ {
			candidate := slug
			if i > 0 {
				candidate = fmt.Sprintf("%s-%d", slug, i)
			}
			var count int64
			if err := tx.Model(&models.TenantRole{}).
				Where("tenant_id = ? AND slug = ?", tenantID, candidate).
				Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				continue
			}
			actor := actorID
			row := &models.TenantRole{
				TenantID:    tenantID,
				Name:        name,
				Slug:        candidate,
				Description: strings.TrimSpace(description),
				AuditFields: models.AuditFields{
					CreatedByID: &actor,
					UpdatedByID: &actor,
				},
			}
			if err := tx.Create(row).Error; err != nil {
				return err
			}
			out = row
			return nil
		}
		return errorz.ErrConflict.JoinError("could not allocate unique slug")
	})
	return out, err
}

// UpdateRole patches name/description.
func (s *Service) UpdateRole(actorID uint, tenantID, roleID types.BinaryUUID, name, description *string) (*models.TenantRole, error) {
	row, err := s.repo.FindRoleByID(roleID)
	if err != nil {
		return nil, err
	}
	if row == nil || row.TenantID != tenantID {
		return nil, errorz.ErrRecordNotFound
	}
	updates := map[string]any{"updated_by_id": actorID}
	if name != nil {
		updates["name"] = strings.TrimSpace(*name)
	}
	if description != nil {
		updates["description"] = strings.TrimSpace(*description)
	}
	if err := s.repo.UpdateRole(row, updates); err != nil {
		return nil, err
	}
	return s.repo.FindRoleByID(roleID)
}

// DeleteRole soft-deletes a role if no membership references it.
func (s *Service) DeleteRole(actorID uint, tenantID, roleID types.BinaryUUID) error {
	row, err := s.repo.FindRoleByID(roleID)
	if err != nil {
		return err
	}
	if row == nil || row.TenantID != tenantID {
		return errorz.ErrRecordNotFound
	}
	n, err := s.repo.CountMembershipsUsingRole(roleID)
	if err != nil {
		return err
	}
	if n > 0 {
		return errorz.ErrConflict.JoinError("role is still assigned to members")
	}
	return s.repo.DeleteRole(roleID)
}

// SetRolePermissions replaces permission keys for a role (curated registry only).
func (s *Service) SetRolePermissions(actorID uint, tenantID, roleID types.BinaryUUID, keys []string) error {
	row, err := s.repo.FindRoleByID(roleID)
	if err != nil {
		return err
	}
	if row == nil || row.TenantID != tenantID {
		return errorz.ErrRecordNotFound
	}
	for _, k := range keys {
		if !constants.IsValidPermission(strings.TrimSpace(k)) {
			return errorz.ErrBadRequest.JoinError("unknown permission: " + k)
		}
	}
	return s.repo.Transaction(func(tx *gorm.DB) error {
		if err := s.repo.ReplacePermissions(tx, roleID, keys); err != nil {
			return err
		}
		return tx.Model(&models.TenantRole{}).Where("id = ?", roleID).Update("updated_by_id", actorID).Error
	})
}

// AssignMembershipTenantRole sets or clears the custom tenant_role_id on a membership.
func (s *Service) AssignMembershipTenantRole(actorID uint, tenantID types.BinaryUUID, targetUserID uint, tenantRoleID *types.BinaryUUID) error {
	target, err := s.repo.FindMembership(targetUserID, tenantID)
	if err != nil {
		return err
	}
	if target == nil {
		return errorz.ErrMembershipNotFound
	}
	if tenantRoleID != nil {
		roleRow, err := s.repo.FindRoleByID(*tenantRoleID)
		if err != nil {
			return err
		}
		if roleRow == nil || roleRow.TenantID != tenantID {
			return errorz.ErrBadRequest.JoinError("tenant_role_id is not valid for this tenant")
		}
	}
	return s.repo.Transaction(func(tx *gorm.DB) error {
		return s.repo.UpdateMembershipCustomRole(tx, targetUserID, tenantID, tenantRoleID, actorID)
	})
}
