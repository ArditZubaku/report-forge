package apiserver

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"uuid"

	"github.com/ArditZubaku/async-api/store"
)

type userCtxKey struct{}

func ContextWithUser(ctx context.Context, user *store.User) context.Context {
	return context.WithValue(ctx, userCtxKey{}, user)
}

func UserFromContext(ctx context.Context) (*store.User, error) {
	user, ok := ctx.Value(userCtxKey{}).(*store.User)
	if !ok || user == nil {
		return nil, errors.New("user not found in context")
	}

	return user, nil
}

func newLoggingMiddleware(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Info("http request", "path", r.Method+" "+r.URL.Path)
			next.ServeHTTP(w, r)
		})
	}
}

func newAuthMiddleware(jwtManager *JwtManager, userStore *store.UserStore, logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/auth") {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				logger.Error("auth header is empty")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			var token string
			if parts := strings.SplitN(authHeader, " ", 2); len(parts) == 2 {
				token = parts[1]
			}

			if token == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			parsedToken, err := jwtManager.Parse(token)
			if err != nil {
				logger.Error("failed to parse token", "error", err)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			if !jwtManager.IsAccessToken(parsedToken) {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			userIdStr, err := parsedToken.Claims.GetSubject()
			if err != nil {
				logger.Error("failed to extract user id from token subject", "error", err)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			userId, err := uuid.Parse(userIdStr)
			if err != nil {
				logger.Error("user id is not valid uuid", "error", err)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			user, err := userStore.ById(r.Context(), userId)
			if err != nil {
				logger.Error("failed to get user by id", "error", err)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r.WithContext(ContextWithUser(r.Context(), user)))
		})
	}
}
