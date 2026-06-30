// Package validator wraps go-playground/validator with EN + ID translators and
// returns a localized validation AppError.
package validator

import (
	"errors"
	"reflect"
	"strings"

	enlocale "github.com/go-playground/locales/en"
	idlocale "github.com/go-playground/locales/id"
	ut "github.com/go-playground/universal-translator"
	govalidator "github.com/go-playground/validator/v10"
	entranslations "github.com/go-playground/validator/v10/translations/en"
	idtranslations "github.com/go-playground/validator/v10/translations/id"

	apperrors "router-lens/internal/shared/errors"
	"router-lens/internal/shared/i18n"
)

// ponytail: codeValidation duplicated in i18n.CodeValidation — reusing would
// create a cross-package dependency not warranted yet; keep local until a
// shared const makes sense.
const codeValidation = "validation_failed"

// Validator wraps go-playground Validate + UniversalTranslator for EN and ID.
type Validator struct {
	validate *govalidator.Validate
	uni      *ut.UniversalTranslator
}

// New builds a Validator with EN (default) and ID translators registered.
// Construct once at wiring time (cmd/server/main.go) and inject.
func New() (*Validator, error) {
	enLoc := enlocale.New()
	uni := ut.New(enLoc, enLoc, idlocale.New())
	validate := govalidator.New()
	validate.RegisterTagNameFunc(jsonFieldName)

	enT, _ := uni.GetTranslator(string(i18n.EN))
	if err := entranslations.RegisterDefaultTranslations(validate, enT); err != nil {
		return nil, err
	}
	idT, _ := uni.GetTranslator(string(i18n.ID))
	if err := idtranslations.RegisterDefaultTranslations(validate, idT); err != nil {
		return nil, err
	}
	return &Validator{validate: validate, uni: uni}, nil
}

// Struct validates s by its `validate` tags. Returns nil on success, or a
// *apperrors.AppError(KindValidation) with details = map[json-field]message.
func (v *Validator) Struct(s any, lang i18n.Lang) error {
	err := v.validate.Struct(s)
	if err == nil {
		return nil
	}
	var verrs govalidator.ValidationErrors
	ok := errors.As(err, &verrs)
	if !ok {
		return err
	}
	trans, _ := v.uni.GetTranslator(string(lang))
	details := make(map[string]string, len(verrs))
	for _, fe := range verrs {
		details[fe.Field()] = fe.Translate(trans)
	}
	return apperrors.New(apperrors.KindValidation, codeValidation, "validation failed").WithDetails(details)
}

// jsonFieldName resolves the JSON tag name for a struct field so validation
// errors report the field as the client sees it, not the Go identifier.
func jsonFieldName(fld reflect.StructField) string {
	name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
	if name == "-" {
		return ""
	}
	if name == "" {
		return fld.Name
	}
	return name
}
