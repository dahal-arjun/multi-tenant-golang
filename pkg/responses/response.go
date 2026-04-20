package responses

import (
	"clean-architecture/pkg/framework"
	"clean-architecture/pkg/utils"

	"github.com/gin-gonic/gin"
)

// JSON : json response function
func JSON(c *gin.Context, statusCode int, data any) {
	c.JSON(statusCode, gin.H{"data": data})
}

// ErrorJSON : json error response function
func ErrorJSON(c *gin.Context, statusCode int, data any) {
	c.JSON(statusCode, gin.H{"error": data})
}

// SuccessJSON : json error response function
func SuccessJSON(c *gin.Context, statusCode int, data any) {
	c.JSON(statusCode, gin.H{"msg": data})
}

// PaginationMeta is returned alongside list payloads.
type PaginationMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
}

// JSONPaginated responds with `data` and structured pagination metadata.
func JSONPaginated(c *gin.Context, statusCode int, data any, total int64, p utils.Pagination) {
	limit := p.Limit
	if limit <= 0 {
		limit = utils.DefaultPageSize
	}
	page := p.Page
	if page <= 0 {
		page = utils.DefaultPage
	}
	var totalPages int
	if limit > 0 && total > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}
	c.JSON(statusCode, gin.H{
		"data": data,
		"pagination": PaginationMeta{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
			HasNext:    int64(page*limit) < total,
		},
	})
}

// JSONWithPagination : json response function
func JSONWithPagination(c *gin.Context, statusCode int, response map[string]any) {
	limitAny, _ := c.Get(framework.Limit)
	pageAny, _ := c.Get(framework.Page)
	limit, _ := limitAny.(int)
	page, _ := pageAny.(int)
	if limit <= 0 {
		limit = utils.DefaultPageSize
	}
	if page <= 0 {
		page = utils.DefaultPage
	}
	count := response["count"].(int64)

	c.JSON(
		statusCode,
		gin.H{
			"data": response["data"],
			"pagination": gin.H{
				"page":     page,
				"limit":    limit,
				"count":    count,
				"has_next": int64(page*limit) < count,
			},
		})
}
