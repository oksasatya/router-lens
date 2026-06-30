package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	apperrors "router-lens/internal/shared/errors"
	"router-lens/internal/shared/i18n"
)

func TestError_LocalizesWithMeta(t *testing.T) {
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), rec)
	c.Set(i18n.ContextKey, i18n.ID)

	_ = Error(c, apperrors.New(apperrors.KindValidation, "validation_failed", "validation failed"))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d want 400", rec.Code)
	}
	var body struct {
		Error struct{ Code, Message string }   `json:"error"`
		Meta  struct{ Lang, Timestamp string } `json:"meta"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Error.Message != "Validasi gagal" {
		t.Fatalf("expected ID message, got %q", body.Error.Message)
	}
	if body.Meta.Lang != "id" || body.Meta.Timestamp == "" {
		t.Fatalf("meta wrong: %+v", body.Meta)
	}
}

func TestError_EchoHTTPError_PreservesStatus(t *testing.T) {
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), rec)

	_ = Error(c, echo.NewHTTPError(http.StatusNotFound))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: got %d want 404", rec.Code)
	}
	var body struct {
		Error struct{ Code, Message string }   `json:"error"`
		Meta  struct{ Lang, Timestamp string } `json:"meta"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Error.Message == "" {
		t.Fatal("expected non-empty error message")
	}
	if body.Meta.Timestamp == "" {
		t.Fatal("expected non-empty meta.timestamp")
	}
}
