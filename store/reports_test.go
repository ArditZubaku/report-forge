package store_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/ArditZubaku/async-api/fixtures"
	"github.com/ArditZubaku/async-api/store"
	"github.com/stretchr/testify/require"
)

func TestReportStore(t *testing.T) {
	env := fixtures.NewTestEnv(t)
	cleanup := env.SetupDB(t)
	t.Cleanup(func() {
		cleanup(t)
	})

	ctx := context.Background()

	reportStore := store.NewReportStore(env.DB)
	userStore := store.NewUserStore(env.DB)
	user, err := userStore.CreateUser(ctx, "test@email.com", "testPassword")
	require.NoError(t, err)

	var now time.Time
	require.NoError(t, env.DB.QueryRowContext(ctx, "SELECT CURRENT_TIMESTAMP").Scan(&now))
	report, err := reportStore.Create(ctx, user.Id, "test_type")
	require.NoError(t, err)
	require.Equal(t, user.Id, report.UserId)
	require.Equal(t, report.ReportType, "test_type")
	require.Less(t, now.UnixNano(), report.CreatedAt.UnixNano())

	startedAt := report.CreatedAt.Add(time.Second)
	completedAt := report.CreatedAt.Add(2 * time.Second)
	failedAt := report.CreatedAt.Add(3 * time.Second)
	errorMsg := "there was a failure"
	downloadUrl := "http://localhost:8080/reports"
	downloadUrlExpiresAt := report.CreatedAt.Add(4 * time.Second)

	report.ReportType = "another_type"
	report.StartedAt = &startedAt
	report.CompletedAt = &completedAt
	report.FailedAt = &failedAt
	report.ErrorMessage = &errorMsg
	report.DownloadUrl = &downloadUrl
	report.DownloadUrlExpiresAt = &downloadUrlExpiresAt

	report2, err := reportStore.Update(ctx, report)
	require.NoError(t, err)
	require.Equal(t, report, report2)
	require.Equal(t, report.CreatedAt.UnixNano(), report2.CreatedAt.UnixNano())
	require.Equal(t, report.CompletedAt.UnixNano(), report2.CompletedAt.UnixNano())
	require.Equal(t, report.ErrorMessage, report2.ErrorMessage)
	require.Equal(t, report.DownloadUrl, report2.DownloadUrl)
	require.Equal(t, report.DownloadUrlExpiresAt, report2.DownloadUrlExpiresAt)
	require.Equal(t, report.OutputFilePath, report2.OutputFilePath)

	report3, err := reportStore.ByPrimaryKey(ctx, report.UserId, report.Id)
	require.NoError(t, err)
	require.Equal(t, report, report3)
	require.Equal(t, report.CreatedAt.UnixNano(), report3.CreatedAt.UnixNano())
	require.Equal(t, report.CompletedAt.UnixNano(), report3.CompletedAt.UnixNano())
	require.Equal(t, report.ErrorMessage, report3.ErrorMessage)
	require.Equal(t, report.DownloadUrl, report3.DownloadUrl)
	require.Equal(t, report.DownloadUrlExpiresAt, report3.DownloadUrlExpiresAt)
	require.Equal(t, report.OutputFilePath, report3.OutputFilePath)

	err = reportStore.Delete(ctx, report.Id)
	require.NoError(t, err)

	_, err = reportStore.ByPrimaryKey(ctx, report.UserId, report.Id)
	require.Error(t, err)
	require.ErrorIs(t, err, sql.ErrNoRows)
}
