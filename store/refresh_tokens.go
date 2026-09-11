package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"fmt"
	"time"
	"uuid"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type RefreshTokenStore struct {
	db *sqlx.DB
}

type RefreshToken struct {
	UserID      uuid.UUID `db:"user_id"`
	HashedToken string    `db:"hashed_token"`
	CreatedAt   time.Time `db:"created_at"`
	ExpiresAt   time.Time `db:"expires_at"`
}

func NewRefreshTokenStore(db *sql.DB) *RefreshTokenStore {
	return &RefreshTokenStore{
		db: sqlx.NewDb(db, "postgres"),
	}
}

func (s *RefreshTokenStore) Create(ctx context.Context, userId uuid.UUID, token *jwt.Token) (*RefreshToken, error) {
	base64TokenHash, err := s.getBase64HashFromToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to get base64 hash from token: %w", err)
	}

	expiresAt, err := token.Claims.GetExpirationTime()
	if err != nil {
		return nil, fmt.Errorf("failed to get expiration time: %w", err)
	}

	const insertQuery = `INSERT INTO refresh_tokens (user_id, hashed_token, expires_at) VALUES ($1, $2, $3) RETURNING *`

	var refreshToken RefreshToken
	if err := s.db.GetContext(ctx, &refreshToken, insertQuery, userId, base64TokenHash, expiresAt.Time); err != nil {
		return nil, fmt.Errorf("failed to create refresh token record: %w", err)
	}

	return &refreshToken, nil
}

func (s *RefreshTokenStore) ByPrimaryKey(ctx context.Context, userId uuid.UUID, token *jwt.Token) (*RefreshToken, error) {
	base64TokenHash, err := s.getBase64HashFromToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to get base64 hash from token: %w", err)
	}

	const selectQuery = `SELECT * FROM refresh_tokens WHERE user_id = $1 AND hashed_token = $2`

	var refreshToken RefreshToken
	if err := s.db.GetContext(ctx, &refreshToken, selectQuery, userId, base64TokenHash); err != nil {
		return nil, fmt.Errorf("failed to get refresh token %s record for user %s: %w", base64TokenHash, userId, err)
	}

	return &refreshToken, nil
}

func (s *RefreshTokenStore) DeleteUserTokens(ctx context.Context, userId uuid.UUID) (sql.Result, error) {
	const deleteQuery = `DELETE FROM refresh_tokens WHERE user_id = $1;`

	res, err := s.db.ExecContext(ctx, deleteQuery, userId)
	if err != nil {
		return nil, fmt.Errorf("failed to delete refresh token record for user %s: %w", userId, err)
	}

	return res, nil
}

func (s *RefreshTokenStore) getBase64HashFromToken(token *jwt.Token) (string, error) {
	h := sha256.New()
	_, err := h.Write([]byte(token.Raw))
	if err != nil {
		return "", fmt.Errorf("failed to hash token: %w", err)
	}
	hashedToken := h.Sum(nil)
	base64TokenHash := base64.StdEncoding.EncodeToString(hashedToken)
	return base64TokenHash, nil
}
