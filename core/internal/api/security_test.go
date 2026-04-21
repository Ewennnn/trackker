package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"trackker/internal/config"

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

func setupControlTest() (*Server, http.Handler) {
	server := newTestServer()
	mux := http.NewServeMux()
	mux.Handle("POST /api/pincode/{code}", server.checkPinCode())
	mux.Handle("GET /api/control/session", server.getSessionStatus())
	return server, server.sessionManager.LoadAndSave(mux)
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

			if i < 2 && rec.Code != http.StatusUnauthorized {
				t.Fatalf("attempt %d expected 401 got %d", i, rec.Code)
			} else if i == 2 && rec.Code != http.StatusTooManyRequests {
				t.Fatalf("attempt %d expected 429 got %d", i, rec.Code)
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

func TestSessionStatus(t *testing.T) {
	type sessionPayload struct {
		Authenticated bool `json:"authenticated"`
	}

	t.Run("returns false when not authenticated", func(t *testing.T) {
		_, handler := setupControlTest()

		req := httptest.NewRequest(http.MethodGet, "/api/control/session", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 got %d", rec.Code)
		}

		var payload sessionPayload
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("invalid JSON payload: %v", err)
		}

		if payload.Authenticated {
			t.Fatalf("expected authenticated=false")
		}
	})

	t.Run("returns true after valid PIN with same session cookie", func(t *testing.T) {
		_, handler := setupControlTest()

		loginReq := httptest.NewRequest(http.MethodPost, "/api/pincode/123456", nil)
		loginReq.SetPathValue("code", "123456")
		loginRec := httptest.NewRecorder()
		handler.ServeHTTP(loginRec, loginReq)

		if loginRec.Code != http.StatusOK {
			t.Fatalf("expected login 200 got %d", loginRec.Code)
		}

		sessionReq := httptest.NewRequest(http.MethodGet, "/api/control/session", nil)
		for _, cookie := range loginRec.Result().Cookies() {
			sessionReq.AddCookie(cookie)
		}

		sessionRec := httptest.NewRecorder()
		handler.ServeHTTP(sessionRec, sessionReq)

		if sessionRec.Code != http.StatusOK {
			t.Fatalf("expected session status 200 got %d", sessionRec.Code)
		}

		var payload sessionPayload
		if err := json.Unmarshal(sessionRec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("invalid JSON payload: %v", err)
		}

		if !payload.Authenticated {
			t.Fatalf("expected authenticated=true")
		}
	})
}
