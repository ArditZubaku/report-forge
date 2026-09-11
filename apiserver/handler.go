package apiserver

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
)

type SignUpRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ApiResponse[T any] struct {
	Data    *T     `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
}

func (r SignUpRequest) Validate() error {
	if r.Email == "" {
		return errors.New("email is required")
	}

	if r.Password == "" {
		return errors.New("password is required")
	}

	return nil
}

func (s *ApiServer) signUpHandler() http.HandlerFunc {
	return handler(
		func(w http.ResponseWriter, r *http.Request) error {
			req, err := decode[SignUpRequest](r)
			if err != nil {
				return newErrWithStatus(http.StatusBadRequest, err)
			}

			existingUser, err := s.store.Users.ByEmail(r.Context(), req.Email)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return newErrWithStatus(http.StatusInternalServerError, err)
			}

			if existingUser != nil {
				return newErrWithStatus(http.StatusConflict, fmt.Errorf("email already registered"))
			}

			_, err = s.store.Users.CreateUser(r.Context(), req.Email, req.Password)
			if err != nil {
				return newErrWithStatus(http.StatusInternalServerError, err)
			}

			w.WriteHeader(http.StatusCreated)
			if err := encode(
				ApiResponse[any]{Message: "successfully signed up user"},
				http.StatusCreated,
				w,
			); err != nil {
				return newErrWithStatus(http.StatusInternalServerError, err)
			}

			return nil
		},
		s.logger,
	)
}
