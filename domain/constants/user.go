package constants

type UserRole string

const (
	UserRoleAdmin         UserRole = "admin"
	UserRoleSystemManager UserRole = "system_manager"
)

// IsPlatformStaff returns true if the user may obtain platform (tenant-less) API tokens.
func IsPlatformStaff(role UserRole) bool {
	switch role {
	case UserRoleAdmin, UserRoleSystemManager:
		return true
	default:
		return false
	}
}
