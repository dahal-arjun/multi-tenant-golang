package models

import (
	"clean-architecture/pkg/types"

	_ "ariga.io/atlas-provider-gorm/gormschema"
)

// AuditFields stores actor user IDs for create / update / soft-delete attribution (FK → users.id).
// Embed in any table that should track who changed a row.
type AuditFields struct {
	CreatedByID *uint `json:"created_by_id,omitempty" gorm:"index"`
	UpdatedByID *uint `json:"updated_by_id,omitempty" gorm:"index"`
	DeletedByID *uint `json:"deleted_by_id,omitempty" gorm:"index"`
}

// TenantBound is for rows that always belong to exactly one tenant (RLS + tenant_id filters).
type TenantBound struct {
	TenantID types.BinaryUUID `json:"tenant_id" gorm:"type:uuid;not null;index"`
}

// NullableTenant is an optional primary/home tenant (e.g. users after registration).
type NullableTenant struct {
	TenantID *types.BinaryUUID `json:"tenant_id,omitempty" gorm:"type:uuid;index"`
}
