package apiserver

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/ArditZubaku/async-api/config"
	"github.com/ArditZubaku/async-api/store"
)

type ApiServer struct {
	config     *config.Config
	logger     *slog.Logger
	store      *store.Store
	jwtManager *JwtManager
}

func New(
	config *config.Config,
	logger *slog.Logger,
	store *store.Store,
	jwtManager *JwtManager,
) *ApiServer {
	return &ApiServer{
		config:     config,
		logger:     logger,
		store:      store,
		jwtManager: jwtManager,
	}
}

func (s *ApiServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ping", s.ping)
	mux.HandleFunc("POST /auth/signup", s.signUpHandler())
	mux.HandleFunc("POST /auth/signin", s.signInHandler())
	mux.HandleFunc("POST /auth/refresh", s.tokenRefreshHandler())

	logging := newLoggingMiddleware(s.logger)
	auth := newAuthMiddleware(s.jwtManager, s.store.Users, s.logger)

	server := &http.Server{
		Addr:    net.JoinHostPort(s.config.APIServerHost, s.config.APIServerPort),
		Handler: logging(auth(mux)),
	}

	go func() {
		s.logger.Info("Listening on ", "port", s.config.APIServerPort)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("apiserver failed to listen and serve", "error", err)
		}
	}()

	var wg sync.WaitGroup
	wg.Go(func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("apiserver failed to shutdown", "error", err)
		}
	})

	wg.Wait()

	return server.ListenAndServe()
}

func (s *ApiServer) ping(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("pong"))
	if err != nil {
		log.Println("Failed to write in ping")
	}
}
