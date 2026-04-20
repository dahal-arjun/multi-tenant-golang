package tenancy

import (
	"fmt"

	"gorm.io/gorm"
)

// WithTenant runs fn inside a transaction with PostgreSQL RLS GUC app.current_tenant_id set
// for the transaction (is_local=true). Use for queries against RLS-protected tables.
func WithTenant(db *gorm.DB, tenantID string, fn func(tx *gorm.DB) error) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`SELECT set_config('app.current_tenant_id', ?, true)`, tenantID).Error; err != nil {
			return fmt.Errorf("set tenant rls: %w", err)
		}
		return fn(tx)
	})
}
