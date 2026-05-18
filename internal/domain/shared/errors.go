package shared

import "fmt"

type AppError struct {
	Status  int
	Code    string
	Message string
	Details any
	Wrap    error
}

func (e *AppError) Error() string {
	if e.Wrap != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Wrap)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error { return e.Wrap }

func NewAppError(status int, code, message string) *AppError {
	return &AppError{Status: status, Code: code, Message: message}
}

func NewAppErrorWithDetails(status int, code, message string, details any) *AppError {
	return &AppError{Status: status, Code: code, Message: message, Details: details}
}

func WrapAppError(err error, status int, code, message string) *AppError {
	return &AppError{Status: status, Code: code, Message: message, Wrap: err}
}

func NotFound(message string) *AppError {
	return NewAppError(404, "not_found", message)
}

func BadRequest(message string) *AppError {
	return NewAppError(400, "bad_request", message)
}

func BadRequestDetails(message string, details any) *AppError {
	return NewAppErrorWithDetails(400, "bad_request", message, details)
}

func Conflict(message string, details any) *AppError {
	return NewAppErrorWithDetails(409, "conflict", message, details)
}

func Unprocessable(message string) *AppError {
	return NewAppError(422, "unprocessable_entity", message)
}

func Internal(message string) *AppError {
	return NewAppError(500, "internal_error", message)
}

func Unauthorized(message string) *AppError {
	return NewAppError(401, "unauthorized", message)
}

func Forbidden(message string) *AppError {
	return NewAppError(403, "forbidden", message)
}
