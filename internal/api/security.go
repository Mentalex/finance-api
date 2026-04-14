package api

import "net/http"

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// force HTTPS in production
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// disable caching for API responses
		w.Header().Set("Cache-Control", "no-store")

		// basic XSS protection
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		next.ServeHTTP(w, r)
	})
}
