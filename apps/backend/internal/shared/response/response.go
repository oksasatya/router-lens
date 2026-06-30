// Package response writes the canonical JSON envelope (with meta + localized
// error messages) for all handlers.
package response

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	apperrors "router-lens/internal/shared/errors"
	"router-lens/internal/shared/i18n"

	"github.com/labstack/echo/v4"
)

type Meta struct {
	Lang      string `json:"lang"`
	RequestID string `json:"request_id,omitempty"`
	Timestamp string `json:"timestamp"`
}

type envelope struct {
	Data  any        `json:"data,omitempty"`
	Error *errorBody `json:"error,omitempty"`
	Meta  Meta       `json:"meta"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

const codeHTTPError = "http_error"

// LangOf returns the language resolved by the Lang middleware, or the default.
func LangOf(c echo.Context) i18n.Lang {
	if l, ok := c.Get(i18n.ContextKey).(i18n.Lang); ok && l != "" {
		return l
	}
	return i18n.Default
}

func meta(c echo.Context) Meta {
	return Meta{
		Lang:      string(LangOf(c)),
		RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

func Data(c echo.Context, status int, data any) error {
	return c.JSON(status, envelope{Data: data, Meta: meta(c)})
}

func Created(c echo.Context, data any) error {
	return c.JSON(http.StatusCreated, envelope{Data: data, Meta: meta(c)})
}

func NoContent(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

type pageMeta struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total"`
}

type pageData struct {
	Items      any      `json:"items"`
	Pagination pageMeta `json:"pagination"`
}

// Paginated writes the canonical list envelope:
// { "data": { "items": [...], "pagination": {page, limit, total} }, "meta": {...} }.
// items must be a slice; nil renders as an empty array on the client via omitempty-free encoding.
func Paginated(c echo.Context, status int, items any, page, limit, total int) error {
	return c.JSON(status, envelope{
		Data: pageData{Items: items, Pagination: pageMeta{Page: page, Limit: limit, Total: total}},
		Meta: meta(c),
	})
}

func Error(c echo.Context, err error) error {
	lang := LangOf(c)
	if ae, ok := apperrors.As(err); ok {
		return c.JSON(apperrors.HTTPStatus(ae.Kind), envelope{
			Error: &errorBody{Code: ae.Code, Message: i18n.Message(ae.Code, lang, ae.Message), Details: ae.Details},
			Meta:  meta(c),
		})
	}
	if he, ok := errors.AsType[*echo.HTTPError](err); ok {
		msg := fmt.Sprintf("%v", he.Message)
		if msg == "" {
			msg = http.StatusText(he.Code)
		}
		return c.JSON(he.Code, envelope{
			Error: &errorBody{Code: codeHTTPError, Message: msg},
			Meta:  meta(c),
		})
	}
	return c.JSON(http.StatusInternalServerError, envelope{
		Error: &errorBody{Code: i18n.CodeInternal, Message: i18n.Message(i18n.CodeInternal, lang, "internal server error")},
		Meta:  meta(c),
	})
}
