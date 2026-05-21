package response

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

type Meta struct {
	Page       int   `json:"page,omitempty"`
	PageSize   int   `json:"page_size,omitempty"`
	TotalItems int64 `json:"total_items,omitempty"`
	TotalPages int   `json:"total_pages,omitempty"`
}

type SuccessResponse struct {
	Success   bool        `json:"success"`
	Status    int         `json:"status"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data"`
	Meta      *Meta       `json:"meta,omitempty"`
	RequestID string      `json:"request_id"`
	Timestamp string      `json:"timestamp"`
}

type ErrorResponse struct {
	Success   bool   `json:"success"`
	Status    int    `json:"status"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
	RequestID string `json:"request_id"`
}

func getRequestID(c echo.Context) string {
	id, _ := c.Get("request_id").(string)
	return id
}

func Success(c echo.Context, httpStatus int, message string, data interface{}) error {
	return c.JSON(httpStatus, SuccessResponse{
		Success:   true,
		Status:    httpStatus,
		Message:   message,
		Data:      data,
		RequestID: getRequestID(c),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func SendError(c echo.Context, code, message string, httpStatus int) error {
	return c.JSON(httpStatus, ErrorResponse{
		Success:   false,
		Status:    httpStatus,
		Code:      code,
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RequestID: getRequestID(c),
	})
}

func Paginated(c echo.Context, message string, data interface{}, page, pageSize int, totalItems int64) error {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	totalPages := int((totalItems + int64(pageSize) - 1) / int64(pageSize))
	if totalPages < 1 {
		totalPages = 1
	}
	return c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Status:  200,
		Message: message,
		Data:    data,
		Meta: &Meta{
			Page:       page,
			PageSize:   pageSize,
			TotalItems: totalItems,
			TotalPages: totalPages,
		},
		RequestID: getRequestID(c),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}
