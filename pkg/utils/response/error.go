package response

import "fmt"

type APIError struct {
	Code    string
	Message string
	Status  int
	Err     error
}

func (e *APIError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *APIError) HTTPStatus() int {
	return e.Status
}

func (e *APIError) Unwrap() error {
	return e.Err
}

func BadRequest(msg string) *APIError {
	return &APIError{Code: "BAD_REQUEST", Message: msg, Status: 400}
}

func Unauthorized(msg string) *APIError {
	return &APIError{Code: "UNAUTHORIZED", Message: msg, Status: 401}
}

func Forbidden(msg string) *APIError {
	return &APIError{Code: "FORBIDDEN", Message: msg, Status: 403}
}

func NotFound(msg string) *APIError {
	return &APIError{Code: "NOT_FOUND", Message: msg, Status: 404}
}

func Conflict(msg string) *APIError {
	return &APIError{Code: "CONFLICT", Message: msg, Status: 409}
}

func ValidationError(msg string) *APIError {
	return &APIError{Code: "VALIDATION_ERROR", Message: msg, Status: 422}
}

func TooManyRequests(msg string) *APIError {
	return &APIError{Code: "TOO_MANY_REQUESTS", Message: msg, Status: 429}
}

func Internal(msg string, err error) *APIError {
	return &APIError{Code: "INTERNAL_ERROR", Message: msg, Status: 500, Err: err}
}
