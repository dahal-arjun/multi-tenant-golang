package platform

import (
	"clean-architecture/pkg/framework"
	"clean-architecture/pkg/responses"

	"github.com/gin-gonic/gin"
)

// Controller exposes platform (system) HTTP handlers.
type Controller struct {
	service *Service
	logger  framework.Logger
}

// NewController constructs the controller.
func NewController(service *Service, logger framework.Logger) *Controller {
	return &Controller{service: service, logger: logger}
}

// Health godoc
// @Summary Platform API health (authenticated)
// @Tags platform
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]string
// @Router /api/platform/health [get]
func (c *Controller) Health(ctx *gin.Context) {
	responses.JSON(ctx, 200, gin.H{"status": "ok", "scope": "platform"})
}

// TenantSummary godoc
// @Summary Cross-tenant counts for operators
// @Description Requires a platform access token (from login for system_manager/admin). Does not use tenant RLS.
// @Tags platform
// @Security BearerAuth
// @Produce json
// @Success 200 {object} platform.TenantSummary
// @Router /api/platform/tenants/summary [get]
func (c *Controller) TenantSummary(ctx *gin.Context) {
	out, err := c.service.GetTenantSummary()
	if err != nil {
		responses.HandleError(c.logger, ctx, err)
		return
	}
	responses.JSON(ctx, 200, out)
}
