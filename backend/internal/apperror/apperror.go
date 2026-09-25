package apperror

import (
	"errors"
	"net/http"
)

type AppError struct {
	Code    string
	Message string
	Status  int
	Details any
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}

	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func (e *AppError) WithCause(err error) *AppError {
	e.Err = err
	return e
}

func BadRequest(code, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Status:  http.StatusBadRequest,
	}
}

func Validation(details any) *AppError {
	return &AppError{
		Code:    "VALIDATION_ERROR",
		Message: "The request contains invalid data",
		Status:  http.StatusBadRequest,
		Details: details,
	}
}

func Unauthorized(message string) *AppError {
	return &AppError{
		Code:    "UNAUTHORIZED",
		Message: message,
		Status:  http.StatusUnauthorized,
	}
}

func NotFound(code, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Status:  http.StatusNotFound,
	}
}

func Conflict(code, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Status:  http.StatusConflict,
	}
}

func MethodNotAllowed() *AppError {
	return &AppError{
		Code:    "METHOD_NOT_ALLOWED",
		Message: "This method is not allowed",
		Status:  http.StatusMethodNotAllowed,
	}
}

func Internal(err error) *AppError {
	return &AppError{
		Code:    "INTERNAL_ERROR",
		Message: "Internal server error",
		Status:  http.StatusInternalServerError,
		Err:     err,
	}
}

func From(err error) *AppError {
	var appErr *AppError

	if errors.As(err, &appErr) {
		return appErr
	}

	return Internal(err)
}
