package utils

import (
	"errors"
	"net/http"
)

type AppError struct {
	Status  int
	Code    string
	Message string
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

func BadRequest(message string) *AppError {
	return &AppError{Status: http.StatusBadRequest, Code: "bad_request", Message: message}
}

func Forbidden(message string) *AppError {
	return &AppError{Status: http.StatusForbidden, Code: "forbidden", Message: message}
}

func NotFound(message string) *AppError {
	return &AppError{Status: http.StatusNotFound, Code: "not_found", Message: message}
}

func Conflict(message string) *AppError {
	return &AppError{Status: http.StatusConflict, Code: "conflict", Message: message}
}

func Internal(message string, err error) *AppError {
	return &AppError{Status: http.StatusInternalServerError, Code: "internal_error", Message: message, Err: err}
}

func AsAppError(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return Internal("unexpected error", err)
}
