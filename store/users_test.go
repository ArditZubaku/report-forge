package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/ArditZubaku/async-api/fixtures"
	"github.com/ArditZubaku/async-api/store"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/stretchr/testify/require"
)

func TestUserStore(t *testing.T) {
	testEnv := fixtures.NewTestEnv(t)
	cleanup := testEnv.SetupDB(t)
	t.Cleanup(func() {
		cleanup(t)
	})

	ctx := context.Background()
	now := time.Now()

	userStore := store.NewUserStore(testEnv.DB)
	user, err := userStore.CreateUser(ctx, "test@test.com", "testingPassword")
	require.NoError(t, err)

	require.Equal(t, "test@test.com", user.Email)
	require.NoError(t, user.ComparePassword("testingPassword"))
	require.Less(t, now.UnixNano(), user.CreatedAt.UnixNano())

	user2, err := userStore.ById(ctx, user.Id)
	require.NoError(t, err)
	require.Equal(t, user.Email, user2.Email)
	require.Equal(t, user.Id, user2.Id)
	require.Equal(t, user.HashedPasswordBase64, user2.HashedPasswordBase64)
	require.Equal(t, user.CreatedAt.UnixNano(), user2.CreatedAt.UnixNano())

	user3, err := userStore.ByEmail(ctx, user.Email)
	require.NoError(t, err)
	require.Equal(t, user.Email, user3.Email)
	require.Equal(t, user.Id, user3.Id)
	require.Equal(t, user.HashedPasswordBase64, user3.HashedPasswordBase64)
	require.Equal(t, user.CreatedAt.UnixNano(), user3.CreatedAt.UnixNano())
}

func handleErr(t *testing.T, f func() error) {
	t.Helper()
	if err := f(); err != nil {
		t.Fail()
	}
}
