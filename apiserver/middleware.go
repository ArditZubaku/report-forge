package apiserver

import (
	"log/slog"
	"net/http"
)

func newLoggingMiddleware(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Info("http request", "path", r.Method+" "+r.URL.Path)
			next.ServeHTTP(w, r)
		})
	}
}
