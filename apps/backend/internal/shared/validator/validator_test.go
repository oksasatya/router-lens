package validator

import (
	"testing"

	apperrors "router-lens/internal/shared/errors"
	"router-lens/internal/shared/i18n"
)

type sample struct {
	Email string `json:"email" validate:"required,email"`
	Name  string `json:"name" validate:"required,max=5"`
}

func TestValidator(t *testing.T) {
	v, err := New()
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	t.Run("valid passes", func(t *testing.T) {
		if err := v.Struct(sample{Email: "a@b.com", Name: "ok"}, i18n.EN); err != nil {
			t.Fatalf("expected valid, got %v", err)
		}
	})

	t.Run("invalid produces localized validation AppError", func(t *testing.T) {
		err := v.Struct(sample{Email: "bad", Name: "toolong"}, i18n.ID)
		ae, ok := apperrors.As(err)
		if !ok || ae.Kind != apperrors.KindValidation {
			t.Fatalf("expected validation AppError, got %v", err)
		}
		details, ok := ae.Details.(map[string]string)
		if !ok || details["email"] == "" || details["name"] == "" {
			t.Fatalf("expected localized field details, got %+v", ae.Details)
		}
	})
}
