package api

import (
	"net/http"
	"time"
)

type Attempt struct {
	FailedAttempts int
	BlockedUntil   time.Time
}

func (s *Server) checkPinCode() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.PathValue("code")
		ip := r.RemoteAddr

		s.mu.Lock()
		att := s.attempts[ip]
		s.mu.Unlock()

		if att != nil && time.Now().Before(att.BlockedUntil) {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

		if code != s.config.Control.PinCode {
			s.mu.Lock()
			if att == nil {
				att = &Attempt{}
				s.attempts[ip] = att
			}

			att.FailedAttempts++

			httpStatus := http.StatusUnauthorized
			if att.FailedAttempts >= 3 {
				att.BlockedUntil = time.Now().Add(5 * time.Minute)
				httpStatus = http.StatusTooManyRequests
			}
			s.mu.Unlock()

			w.WriteHeader(httpStatus)
			return
		}

		s.sessionManager.Put(r.Context(), "authenticated", true)

		s.mu.Lock()
		delete(s.attempts, ip)
		s.mu.Unlock()

		w.WriteHeader(http.StatusOK)
	}
}
