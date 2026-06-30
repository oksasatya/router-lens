package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestPaginated(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := Paginated(c, http.StatusOK, []string{"a", "b"}, 2, 20, 41); err != nil {
		t.Fatalf("Paginated: %v", err)
	}
	var got struct {
		Data struct {
			Items      []string `json:"items"`
			Pagination struct {
				Page  int `json:"page"`
				Limit int `json:"limit"`
				Total int `json:"total"`
			} `json:"pagination"`
		} `json:"data"`
		Meta struct {
			Timestamp string `json:"timestamp"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got.Data.Items) != 2 || got.Data.Pagination.Total != 41 || got.Data.Pagination.Page != 2 {
		t.Fatalf("bad envelope: %+v", got)
	}
	if got.Meta.Timestamp == "" {
		t.Fatal("meta missing")
	}
}
