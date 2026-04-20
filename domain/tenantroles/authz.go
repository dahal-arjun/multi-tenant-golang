package tenantroles

import (
	"clean-architecture/domain/constants"
	"clean-architecture/pkg/framework"

	"github.com/gin-gonic/gin"
)

// CanManageTenantRoles is true for built-in owner/admin or users with roles.manage.
func CanManageTenantRoles(c *gin.Context) bool {
	r, _ := c.Get(framework.Role)
	role, _ := r.(string)
	if role == string(constants.TenantRoleOwner) || role == string(constants.TenantRoleAdmin) {
		return true
	}
	raw, _ := c.Get(framework.Permissions)
	perms, _ := raw.([]string)
	for _, p := range perms {
		if p == string(constants.PermissionRolesManage) {
			return true
		}
	}
	return false
}
