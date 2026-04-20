package utils

import (
	"clean-architecture/pkg/framework"
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	DefaultPage     = 1
	DefaultPageSize = 20
	MaxPageSize     = 100
)

type Pagination struct {
	Page   int
	Limit  int
	Offset int
}

// DefaultPagination is the first page using DefaultPage and DefaultPageSize (matches BuildPagination when query params are absent).
func DefaultPagination() Pagination {
	page := DefaultPage
	limit := DefaultPageSize
	return Pagination{
		Page:   page,
		Limit:  limit,
		Offset: (page - 1) * limit,
	}
}

// BuildPagination parses `page` and `limit` query params (1-based page), applies defaults, caps limit, stores values on the Gin context, and returns offset for SQL.
func BuildPagination(ctx *gin.Context) Pagination {
	page := DefaultPage
	limit := DefaultPageSize

	if v := ctx.Query("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			page = p
		}
	}
	if v := ctx.Query("limit"); v != "" {
		if l, err := strconv.Atoi(v); err == nil && l > 0 {
			limit = l
		}
	}
	if limit > MaxPageSize {
		limit = MaxPageSize
	}

	ctx.Set(framework.Page, page)
	ctx.Set(framework.Limit, limit)

	return Pagination{
		Page:   page,
		Limit:  limit,
		Offset: (page - 1) * limit,
	}
}
