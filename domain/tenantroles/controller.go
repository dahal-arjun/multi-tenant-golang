package tenantroles

import (
	"clean-architecture/pkg/errorz"
	"clean-architecture/pkg/framework"
	"clean-architecture/pkg/responses"
	"clean-architecture/pkg/types"
	"clean-architecture/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Controller exposes tenant role HTTP handlers.
type Controller struct {
	service *Service
	logger  framework.Logger
}

// NewController constructs the controller.
func NewController(service *Service, logger framework.Logger) *Controller {
	return &Controller{service: service, logger: logger}
}

func (c *Controller) tenantIDStr(ctx *gin.Context) (string, bool) {
	t, ok := ctx.Get(framework.TenantID)
	s, ok2 := t.(string)
	if !ok || !ok2 || s == "" {
		return "", false
	}
	return s, true
}

// List godoc
// @Summary List tenant-defined roles
// @Tags tenant-roles
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page (1-based)" default(1)
// @Param limit query int false "Page size (max 100)" default(20)
// @Success 200 {object} map[string]interface{} "data: array of models.TenantRole; pagination"
// @Router /api/tenant-roles [get]
func (c *Controller) List(ctx *gin.Context) {
	tidStr, ok := c.tenantIDStr(ctx)
	if !ok {
		responses.HandleError(c.logger, ctx, errorz.ErrUnauthorizedAccess)
		return
	}
	tid, err := types.ShouldParseUUID(tidStr)
	if err != nil {
		responses.HandleError(c.logger, ctx, err)
		return
	}
	p := utils.BuildPagination(ctx)
	items, total, err := c.service.ListRoles(tid, p.Offset, p.Limit)
	if err != nil {
		responses.HandleError(c.logger, ctx, err)
		return
	}
	responses.JSONPaginated(ctx, 200, items, total, p)
}

// Create godoc
// @Summary Create a tenant role
// @Tags tenant-roles
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body CreateTenantRoleRequest true "Role"
// @Success 201 {object} models.TenantRole
// @Router /api/tenant-roles [post]
func (c *Controller) Create(ctx *gin.Context) {
	if !CanManageTenantRoles(ctx) {
		responses.HandleError(c.logger, ctx, errorz.ErrForbiddenAccess)
		return
	}
	tidStr, ok := c.tenantIDStr(ctx)
	if !ok {
		responses.HandleError(c.logger, ctx, errorz.ErrUnauthorizedAccess)
		return
	}
	tid, err := types.ShouldParseUUID(tidStr)
	if err != nil {
		responses.HandleError(c.logger, ctx, err)
		return
	}
	dbID, _ := ctx.Get(framework.UserDBID)
	actorID, ok := dbID.(uint)
	if !ok || actorID == 0 {
		responses.HandleError(c.logger, ctx, errorz.ErrUnauthorizedAccess)
		return
	}
	var req CreateTenantRoleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		responses.HandleValidationError(c.logger, ctx, err)
		return
	}
	row, err := c.service.CreateRole(actorID, tid, req.Name, req.Description, req.Slug)
	if err != nil {
		responses.HandleError(c.logger, ctx, err)
		return
	}
	responses.JSON(ctx, 201, row)
}

// Update godoc
// @Summary Update a tenant role
// @Tags tenant-roles
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Role UUID"
// @Param body body UpdateTenantRoleRequest true "Fields"
// @Success 200 {object} models.TenantRole
// @Router /api/tenant-roles/{id} [patch]
func (c *Controller) Update(ctx *gin.Context) {
	if !CanManageTenantRoles(ctx) {
		responses.HandleError(c.logger, ctx, errorz.ErrForbiddenAccess)
		return
	}
	tidStr, ok := c.tenantIDStr(ctx)
	if !ok {
		responses.HandleError(c.logger, ctx, errorz.ErrUnauthorizedAccess)
		return
	}
	tid, err := types.ShouldParseUUID(tidStr)
	if err != nil {
		responses.HandleError(c.logger, ctx, err)
		return
	}
	rid, err := types.ShouldParseUUID(ctx.Param("id"))
	if err != nil {
		responses.HandleError(c.logger, ctx, err)
		return
	}
	dbID, _ := ctx.Get(framework.UserDBID)
	actorID, ok := dbID.(uint)
	if !ok || actorID == 0 {
		responses.HandleError(c.logger, ctx, errorz.ErrUnauthorizedAccess)
		return
	}
	var req UpdateTenantRoleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		responses.HandleValidationError(c.logger, ctx, err)
		return
	}
	row, err := c.service.UpdateRole(actorID, tid, rid, req.Name, req.Description)
	if err != nil {
		responses.HandleError(c.logger, ctx, err)
		return
	}
	responses.JSON(ctx, 200, row)
}

// Delete godoc
// @Summary Delete a tenant role
// @Tags tenant-roles
// @Security BearerAuth
// @Param id path string true "Role UUID"
// @Success 204 "No Content"
// @Router /api/tenant-roles/{id} [delete]
func (c *Controller) Delete(ctx *gin.Context) {
	if !CanManageTenantRoles(ctx) {
		responses.HandleError(c.logger, ctx, errorz.ErrForbiddenAccess)
		return
	}
	tidStr, ok := c.tenantIDStr(ctx)
	if !ok {
		responses.HandleError(c.logger, ctx, errorz.ErrUnauthorizedAccess)
		return
	}
	tid, err := types.ShouldParseUUID(tidStr)
	if err != nil {
		responses.HandleError(c.logger, ctx, err)
		return
	}
	rid, err := types.ShouldParseUUID(ctx.Param("id"))
	if err != nil {
		responses.HandleError(c.logger, ctx, err)
		return
	}
	dbID, _ := ctx.Get(framework.UserDBID)
	actorID, ok := dbID.(uint)
	if !ok || actorID == 0 {
		responses.HandleError(c.logger, ctx, errorz.ErrUnauthorizedAccess)
		return
	}
	if err := c.service.DeleteRole(actorID, tid, rid); err != nil {
		responses.HandleError(c.logger, ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

// SetPermissions godoc
// @Summary Replace permissions for a tenant role
// @Tags tenant-roles
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Role UUID"
// @Param body body SetTenantRolePermissionsRequest true "Permission keys"
// @Success 204 "No Content"
// @Router /api/tenant-roles/{id}/permissions [put]
func (c *Controller) SetPermissions(ctx *gin.Context) {
	if !CanManageTenantRoles(ctx) {
		responses.HandleError(c.logger, ctx, errorz.ErrForbiddenAccess)
		return
	}
	tidStr, ok := c.tenantIDStr(ctx)
	if !ok {
		responses.HandleError(c.logger, ctx, errorz.ErrUnauthorizedAccess)
		return
	}
	tid, err := types.ShouldParseUUID(tidStr)
	if err != nil {
		responses.HandleError(c.logger, ctx, err)
		return
	}
	rid, err := types.ShouldParseUUID(ctx.Param("id"))
	if err != nil {
		responses.HandleError(c.logger, ctx, err)
		return
	}
	dbID, _ := ctx.Get(framework.UserDBID)
	actorID, ok := dbID.(uint)
	if !ok || actorID == 0 {
		responses.HandleError(c.logger, ctx, errorz.ErrUnauthorizedAccess)
		return
	}
	var req SetTenantRolePermissionsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		responses.HandleValidationError(c.logger, ctx, err)
		return
	}
	if err := c.service.SetRolePermissions(actorID, tid, rid, req.Permissions); err != nil {
		responses.HandleError(c.logger, ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

// AssignMemberRole godoc
// @Summary Assign a custom tenant role to a member
// @Tags tenant-roles
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body AssignMemberTenantRoleRequest true "Target user and role"
// @Success 204 "No Content"
// @Router /api/tenant-members/role [patch]
func (c *Controller) AssignMemberRole(ctx *gin.Context) {
	if !CanManageTenantRoles(ctx) {
		responses.HandleError(c.logger, ctx, errorz.ErrForbiddenAccess)
		return
	}
	tidStr, ok := c.tenantIDStr(ctx)
	if !ok {
		responses.HandleError(c.logger, ctx, errorz.ErrUnauthorizedAccess)
		return
	}
	tid, err := types.ShouldParseUUID(tidStr)
	if err != nil {
		responses.HandleError(c.logger, ctx, err)
		return
	}
	dbID, _ := ctx.Get(framework.UserDBID)
	actorID, ok := dbID.(uint)
	if !ok || actorID == 0 {
		responses.HandleError(c.logger, ctx, errorz.ErrUnauthorizedAccess)
		return
	}
	var req AssignMemberTenantRoleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		responses.HandleValidationError(c.logger, ctx, err)
		return
	}
	var roleID *types.BinaryUUID
	if req.TenantRoleID != nil && *req.TenantRoleID != "" {
		rid, err := types.ShouldParseUUID(*req.TenantRoleID)
		if err != nil {
			responses.HandleError(c.logger, ctx, err)
			return
		}
		roleID = &rid
	}
	if err := c.service.AssignMembershipTenantRole(actorID, tid, req.UserID, roleID); err != nil {
		responses.HandleError(c.logger, ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}
