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
	config *config.Config
	logger *slog.Logger
	store  *store.Store
}

func New(
	config *config.Config,
	logger *slog.Logger,
	store *store.Store,
) *ApiServer {
	return &ApiServer{
		config: config,
		logger: logger,
		store:  store,
	}
}

func (s *ApiServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ping", s.ping)
	mux.HandleFunc("POST /auth/signup", s.signUpHandler())

	middleware := newLoggingMiddleware(s.logger)

	server := &http.Server{
		Addr:    net.JoinHostPort(s.config.ApiServerHost, s.config.ApiServerPort),
		Handler: middleware(mux),
	}

	go func() {
		s.logger.Info("Listening on ", "port", s.config.ApiServerPort)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("apiserver failed to listen and serve", "error", err)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("apiserver failed to shutdown", "error", err)
		}
	}()

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
