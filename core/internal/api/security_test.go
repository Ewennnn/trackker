package api

import (
	"djtracker/internal/config"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckPinCode_OK(t *testing.T) {
	server := &Server{
		config: &config.Config{
			Control: config.ControlConfig{
				PinCode: "123456",
			},
		},
	}

	handler := server.checkPinCode()

	t.Run("valid pin code", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/pincode/123456", nil)
		req.SetPathValue("code", "123456")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
	})

	t.Run("invalid pin code", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/pincode/000000", nil)
		req.SetPathValue("code", "000000")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", rec.Code)
		}
	})
}
