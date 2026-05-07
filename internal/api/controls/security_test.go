package controls

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexedwards/scs/v2"
)

func newTestServer() *Server {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	return &Server{
		pinCode:        "123456",
		log:            logger,
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

//func TestControlActions(t *testing.T) {
//	server := newTestServer()
//	mux := http.NewServeMux()
//	mux.Handle("POST /api/pincode/{code}", server.checkPinCode())
//	mux.Handle("POST /api/control/actions/{action}", server.authMiddleware(server.handleControlAction()))
//	handler := server.sessionManager.LoadAndSave(mux)
//
//	loginReq := httptest.NewRequest(http.MethodPost, "/api/pincode/123456", nil)
//	loginReq.SetPathValue("code", "123456")
//	loginRec := httptest.NewRecorder()
//	handler.ServeHTTP(loginRec, loginReq)
//
//	if loginRec.Code != http.StatusOK {
//		t.Fatalf("expected login 200 got %d", loginRec.Code)
//	}
//
//	buildActionRequest := func(action string) *http.Request {
//		req := httptest.NewRequest(http.MethodPost, "/api/control/actions/"+action, nil)
//		req.SetPathValue("action", action)
//		for _, cookie := range loginRec.Result().Cookies() {
//			req.AddCookie(cookie)
//		}
//		return req
//	}

//t.Run("blackout toggles on then off", func(t *testing.T) {
//	rec1 := httptest.NewRecorder()
//	handler.ServeHTTP(rec1, buildActionRequest("blackout"))
//	if rec1.Code != http.StatusOK {
//		t.Fatalf("expected 200 got %d", rec1.Code)
//	}
//
//	var payload1 struct {
//		Mode DisplayMode `json:"mode"`
//	}
//	if err := json.Unmarshal(rec1.Body.Bytes(), &payload1); err != nil {
//		t.Fatalf("invalid payload: %v", err)
//	}
//	if payload1.Mode != DisplayModeBlackout {
//		t.Fatalf("expected mode blackout got %s", payload1.Mode)
//	}
//
//	rec2 := httptest.NewRecorder()
//	handler.ServeHTTP(rec2, buildActionRequest("blackout"))
//	if rec2.Code != http.StatusOK {
//		t.Fatalf("expected 200 got %d", rec2.Code)
//	}
//
//	var payload2 struct {
//		Mode DisplayMode `json:"mode"`
//	}
//	if err := json.Unmarshal(rec2.Body.Bytes(), &payload2); err != nil {
//		t.Fatalf("invalid payload: %v", err)
//	}
//	if payload2.Mode != DisplayModeLive {
//		t.Fatalf("expected mode live got %s", payload2.Mode)
//	}
//})
//
//t.Run("freeze action sets freeze mode", func(t *testing.T) {
//	rec := httptest.NewRecorder()
//	handler.ServeHTTP(rec, buildActionRequest("freeze_tracking"))
//	if rec.Code != http.StatusOK {
//		t.Fatalf("expected 200 got %d", rec.Code)
//	}
//
//	var payload struct {
//		Mode DisplayMode `json:"mode"`
//	}
//	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
//		t.Fatalf("invalid payload: %v", err)
//	}
//	if payload.Mode != DisplayModeFreezeTracking {
//		t.Fatalf("expected mode freeze_tracking got %s", payload.Mode)
//	}
//})
//
//t.Run("unauthenticated action request is rejected", func(t *testing.T) {
//	req := httptest.NewRequest(http.MethodPost, "/api/control/actions/blackout", nil)
//	req.SetPathValue("action", "blackout")
//	rec := httptest.NewRecorder()
//	handler.ServeHTTP(rec, req)
//
//	if rec.Code != http.StatusUnauthorized {
//		t.Fatalf("expected 401 got %d", rec.Code)
//	}
//})
//}
