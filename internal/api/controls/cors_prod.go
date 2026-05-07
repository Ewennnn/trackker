//go:build !dev

package controls

import "net/http"

// In production, the controls web page is loaded from the same origin as the API, so CORS is not needed. This middleware is a no-op.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}
