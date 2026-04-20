package models

import (
	"clean-architecture/pkg/types"
	"time"

	_ "ariga.io/atlas-provider-gorm/gormschema"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// EmailVerificationToken stores a hashed one-time token for email verification.
type EmailVerificationToken struct {
	ID        types.BinaryUUID `gorm:"type:uuid;primaryKey" json:"id"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`

	UserID    uint       `json:"user_id" gorm:"not null;index"`
	TokenHash string     `json:"-" gorm:"size:64;not null"`
	ExpiresAt time.Time  `json:"expires_at" gorm:"not null"`
	UsedAt    *time.Time `json:"used_at,omitempty"`

	User User `json:"-" gorm:"foreignKey:UserID"`
}

func (e *EmailVerificationToken) BeforeCreate(tx *gorm.DB) error {
	if e.ID.String() == (types.BinaryUUID{}).String() {
		id, err := uuid.NewRandom()
		e.ID = types.BinaryUUID(id)
		return err
	}
	return nil
}

func (*EmailVerificationToken) TableName() string {
	return "email_verification_tokens"
}
