package models

import (
	"clean-architecture/domain/constants"
	"clean-architecture/pkg/types"
	"time"

	_ "ariga.io/atlas-provider-gorm/gormschema"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User model
type User struct {
	gorm.Model
	NullableTenant
	AuditFields

	UUID         types.BinaryUUID `json:"uuid" gorm:"type:uuid;not null;uniqueIndex"`
	PasswordHash string           `json:"-" gorm:"size:255;not null"`

	FirstName   string `json:"first_name" gorm:"size:255"`
	LastName    string `json:"last_name" gorm:"size:255"`
	FirstNameJa string `json:"first_name_ja" gorm:"size:255"`
	LastNameJa  string `json:"last_name_ja" gorm:"size:255"`

	Email string             `json:"email" gorm:"notnull;uniqueIndex;size:255"`
	Role  constants.UserRole `json:"role" gorm:"size:25" copier:"-"`

	IsActive        bool `json:"is_active" gorm:"default:false"`
	IsEmailVerified bool `json:"is_email_verified" gorm:"default:false"`

	// EmailVerificationDeadline is when an unverified user may still log in to see ErrEmailNotVerified;
	// after this instant, login returns ErrVerificationExpired until resend-verification extends it.
	EmailVerificationDeadline *time.Time `json:"-" gorm:"column:email_verification_deadline"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.UUID.String() == (types.BinaryUUID{}).String() {
		id, err := uuid.NewRandom()
		u.UUID = types.BinaryUUID(id)
		return err
	}
	return nil
}

func (*User) TableName() string {
	return "users"
}
