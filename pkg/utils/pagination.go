package utils

import (
	"clean-architecture/pkg/framework"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Pagination struct {
	Page   int
	Limit  int
	Offset int
}

func BuildPagination(ctx *gin.Context) Pagination {
	// setting pagination
	pageStr := ctx.Query("page")
	limitStr := ctx.Query("limit")

	var (
		err   error
		page  int
		limit int
	)

	page, err = strconv.Atoi(pageStr)
	if err != nil || page == 0 {
		page = 1
	}

	limit, err = strconv.Atoi(limitStr)
	if err != nil || limit == 0 {
		limit = 10
	}

	ctx.Set(framework.Page, page)
	ctx.Set(framework.Limit, limit)

	return Pagination{
		Page:   page,
		Limit:  limit,
		Offset: (page - 1) * limit,
	}
}
