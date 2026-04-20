package models

import (
	"clean-architecture/pkg/types"
	"time"

	_ "ariga.io/atlas-provider-gorm/gormschema"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TenantRole is a tenant-defined role with a set of permission keys.
type TenantRole struct {
	ID        types.BinaryUUID `gorm:"type:uuid;primaryKey" json:"id"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
	DeletedAt gorm.DeletedAt   `gorm:"index" json:"-"`

	AuditFields

	TenantID    types.BinaryUUID `json:"tenant_id" gorm:"type:uuid;not null;index"`
	Name        string           `json:"name" gorm:"size:255;not null"`
	Slug        string           `json:"slug" gorm:"size:100;not null"`
	Description string           `json:"description,omitempty" gorm:"size:500"`

	Tenant Tenant `json:"-" gorm:"foreignKey:TenantID"`
}

func (tr *TenantRole) BeforeCreate(tx *gorm.DB) error {
	if tr.ID.String() == (types.BinaryUUID{}).String() {
		id, err := uuid.NewRandom()
		tr.ID = types.BinaryUUID(id)
		return err
	}
	return nil
}

func (*TenantRole) TableName() string {
	return "tenant_roles"
}
