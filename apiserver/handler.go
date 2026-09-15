package apiserver

import (
	"database/sql"
	"encoding/json/v2"
	"errors"
	"fmt"
	"net/http"
	"time"
	"uuid"

	"github.com/ArditZubaku/async-api/reports"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
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

type SignInRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type SignInResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (r SignInRequest) Validate() error {
	if r.Email == "" {
		return errors.New("email is required")
	}
	if r.Password == "" {
		return errors.New("password is required")
	}
	return nil
}

func (s *ApiServer) signInHandler() http.HandlerFunc {
	return handler(func(w http.ResponseWriter, r *http.Request) error {
		req, err := decode[SignInRequest](r)
		if err != nil {
			return newErrWithStatus(http.StatusBadRequest, err)
		}

		user, err := s.store.Users.ByEmail(r.Context(), req.Email)
		if err != nil {
			return newErrWithStatus(http.StatusInternalServerError, err)
		}

		if err := user.ComparePassword(req.Password); err != nil {
			return newErrWithStatus(http.StatusUnauthorized, err)
		}

		tokenPair, err := s.jwtManager.GenerateTokenPair(user.Id)
		if err != nil {
			return newErrWithStatus(http.StatusInternalServerError, err)
		}

		if _, err = s.store.RefreshTokens.DeleteUserTokens(r.Context(), user.Id); err != nil {
			return newErrWithStatus(http.StatusInternalServerError, err)
		}

		if _, err := s.store.RefreshTokens.Create(r.Context(), user.Id, tokenPair.RefreshToken); err != nil {
			return newErrWithStatus(http.StatusInternalServerError, err)
		}

		if err := encode(ApiResponse[SignInResponse]{
			Data: &SignInResponse{
				AccessToken:  tokenPair.AccessToken.Raw,
				RefreshToken: tokenPair.RefreshToken.Raw,
			},
		},
			http.StatusOK,
			w,
		); err != nil {
			return newErrWithStatus(http.StatusInternalServerError, err)
		}

		return nil
	},
		s.logger,
	)
}

type TokenRefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (r TokenRefreshRequest) Validate() error {
	if r.RefreshToken == "" {
		return errors.New("refresh token is required")
	}
	return nil
}

type TokenRefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (s *ApiServer) tokenRefreshHandler() http.HandlerFunc {
	return handler(func(w http.ResponseWriter, r *http.Request) error {
		req, err := decode[TokenRefreshRequest](r)
		if err != nil {
			return newErrWithStatus(http.StatusBadRequest, err)
		}

		currentRefreshToken, err := s.jwtManager.Parse(req.RefreshToken)
		if err != nil {
			return newErrWithStatus(http.StatusUnauthorized, err)
		}

		userIdStr, err := currentRefreshToken.Claims.GetSubject()
		if err != nil {
			return newErrWithStatus(http.StatusUnauthorized, err)
		}

		userId, err := uuid.Parse(userIdStr)
		if err != nil {
			return newErrWithStatus(http.StatusUnauthorized, err)
		}

		currentRefreshTokenRecord, err := s.store.RefreshTokens.ByPrimaryKey(r.Context(), userId, currentRefreshToken)
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, sql.ErrNoRows) {
				status = http.StatusUnauthorized
			}
			return newErrWithStatus(status, err)
		}

		if currentRefreshTokenRecord.ExpiresAt.Before(time.Now()) {
			return newErrWithStatus(http.StatusUnauthorized, fmt.Errorf("refresh token expired"))
		}

		tokenPair, err := s.jwtManager.GenerateTokenPair(userId)
		if err != nil {
			return newErrWithStatus(http.StatusInternalServerError, err)
		}

		if _, err := s.store.RefreshTokens.DeleteUserTokens(r.Context(), userId); err != nil {
			return newErrWithStatus(http.StatusInternalServerError, err)
		}

		if _, err := s.store.RefreshTokens.Create(r.Context(), userId, tokenPair.RefreshToken); err != nil {
			return newErrWithStatus(http.StatusInternalServerError, err)
		}

		if err := encode(ApiResponse[TokenRefreshResponse]{
			Data: &TokenRefreshResponse{
				AccessToken:  tokenPair.AccessToken.Raw,
				RefreshToken: tokenPair.RefreshToken.Raw,
			},
		},
			http.StatusOK,
			w,
		); err != nil {
			return newErrWithStatus(http.StatusInternalServerError, err)
		}

		return nil
	},
		s.logger,
	)
}

type ApiReport struct {
	Id                   uuid.UUID  `json:"id"`
	ReportType           string     `json:"report_type,omitempty"`
	OutputFilePath       *string    `json:"output_file_path,omitempty"`
	DownloadUrl          *string    `json:"download_url,omitempty"`
	DownloadUrlExpiresAt *time.Time `json:"download_url_expires_at,omitempty"`
	ErrorMessage         *string    `json:"error_message,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	StartedAt            *time.Time `json:"started_at,omitempty"`
	CompletedAt          *time.Time `json:"completed_at,omitempty"`
	FailedAt             *time.Time `json:"failed_at,omitempty"`
	Status               string     `json:"status,omitempty"`
}

type CreateReportRequest struct {
	ReportType string `json:"report_type"`
}

func (r CreateReportRequest) Validate() error {
	if r.ReportType == "" {
		return errors.New("report_type is required")
	}

	return nil
}

func (s *ApiServer) createReportHandler() http.HandlerFunc {
	return handler(func(w http.ResponseWriter, r *http.Request) error {
		req, err := decode[CreateReportRequest](r)
		if err != nil {
			return newErrWithStatus(http.StatusBadRequest, err)
		}

		user, err := UserFromContext(r.Context())
		if err != nil {
			return newErrWithStatus(http.StatusUnauthorized, err)
		}

		report, err := s.store.Reports.Create(r.Context(), user.Id, req.ReportType)
		if err != nil {
			return newErrWithStatus(http.StatusInternalServerError, err)
		}

		sqsMsg := reports.SQSMessage{
			UserId:   report.UserId,
			ReportId: report.Id,
		}

		bytes, err := json.Marshal(&sqsMsg)
		if err != nil {
			return newErrWithStatus(http.StatusInternalServerError, err)
		}

		queueURLOutput, err := s.sqsClient.GetQueueUrl(r.Context(), &sqs.GetQueueUrlInput{
			QueueName: aws.String(s.config.SQSQueue),
		})
		if err != nil {
			return newErrWithStatus(http.StatusInternalServerError, err)
		}

		_, err = s.sqsClient.SendMessage(r.Context(), &sqs.SendMessageInput{
			MessageBody: aws.String(string(bytes)),
			QueueUrl:    queueURLOutput.QueueUrl,
		})
		if err != nil {
			return newErrWithStatus(http.StatusInternalServerError, err)
		}

		if err := encode(ApiResponse[ApiReport]{
			Data: &ApiReport{
				Id:                   report.Id,
				ReportType:           req.ReportType,
				OutputFilePath:       report.OutputFilePath,
				DownloadUrl:          report.DownloadUrl,
				DownloadUrlExpiresAt: report.DownloadUrlExpiresAt,
				ErrorMessage:         report.ErrorMessage,
				CreatedAt:            report.CreatedAt,
				StartedAt:            report.StartedAt,
				CompletedAt:          report.CompletedAt,
				FailedAt:             report.FailedAt,
				Status:               report.Status(),
			},
		}, http.StatusCreated, w); err != nil {
			return newErrWithStatus(http.StatusInternalServerError, err)
		}

		return nil
	},
		s.logger,
	)
}

func (s *ApiServer) getReportHandler() http.HandlerFunc {
	return handler(func(w http.ResponseWriter, r *http.Request) error {
		reportIdStr := r.PathValue("id")
		reportId, err := uuid.Parse(reportIdStr)
		if err != nil {
			return newErrWithStatus(http.StatusBadRequest, err)
		}

		user, err := UserFromContext(r.Context())
		if err != nil {
			return newErrWithStatus(http.StatusUnauthorized, err)
		}

		report, err := s.store.Reports.ByPrimaryKey(r.Context(), user.Id, reportId)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return newErrWithStatus(http.StatusNotFound, err)
			}
			return newErrWithStatus(http.StatusInternalServerError, err)
		}

		if report.CompletedAt != nil {
			needsRefresh := report.DownloadUrlExpiresAt != nil && report.DownloadUrlExpiresAt.Before(time.Now())
			if report.DownloadUrl == nil || needsRefresh {
				expiresAt := time.Now().Add(time.Second * 10)
				signedUrl, err := s.s3PresignClient.PresignGetObject(r.Context(), &s3.GetObjectInput{
					Bucket: aws.String(s.config.S3Bucket),
					Key:    report.OutputFilePath,
				}, func(options *s3.PresignOptions) {
					options.Expires = time.Second * 10
				})
				if err != nil {
					return newErrWithStatus(http.StatusInternalServerError, err)
				}

				report.DownloadUrl = &signedUrl.URL
				report.DownloadUrlExpiresAt = &expiresAt
				report, err = s.store.Reports.Update(r.Context(), report)
				if err != nil {
					return newErrWithStatus(http.StatusInternalServerError, err)
				}
			}
		}

		if err := encode(ApiResponse[ApiReport]{
			Data: &ApiReport{
				Id:                   report.Id,
				ReportType:           report.ReportType,
				OutputFilePath:       report.OutputFilePath,
				DownloadUrl:          report.DownloadUrl,
				DownloadUrlExpiresAt: report.DownloadUrlExpiresAt,
				ErrorMessage:         report.ErrorMessage,
				CreatedAt:            report.CreatedAt,
				StartedAt:            report.StartedAt,
				CompletedAt:          report.CompletedAt,
				FailedAt:             report.FailedAt,
				Status:               report.Status(),
			},
		}, http.StatusOK, w); err != nil {
			return newErrWithStatus(http.StatusInternalServerError, err)
		}

		return nil
	},
		s.logger,
	)
}
