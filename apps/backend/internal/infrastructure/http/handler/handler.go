package handler

import (
	"github.com/labstack/echo/v4"

	"router-lens/internal/shared/response"
	"router-lens/internal/shared/validator"
)

// bindAndValidate binds the JSON body and validates it in the request language.
// Shared by every handler in this package.
func bindAndValidate(c echo.Context, v *validator.Validator, dst any) error {
	if err := c.Bind(dst); err != nil {
		return err
	}
	return v.Struct(dst, response.LangOf(c))
}
