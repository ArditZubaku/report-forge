package apiserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

type ErrorWithStatus struct {
	status int
	err    error
}

func (e *ErrorWithStatus) Error() string {
	return e.err.Error()
}

func newErrWithStatus(status int, err error) *ErrorWithStatus {
	return &ErrorWithStatus{status: status, err: err}
}

func handler(
	f func(w http.ResponseWriter, r *http.Request) error,
	logger *slog.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := http.StatusInternalServerError
		msg := http.StatusText(status)
		if err := f(w, r); err != nil {
			if e, ok := errors.AsType[*ErrorWithStatus](err); ok {
				status = e.status
				msg = http.StatusText(e.status)
				if status == http.StatusBadRequest || status == http.StatusConflict {
					msg = e.err.Error()
				}
			}

			logger.Error(
				"error executing handler",
				"error", err,
				"status", status,
				"msg", msg,
			)

			w.WriteHeader(status)
			if err := json.NewEncoder(w).Encode(ApiResponse[struct{}]{
				Message: msg,
			}); err != nil {
				logger.Error(
					"error encoding response",
					"error", err,
				)
			}
		}
	}
}

func encode[T any](v T, status int, w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return fmt.Errorf("failed to encode response: %w", err)
	}

	return nil
}

type Validator interface {
	Validate() error
}

func decode[T Validator](r *http.Request) (t T, err error) {
	defer func() {
		if closeErr := r.Body.Close(); closeErr != nil {
			err = fmt.Errorf("failed to close body in signup handler - %w", closeErr)
		}
	}()

	if err = json.NewDecoder(r.Body).Decode(&t); err != nil {
		return t, fmt.Errorf("failed to decode request body: %w", err)
	}

	if err = t.Validate(); err != nil {
		return t, err
	}

	return t, nil
}
