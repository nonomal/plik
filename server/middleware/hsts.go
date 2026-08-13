package middleware

import "net/http"

// HSTS adds the Strict-Transport-Security header to all responses.
// max-age=31536000 (1 year) tells browsers to always use HTTPS for this domain.
func HSTS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000")
		next.ServeHTTP(w, r)
	})
}
