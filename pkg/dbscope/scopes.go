package dbscope

import (
	"clean-architecture/pkg/types"

	"gorm.io/gorm"
)

// TenantID restricts the query to rows for the given tenant (column tenant_id).
// For tables without tenant_id, do not use this scope.
func TenantID(tenantUUID string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if tenantUUID == "" {
			return db
		}
		return db.Where("tenant_id = ?", tenantUUID)
	}
}

// TenantIDBinaryUUID is the same as TenantID but accepts a typed UUID.
func TenantIDBinaryUUID(tid types.BinaryUUID) func(db *gorm.DB) *gorm.DB {
	return TenantID(tid.String())
}

// ActiveRows adds an explicit deleted_at IS NULL predicate.
// GORM already excludes soft-deleted rows when models use gorm.DeletedAt; use this after Unscoped()
// or for raw SQL alignment with the same convention.
func ActiveRows(db *gorm.DB) *gorm.DB {
	return db.Where("deleted_at IS NULL")
}

// TenantAndActive combines tenant and explicit non-deleted filter (useful after Unscoped).
func TenantAndActive(tenantUUID string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		db = TenantID(tenantUUID)(db)
		return ActiveRows(db)
	}
}
