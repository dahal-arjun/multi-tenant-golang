package tenantroles

import (
	"clean-architecture/domain/models"
	"clean-architecture/pkg/framework"
	"clean-architecture/pkg/infrastructure"
	"clean-architecture/pkg/types"
	"errors"
	"strings"

	"gorm.io/gorm"
)

// Repository persists tenant role definitions and grants.
type Repository struct {
	infrastructure.Database
	logger framework.Logger
}

// NewRepository constructs the repository.
func NewRepository(db infrastructure.Database, logger framework.Logger) Repository {
	return Repository{Database: db, logger: logger}
}

// ListRolesForTenant returns a page of non-deleted roles for a tenant and the total count.
func (r *Repository) ListRolesForTenant(tenantID types.BinaryUUID, offset, limit int) ([]models.TenantRole, int64, error) {
	var total int64
	if err := r.Model(&models.TenantRole{}).Where("tenant_id = ?", tenantID).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []models.TenantRole
	err := r.Where("tenant_id = ?", tenantID).Order("slug").Offset(offset).Limit(limit).Find(&rows).Error
	return rows, total, err
}

// FindRoleByID loads a role or nil.
func (r *Repository) FindRoleByID(id types.BinaryUUID) (*models.TenantRole, error) {
	var row models.TenantRole
	err := r.Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// FindRoleByTenantAndSlug loads a role by tenant + slug.
func (r *Repository) FindRoleByTenantAndSlug(tenantID types.BinaryUUID, slug string) (*models.TenantRole, error) {
	var row models.TenantRole
	err := r.Where("tenant_id = ? AND slug = ?", tenantID, slug).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// UpdateRole patches fields on a role.
func (r *Repository) UpdateRole(row *models.TenantRole, updates map[string]any) error {
	return r.Model(row).Updates(updates).Error
}

// DeleteRole soft-deletes a role.
func (r *Repository) DeleteRole(id types.BinaryUUID) error {
	return r.Where("id = ?", id).Delete(&models.TenantRole{}).Error
}

// ReplacePermissions sets the full permission set for a role (transactional caller).
func (r *Repository) ReplacePermissions(tx *gorm.DB, tenantRoleID types.BinaryUUID, keys []string) error {
	if err := tx.Where("tenant_role_id = ?", tenantRoleID).Delete(&models.TenantRolePermission{}).Error; err != nil {
		return err
	}
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		p := &models.TenantRolePermission{
			TenantRoleID:  tenantRoleID,
			PermissionKey: k,
		}
		if err := tx.Create(p).Error; err != nil {
			return err
		}
	}
	return nil
}

// ListPermissionKeysForRole returns granted permission keys for a tenant role.
func (r *Repository) ListPermissionKeysForRole(tenantRoleID types.BinaryUUID) ([]string, error) {
	var rows []models.TenantRolePermission
	if err := r.Where("tenant_role_id = ?", tenantRoleID).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].PermissionKey)
	}
	return out, nil
}

// CountMembershipsUsingRole returns memberships referencing the role.
func (r *Repository) CountMembershipsUsingRole(roleID types.BinaryUUID) (int64, error) {
	var n int64
	err := r.Model(&models.TenantMembership{}).Where("tenant_role_id = ?", roleID).Count(&n).Error
	return n, err
}

// UpdateMembershipCustomRole sets tenant_role_id on a membership.
func (r *Repository) UpdateMembershipCustomRole(tx *gorm.DB, userID uint, tenantID types.BinaryUUID, roleID *types.BinaryUUID, actorID uint) error {
	return tx.Model(&models.TenantMembership{}).
		Where("user_id = ? AND tenant_id = ?", userID, tenantID).
		Updates(map[string]any{"tenant_role_id": roleID, "updated_by_id": actorID}).Error
}

// FindMembership returns membership for user in tenant.
func (r *Repository) FindMembership(userID uint, tenantID types.BinaryUUID) (*models.TenantMembership, error) {
	var m models.TenantMembership
	err := r.Where("user_id = ? AND tenant_id = ?", userID, tenantID).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}
