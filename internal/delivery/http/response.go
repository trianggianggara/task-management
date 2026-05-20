package http

import (
	"math"

	"github.com/labstack/echo/v4"
	"task-management/internal/apperror"
)

type PaginationMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

type PaginatedResponse struct {
	Data       interface{}    `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}

func Success(c echo.Context, status int, data interface{}) error {
	return c.JSON(status, data)
}

func Error(c echo.Context, err error) error {
	var appErr *apperror.AppError
	switch e := err.(type) {
	case *apperror.AppError:
		appErr = e
	default:
		appErr = apperror.Internal("internal server error", err)
	}
	resp := apperror.ToErrorResponse(appErr)
	return c.JSON(appErr.Status, resp)
}

func Paginated(c echo.Context, data interface{}, page, limit int, total int64) error {
	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if totalPages < 1 {
		totalPages = 1
	}
	return c.JSON(200, PaginatedResponse{
		Data: data,
		Pagination: PaginationMeta{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}
