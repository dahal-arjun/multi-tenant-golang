package types

import (
	"clean-architecture/pkg/errorz"
	"database/sql/driver"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// BinaryUUID -> binary uuid wrapper over uuid.UUID
type BinaryUUID uuid.UUID

// ParseUUID -> parses string uuid to binary uuid
func ParseUUID(id string) BinaryUUID {
	return BinaryUUID(uuid.MustParse(id))
}

// ShouldParseUUID -> parses string uuid to binary uuid with error
func ShouldParseUUID(id string) (BinaryUUID, error) {
	UUID, err := uuid.Parse(id)
	if err != nil {
		return BinaryUUID{}, errorz.ErrInvalidUUID
	}
	return BinaryUUID(UUID), err
}

func (b BinaryUUID) String() string {
	return uuid.UUID(b).String()
}

// MarshalJSON -> convert to json string
func (b BinaryUUID) MarshalJSON() ([]byte, error) {
	s := uuid.UUID(b)
	str := "\"" + s.String() + "\""
	return []byte(str), nil
}

// UnmarshalJSON -> convert from json string
func (b *BinaryUUID) UnmarshalJSON(by []byte) error {
	s, err := uuid.ParseBytes(by)
	*b = BinaryUUID(s)
	return err
}

// GormDataType default sql data type for gorm
func (BinaryUUID) GormDataType() string {
	return "uuid"
}

// GormDBDataType dialect-specific column type
func (BinaryUUID) GormDBDataType(db *gorm.DB, _ *schema.Field) string {
	switch db.Dialector.Name() {
	case "postgres":
		return "uuid"
	default:
		return "binary(16)"
	}
}

// Scan -> scan value into BinaryUUID
func (b *BinaryUUID) Scan(value any) error {
	if value == nil {
		*b = BinaryUUID{}
		return nil
	}
	switch v := value.(type) {
	case []byte:
		if len(v) == 16 {
			data, err := uuid.FromBytes(v)
			*b = BinaryUUID(data)
			return err
		}
		parsed, err := uuid.ParseBytes(v)
		*b = BinaryUUID(parsed)
		return err
	case string:
		parsed, err := uuid.Parse(v)
		*b = BinaryUUID(parsed)
		return err
	default:
		return fmt.Errorf("unsupported BinaryUUID scan type %T", value)
	}
}

// Value -> return BinaryUUID for driver
func (b BinaryUUID) Value() (driver.Value, error) {
	if uuid.UUID(b) == uuid.Nil {
		return nil, errors.New("BinaryUUID is nil")
	}
	return uuid.UUID(b).String(), nil
}
