package store_test

import (
	"context"
	"testing"

	"github.com/ArditZubaku/async-api/apiserver"
	"github.com/ArditZubaku/async-api/fixtures"
	"github.com/ArditZubaku/async-api/store"
	"github.com/stretchr/testify/require"
)

func TestRefreshTokenStore(t *testing.T) {
	ctx := context.Background()

	env := fixtures.NewTestEnv(t)
	cleanup := env.SetupDB(t)
	t.Cleanup(func() {
		cleanup(t)
	})

	refreshTokenStore := store.NewRefreshTokenStore(env.DB)
	userStore := store.NewUserStore(env.DB)
	jwtManager := apiserver.NewJwtManager(env.Config)

	user, err := userStore.CreateUser(ctx, "test@email.com", "testPassword")
	require.NoError(t, err)
	tokenPair, err := jwtManager.GenerateTokenPair(user.Id)
	require.NoError(t, err)

	refreshTokenRecord, err := refreshTokenStore.Create(ctx, user.Id, tokenPair.RefreshToken)
	require.NoError(t, err)
	require.Equal(t, user.Id, refreshTokenRecord.UserID)
	expectedExpiration, err := tokenPair.RefreshToken.Claims.GetExpirationTime()
	require.NoError(t, err)
	require.Equal(t, expectedExpiration.Time.UnixMilli(), refreshTokenRecord.ExpiresAt.UnixMilli())

	refreshTokenRecord2, err := refreshTokenStore.ByPrimaryKey(ctx, user.Id, tokenPair.RefreshToken)
	require.NoError(t, err)
	require.Equal(t, refreshTokenRecord.UserID, refreshTokenRecord2.UserID)
	require.Equal(t, refreshTokenRecord.HashedToken, refreshTokenRecord2.HashedToken)
	require.Equal(t, refreshTokenRecord.ExpiresAt, refreshTokenRecord2.ExpiresAt)
	require.Equal(t, refreshTokenRecord.CreatedAt, refreshTokenRecord2.CreatedAt)

	result, err := refreshTokenStore.DeleteUserTokens(ctx, user.Id)
	require.NoError(t, err)
	rowsAffected, err := result.RowsAffected()
	require.NoError(t, err)
	require.Equal(t, int64(1), rowsAffected)
}
