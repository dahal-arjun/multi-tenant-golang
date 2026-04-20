package models

import (
	"clean-architecture/pkg/types"
	"time"

	_ "ariga.io/atlas-provider-gorm/gormschema"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Widget is an example tenant-scoped row protected by PostgreSQL RLS.
type Widget struct {
	ID        types.BinaryUUID `gorm:"type:uuid;primaryKey" json:"id"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
	DeletedAt gorm.DeletedAt   `gorm:"index" json:"-"`

	TenantBound
	AuditFields

	Title string `json:"title" gorm:"size:255;not null"`

	Tenant Tenant `json:"-" gorm:"foreignKey:TenantID"`
}

func (w *Widget) BeforeCreate(tx *gorm.DB) error {
	if w.ID.String() == (types.BinaryUUID{}).String() {
		id, err := uuid.NewRandom()
		w.ID = types.BinaryUUID(id)
		return err
	}
	return nil
}

func (*Widget) TableName() string {
	return "widgets"
}
