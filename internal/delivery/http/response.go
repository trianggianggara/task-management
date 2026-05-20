package http

import (
	"math"
	"time"

	"github.com/labstack/echo/v4"
	"task-management/internal/apperror"
)

var requestIDKey = "request_id"

type SuccessResponse struct {
	Status  string         `json:"status"`
	Message string         `json:"message"`
	Data    interface{}    `json:"data"`
	Meta    apperror.Meta  `json:"meta"`
}

type PaginationMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

type PaginatedResponse struct {
	Status     string         `json:"status"`
	Message    string         `json:"message"`
	Data       interface{}    `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
	Meta       apperror.Meta  `json:"meta"`
}

func getMeta(c echo.Context) apperror.Meta {
	requestID, _ := c.Get(requestIDKey).(string)
	return apperror.Meta{
		RequestID: requestID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

func Success(c echo.Context, httpStatus int, message string, data interface{}) error {
	return c.JSON(httpStatus, SuccessResponse{
		Status:  "success",
		Message: message,
		Data:    data,
		Meta:    getMeta(c),
	})
}

func Paginated(c echo.Context, message string, data interface{}, page, limit int, total int64) error {
	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if totalPages < 1 {
		totalPages = 1
	}
	return c.JSON(200, PaginatedResponse{
		Status:  "success",
		Message: message,
		Data:    data,
		Pagination: PaginationMeta{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
		Meta: getMeta(c),
	})
}
