package apiserver

import (
	"fmt"
	"time"
	"uuid"

	"github.com/ArditZubaku/async-api/config"
	"github.com/golang-jwt/jwt/v5"
)

var signingMethod = jwt.SigningMethodHS256

type JwtManager struct {
	config *config.Config
}

type TokenPair struct {
	AccessToken  *jwt.Token
	RefreshToken *jwt.Token
}

type CustomClaims struct {
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

func NewJwtManager(config *config.Config) *JwtManager {
	return &JwtManager{config: config}
}

func (j *JwtManager) Parse(token string) (*jwt.Token, error) {
	parser := jwt.NewParser()
	jwtToken, err := parser.Parse(token, func(t *jwt.Token) (any, error) {
		if t.Method != signingMethod {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}

		return []byte(j.config.JWTSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	return jwtToken, nil
}

func (j *JwtManager) IsAccessToken(token *jwt.Token) bool {
	jwtClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return false
	}

	if tokenType, ok := jwtClaims["token_type"]; ok {
		return tokenType == "access"
	}

	return false
}

func (j *JwtManager) GenerateTokenPair(userId uuid.UUID) (*TokenPair, error) {
	var err error
	jwtAccessToken := jwt.NewWithClaims(
		signingMethod,
		CustomClaims{
			TokenType: "access",
			Subject:   userId.String(),
			Issuer:    "http://" + j.config.APIServerHost + ":" + j.config.APIServerPort,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 15)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	)

	key := []byte(j.config.JWTSecret)
	signedAccessToken, err := jwtAccessToken.SignedString(key)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	accessToken, err := j.Parse(signedAccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to parse access token: %w", err)
	}

	jwtRefreshToken := jwt.NewWithClaims(
		signingMethod,
		CustomClaims{
			TokenType: "refresh",
			Subject:   userId.String(),
			Issuer:    "http://" + j.config.APIServerHost + ":" + j.config.APIServerPort,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24 * 15)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	)

	signedRefreshToken, err := jwtRefreshToken.SignedString(key)
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	refreshToken, err := j.Parse(signedRefreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to parse refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
