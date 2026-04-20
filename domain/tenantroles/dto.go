package tenantroles

// CreateTenantRoleRequest creates a named role in the current tenant.
type CreateTenantRoleRequest struct {
	Name        string  `json:"name" binding:"required,min=1,max=255"`
	Slug        *string `json:"slug,omitempty" binding:"omitempty,max=100"`
	Description string  `json:"description,omitempty" binding:"max=500"`
}

// UpdateTenantRoleRequest patches role metadata.
type UpdateTenantRoleRequest struct {
	Name        *string `json:"name,omitempty" binding:"omitempty,min=1,max=255"`
	Description *string `json:"description,omitempty" binding:"omitempty,max=500"`
}

// SetTenantRolePermissionsRequest replaces the permission set.
type SetTenantRolePermissionsRequest struct {
	Permissions []string `json:"permissions" binding:"required"`
}

// AssignMemberTenantRoleRequest binds a custom role to a membership.
type AssignMemberTenantRoleRequest struct {
	UserID       uint    `json:"user_id" binding:"required"`
	TenantRoleID *string `json:"tenant_role_id,omitempty" binding:"omitempty,uuid"`
}
