package models

import (
	"clean-architecture/pkg/types"
	"time"

	_ "ariga.io/atlas-provider-gorm/gormschema"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Tenant is an organization / account in the multi-tenant system.
type Tenant struct {
	ID        types.BinaryUUID `gorm:"type:uuid;primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	AuditFields

	Name string `json:"name" gorm:"size:255;not null"`
	Slug string `json:"slug" gorm:"size:255;not null;uniqueIndex"`
}

func (t *Tenant) BeforeCreate(tx *gorm.DB) error {
	if t.ID.String() == (types.BinaryUUID{}).String() {
		id, err := uuid.NewRandom()
		t.ID = types.BinaryUUID(id)
		return err
	}
	return nil
}

func (*Tenant) TableName() string {
	return "tenants"
}
