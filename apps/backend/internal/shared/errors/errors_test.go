package errors

import (
	stderrors "errors"
	"net/http"
	"testing"
)

func TestAppError(t *testing.T) {
	t.Run("status mapping", func(t *testing.T) {
		cases := map[Kind]int{
			KindValidation:   http.StatusBadRequest,
			KindUnauthorized: http.StatusUnauthorized,
			KindForbidden:    http.StatusForbidden,
			KindNotFound:     http.StatusNotFound,
			KindConflict:     http.StatusConflict,
			KindInternal:     http.StatusInternalServerError,
		}
		for k, want := range cases {
			if got := HTTPStatus(k); got != want {
				t.Errorf("kind %s: got %d want %d", k, got, want)
			}
		}
	})

	t.Run("wrap and unwrap", func(t *testing.T) {
		base := stderrors.New("boom")
		e := Wrap(KindInternal, "internal_error", "failed", base)
		if !stderrors.Is(e, base) {
			t.Fatal("expected Is to find wrapped error")
		}
		if got, ok := As(e); !ok || got.Code != "internal_error" {
			t.Fatalf("As failed: %+v %v", got, ok)
		}
	})
}
