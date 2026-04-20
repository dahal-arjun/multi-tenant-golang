package models

import (
	"clean-architecture/pkg/types"
	"time"

	_ "ariga.io/atlas-provider-gorm/gormschema"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PasswordResetToken stores a hashed one-time token for password recovery.
type PasswordResetToken struct {
	ID        types.BinaryUUID `gorm:"type:uuid;primaryKey" json:"id"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`

	UserID    uint       `json:"user_id" gorm:"not null;index"`
	TokenHash string     `json:"-" gorm:"size:64;not null"`
	ExpiresAt time.Time  `json:"expires_at" gorm:"not null"`
	UsedAt    *time.Time `json:"used_at,omitempty"`

	User User `json:"-" gorm:"foreignKey:UserID"`
}

func (p *PasswordResetToken) BeforeCreate(tx *gorm.DB) error {
	if p.ID.String() == (types.BinaryUUID{}).String() {
		id, err := uuid.NewRandom()
		p.ID = types.BinaryUUID(id)
		return err
	}
	return nil
}

func (*PasswordResetToken) TableName() string {
	return "password_reset_tokens"
}
