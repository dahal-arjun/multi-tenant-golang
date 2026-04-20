package models

import (
	"clean-architecture/domain/constants"
	"clean-architecture/pkg/types"
	"time"

	_ "ariga.io/atlas-provider-gorm/gormschema"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TenantInvitation is a pending invite to join a tenant by email.
type TenantInvitation struct {
	ID        types.BinaryUUID `gorm:"type:uuid;primaryKey" json:"id"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`

	TenantID         types.BinaryUUID     `json:"tenant_id" gorm:"type:uuid;not null;index"`
	Email            string               `json:"email" gorm:"size:255;not null"`
	Role             constants.TenantRole `json:"role" gorm:"size:50;not null"`
	TenantRoleID     *types.BinaryUUID    `json:"tenant_role_id,omitempty" gorm:"type:uuid;index"`
	TokenHash        string               `json:"-" gorm:"size:64;not null"`
	InvitedByUserID  uint                 `json:"invited_by_user_id" gorm:"not null"`
	ExpiresAt        time.Time            `json:"expires_at" gorm:"not null"`
	AcceptedAt       *time.Time           `json:"accepted_at,omitempty"`
	RevokedAt        *time.Time           `json:"revoked_at,omitempty"`

	Tenant     Tenant     `json:"-" gorm:"foreignKey:TenantID"`
	TenantRole *TenantRole `json:"-" gorm:"foreignKey:TenantRoleID"`
	InvitedBy  User       `json:"-" gorm:"foreignKey:InvitedByUserID"`
}

func (inv *TenantInvitation) BeforeCreate(tx *gorm.DB) error {
	if inv.ID.String() == (types.BinaryUUID{}).String() {
		id, err := uuid.NewRandom()
		inv.ID = types.BinaryUUID(id)
		return err
	}
	return nil
}

func (*TenantInvitation) TableName() string {
	return "tenant_invitations"
}
