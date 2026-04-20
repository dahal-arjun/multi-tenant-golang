package widget

import (
	"clean-architecture/pkg/errorz"
	"clean-architecture/pkg/framework"
	"clean-architecture/pkg/responses"
	"clean-architecture/pkg/utils"

	"github.com/gin-gonic/gin"
)

// Controller exposes widget HTTP handlers.
type Controller struct {
	service *Service
	logger  framework.Logger
}

// NewController constructs the widget controller.
func NewController(service *Service, logger framework.Logger) *Controller {
	return &Controller{service: service, logger: logger}
}

// List godoc
// @Summary List widgets for current tenant
// @Description Rows are filtered by PostgreSQL RLS using the tenant from the JWT.
// @Tags widgets
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page (1-based)" default(1)
// @Param limit query int false "Page size (max 100)" default(20)
// @Success 200 {object} map[string]interface{} "data: array of models.Widget; pagination"
// @Router /api/widgets [get]
func (w *Controller) List(c *gin.Context) {
	tid, ok := c.Get(framework.TenantID)
	tidStr, okStr := tid.(string)
	if !ok || !okStr || tidStr == "" {
		responses.HandleError(w.logger, c, errorz.ErrUnauthorizedAccess)
		return
	}
	p := utils.BuildPagination(c)
	items, total, err := w.service.List(tidStr, p)
	if err != nil {
		responses.HandleError(w.logger, c, err)
		return
	}
	responses.JSONPaginated(c, 200, items, total, p)
}
