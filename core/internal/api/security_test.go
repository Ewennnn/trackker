package api

import (
	"djtracker/internal/config"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexedwards/scs/v2"
)

func newTestServer() *Server {
	return &Server{
		config: &config.Config{
			Control: config.ControlConfig{
				PinCode: "123456",
			},
		},
		sessionManager: scs.New(),
		attempts:       make(map[string]*Attempt),
	}
}

func setupTest() (*Server, http.Handler) {
	server := newTestServer()
	return server, server.sessionManager.LoadAndSave(server.checkPinCode())
}

func TestCheckPinCode(t *testing.T) {
	ip := "127.0.0.1:1234"

	t.Run("valid ping code", func(t *testing.T) {
		_, handler := setupTest()

		req := httptest.NewRequest(http.MethodPost, "/api/pincode/123456", nil)
		req.SetPathValue("code", "123456")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("invalid pin code", func(t *testing.T) {
		_, handler := setupTest()

		req := httptest.NewRequest(http.MethodPost, "/api/pincode/000000", nil)
		req.SetPathValue("code", "000000")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("block after 3 failed attempts", func(t *testing.T) {
		_, handler := setupTest()

		for i := 0; i < 3; i++ {
			req := httptest.NewRequest(http.MethodPost, "/api/pincode/000000", nil)
			req.RemoteAddr = ip
			req.SetPathValue("code", "000000")

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("attempt %d expected 401 got %d", i, rec.Code)
			}
		}

		req := httptest.NewRequest(http.MethodPost, "/api/pincode/000000", nil)
		req.RemoteAddr = ip
		req.SetPathValue("code", "000000")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusTooManyRequests {
			t.Errorf("expected 429, got %d", rec.Code)
		}
	})

	t.Run("reset after successful login", func(t *testing.T) {
		server, handler := setupTest()

		for i := 0; i < 2; i++ {
			req := httptest.NewRequest(http.MethodPost, "/api/pincode/000000", nil)
			req.RemoteAddr = ip
			req.SetPathValue("code", "000000")

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("attempt %d expected 401 got %d", i, rec.Code)
			}
		}

		req := httptest.NewRequest(http.MethodPost, "/api/pincode/123456", nil)
		req.RemoteAddr = ip
		req.SetPathValue("code", "123456")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 got %d", rec.Code)
		}

		server.mu.Lock()
		_, exists := server.attempts[ip]
		server.mu.Unlock()

		if exists {
			t.Errorf("expected attempts to be reset")
		}
	})
}
