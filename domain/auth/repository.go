package auth

import (
	"clean-architecture/domain/models"
	"clean-architecture/pkg/framework"
	"clean-architecture/pkg/infrastructure"
	"clean-architecture/pkg/types"
	"errors"
	"time"

	"gorm.io/gorm"
)

// Repository persists auth-related data.
type Repository struct {
	infrastructure.Database
	logger framework.Logger
}

// NewRepository constructs the auth repository.
func NewRepository(db infrastructure.Database, logger framework.Logger) Repository {
	return Repository{Database: db, logger: logger}
}

// FindUserByEmail returns the user or nil if not found (no error for missing).
func (r *Repository) FindUserByEmail(email string) (*models.User, error) {
	var u models.User
	err := r.Where("email = ?", email).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// FindUserByID loads a user by primary key.
func (r *Repository) FindUserByID(id uint) (*models.User, error) {
	var u models.User
	err := r.First(&u, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// FindUserByUUID loads a user by public uuid.
func (r *Repository) FindUserByUUID(id types.BinaryUUID) (*models.User, error) {
	var u models.User
	err := r.Where("uuid = ?", id).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// ListMembershipsForUser returns all active tenant memberships for a user.
func (r *Repository) ListMembershipsForUser(userID uint) ([]models.TenantMembership, error) {
	var ms []models.TenantMembership
	err := r.Where("user_id = ?", userID).Find(&ms).Error
	return ms, err
}

// FindMembership returns active membership for user and tenant.
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

// FindTenantByID loads tenant by primary key uuid.
func (r *Repository) FindTenantByID(id types.BinaryUUID) (*models.Tenant, error) {
	var t models.Tenant
	err := r.Where("id = ?", id).First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// TenantSlugExists returns true if slug is taken.
func (r *Repository) TenantSlugExists(slug string) (bool, error) {
	var count int64
	err := r.Model(&models.Tenant{}).Where("slug = ?", slug).Count(&count).Error
	return count > 0, err
}

// CreateRefreshToken persists a refresh token row.
func (r *Repository) CreateRefreshToken(tx *gorm.DB, row *models.RefreshToken) error {
	return tx.Create(row).Error
}

// FindValidRefreshByHash finds a non-revoked, non-expired refresh token.
func (r *Repository) FindValidRefreshByHash(hash string) (*models.RefreshToken, error) {
	var row models.RefreshToken
	err := r.Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", hash, time.Now().UTC()).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// RevokeRefresh marks a refresh token revoked.
func (r *Repository) RevokeRefresh(tx *gorm.DB, id types.BinaryUUID) error {
	now := time.Now()
	return tx.Model(&models.RefreshToken{}).Where("id = ?", id).Update("revoked_at", now).Error
}

// RevokeAllRefreshTokensForUser revokes every active refresh token for the user.
func (r *Repository) RevokeAllRefreshTokensForUser(tx *gorm.DB, userID uint) error {
	now := time.Now()
	return tx.Model(&models.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now).Error
}

// InvalidatePendingPasswordResets marks unused reset tokens as used for the user.
func (r *Repository) InvalidatePendingPasswordResets(tx *gorm.DB, userID uint) error {
	now := time.Now()
	return tx.Model(&models.PasswordResetToken{}).
		Where("user_id = ? AND used_at IS NULL", userID).
		Update("used_at", now).Error
}

// CreatePasswordResetToken persists a password reset row.
func (r *Repository) CreatePasswordResetToken(tx *gorm.DB, row *models.PasswordResetToken) error {
	return tx.Create(row).Error
}

// FindValidPasswordResetByHash returns an unused, non-expired reset token row.
func (r *Repository) FindValidPasswordResetByHash(hash string) (*models.PasswordResetToken, error) {
	var row models.PasswordResetToken
	err := r.Where("token_hash = ? AND used_at IS NULL AND expires_at > ?", hash, time.Now().UTC()).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// MarkPasswordResetUsed sets used_at on a reset token.
func (r *Repository) MarkPasswordResetUsed(tx *gorm.DB, id types.BinaryUUID) error {
	now := time.Now()
	return tx.Model(&models.PasswordResetToken{}).Where("id = ?", id).Update("used_at", now).Error
}
