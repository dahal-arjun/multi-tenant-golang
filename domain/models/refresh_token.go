package models

import (
	"clean-architecture/pkg/types"
	"time"

	_ "ariga.io/atlas-provider-gorm/gormschema"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RefreshToken stores hashed refresh tokens for JWT rotation.
type RefreshToken struct {
	ID        types.BinaryUUID `gorm:"type:uuid;primaryKey" json:"id"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
	DeletedAt gorm.DeletedAt   `gorm:"index" json:"-"`

	AuditFields

	UserID    uint             `json:"user_id" gorm:"not null;index"`
	TenantID  types.BinaryUUID `json:"tenant_id" gorm:"type:uuid;not null;index"`
	TokenHash string           `json:"-" gorm:"size:64;not null"`
	ExpiresAt time.Time  `json:"expires_at" gorm:"not null"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`

	User   User   `json:"-" gorm:"foreignKey:UserID"`
	Tenant Tenant `json:"-" gorm:"foreignKey:TenantID"`
}

func (r *RefreshToken) BeforeCreate(tx *gorm.DB) error {
	if r.ID.String() == (types.BinaryUUID{}).String() {
		id, err := uuid.NewRandom()
		r.ID = types.BinaryUUID(id)
		return err
	}
	return nil
}

func (*RefreshToken) TableName() string {
	return "refresh_tokens"
}
