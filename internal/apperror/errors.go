package apperror

import (
	"fmt"
	"time"
)

type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) HTTPStatus() int {
	return e.Status
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func BadRequest(msg string) *AppError {
	return &AppError{Code: "BAD_REQUEST", Message: msg, Status: 400}
}

func Unauthorized(msg string) *AppError {
	return &AppError{Code: "UNAUTHORIZED", Message: msg, Status: 401}
}

func Forbidden(msg string) *AppError {
	return &AppError{Code: "FORBIDDEN", Message: msg, Status: 403}
}

func NotFound(msg string) *AppError {
	return &AppError{Code: "NOT_FOUND", Message: msg, Status: 404}
}

func Conflict(msg string) *AppError {
	return &AppError{Code: "CONFLICT", Message: msg, Status: 409}
}

func ValidationError(msg string) *AppError {
	return &AppError{Code: "VALIDATION_ERROR", Message: msg, Status: 422}
}

func TooManyRequests(msg string) *AppError {
	return &AppError{Code: "TOO_MANY_REQUESTS", Message: msg, Status: 429}
}

func Internal(msg string, err error) *AppError {
	return &AppError{Code: "INTERNAL_ERROR", Message: msg, Status: 500, Err: err}
}

type Meta struct {
	RequestID string `json:"request_id"`
	Timestamp string `json:"timestamp"`
}

func NewMeta(requestID string) Meta {
	return Meta{
		RequestID: requestID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

type ErrorResponse struct {
	Status  string `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Meta    Meta   `json:"meta"`
}

func ToErrorResponse(err *AppError) ErrorResponse {
	return ErrorResponse{
		Status:  "error",
		Code:    err.Code,
		Message: err.Message,
	}
}
