package models

import (
	"time"

	_ "ariga.io/atlas-provider-gorm/gormschema"

	"clean-architecture/pkg/types"
)

// TenantRolePermission maps a tenant role to a curated permission key.
type TenantRolePermission struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	TenantRoleID   types.BinaryUUID `json:"tenant_role_id" gorm:"type:uuid;not null;index"`
	PermissionKey  string           `json:"permission_key" gorm:"size:100;not null"`

	TenantRole TenantRole `json:"-" gorm:"foreignKey:TenantRoleID"`
}

func (*TenantRolePermission) TableName() string {
	return "tenant_role_permissions"
}
