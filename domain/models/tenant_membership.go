package models

import (
	"clean-architecture/domain/constants"
	"clean-architecture/pkg/types"

	_ "ariga.io/atlas-provider-gorm/gormschema"

	"gorm.io/gorm"
)

// TenantMembership links a user to a tenant with a role.
type TenantMembership struct {
	gorm.Model

	AuditFields

	UserID   uint             `json:"user_id" gorm:"not null;index"`
	TenantID types.BinaryUUID `json:"tenant_id" gorm:"type:uuid;not null;index"`
	Role     constants.TenantRole `json:"role" gorm:"size:50;not null"`

	// TenantRoleID optionally links a custom tenant role; effective permissions are builtin(role) ∪ grants(role).
	TenantRoleID *types.BinaryUUID `json:"tenant_role_id,omitempty" gorm:"type:uuid;index"`

	User       User        `json:"-" gorm:"foreignKey:UserID"`
	Tenant     Tenant      `json:"-" gorm:"foreignKey:TenantID"`
	TenantRole *TenantRole `json:"tenant_role,omitempty" gorm:"foreignKey:TenantRoleID"`
}

func (*TenantMembership) TableName() string {
	return "tenant_memberships"
}
