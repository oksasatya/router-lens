// Package errors defines the application-level error type and its HTTP mapping.
package errors

import (
	stderrors "errors"
	"net/http"
)

type Kind string

const (
	KindValidation   Kind = "validation"
	KindUnauthorized Kind = "unauthorized"
	KindForbidden    Kind = "forbidden"
	KindNotFound     Kind = "not_found"
	KindConflict     Kind = "conflict"
	KindInternal     Kind = "internal"
)

type AppError struct {
	Kind    Kind
	Code    string
	Message string
	Details any
	wrapped error
}

func (e *AppError) Error() string { return e.Message }
func (e *AppError) Unwrap() error { return e.wrapped }

func (e *AppError) WithDetails(d any) *AppError {
	e.Details = d
	return e
}

func New(kind Kind, code, message string) *AppError {
	return &AppError{Kind: kind, Code: code, Message: message}
}

func Wrap(kind Kind, code, message string, err error) *AppError {
	return &AppError{Kind: kind, Code: code, Message: message, wrapped: err}
}

func As(err error) (*AppError, bool) {
	if ae, ok := stderrors.AsType[*AppError](err); ok {
		return ae, true
	}
	return nil, false
}

func HTTPStatus(kind Kind) int {
	switch kind {
	case KindValidation:
		return http.StatusBadRequest
	case KindUnauthorized:
		return http.StatusUnauthorized
	case KindForbidden:
		return http.StatusForbidden
	case KindNotFound:
		return http.StatusNotFound
	case KindConflict:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
