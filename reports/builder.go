package reports

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
	"uuid"

	"github.com/ArditZubaku/async-api/config"
	"github.com/ArditZubaku/async-api/store"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type ReportBuilder struct {
	conf        *config.Config
	reportStore *store.ReportStore
	lozClient   *LozClient
	s3Client    *s3.Client
	logger      *slog.Logger
}

func NewReportBuilder(
	conf *config.Config,
	reportStore *store.ReportStore,
	lozClient *LozClient,
	s3Client *s3.Client,
	logger *slog.Logger,
) *ReportBuilder {
	return &ReportBuilder{
		conf:        conf,
		reportStore: reportStore,
		lozClient:   lozClient,
		s3Client:    s3Client,
		logger:      logger,
	}
}

func (b *ReportBuilder) Build(ctx context.Context, userId, reportId uuid.UUID) (report *store.Report, err error) {
	report, err = b.reportStore.ByPrimaryKey(ctx, userId, reportId)
	if err != nil {
		return nil, err
	}

	if report.StartedAt != nil {
		return report, nil
	}

	defer func() {
		if err != nil {
			now := time.Now()
			errMsg := err.Error()
			report.FailedAt = &now
			report.ErrorMessage = &errMsg
			if _, updateErr := b.reportStore.Update(ctx, report); updateErr != nil {
				b.logger.Error("failed to update report when failed", "err", updateErr)
			}
		}
	}()

	now := time.Now()
	report.StartedAt = &now
	report.CompletedAt = nil
	report.FailedAt = nil
	report.ErrorMessage = nil
	report.DownloadUrl = nil
	report.DownloadUrlExpiresAt = nil
	report.OutputFilePath = nil

	report, err = b.reportStore.Update(ctx, report)
	if err != nil {
		return nil, err
	}

	resp, err := b.lozClient.GetMonsters(ctx)
	if err != nil {
		return nil, err
	}

	if len(resp) == 0 {
		return nil, errors.New("no monsters data found")
	}

	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	defer func() {
		if err := gzipWriter.Close(); err != nil {
			b.logger.Error("failed to close gzip writer: %w", err)
		}
	}()

	csvWriter := csv.NewWriter(gzipWriter)

	header := []string{"name", "id", "category", "description", "image", "common_locations", "drops", "dlc"}
	if err := csvWriter.Write(header); err != nil {
		return nil, fmt.Errorf("failed to write csv header: %w", err)
	}

	for _, monster := range resp {
		csvRow := []string{
			monster.Name,
			strconv.Itoa(monster.Id),
			monster.Category,
			monster.Description,
			monster.Image,
			strings.Join(monster.CommonLocations, ","),
			strings.Join(monster.Drops, ","),
			strconv.FormatBool(monster.Dlc),
		}

		if err := csvWriter.Write(csvRow); err != nil {
			return nil, fmt.Errorf("failed to write csv row: %w", err)
		}

		if csvWriter.Error() != nil {
			return nil, fmt.Errorf("failed to write csv: %w", csvWriter.Error())
		}
	}

	csvWriter.Flush()
	if csvWriter.Error() != nil {
		return nil, fmt.Errorf("failed to flush csv: %w", csvWriter.Error())
	}

	key := "/users/" + userId.String() + "/reports/" + reportId.String() + ".csv.gz"
	_, err = b.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(b.conf.S3Bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(buf.Bytes()),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload report to S3: %w", err)
	}
	b.logger.Info("successfully uploaded report to S3")

	now = time.Now()
	report.CompletedAt = &now
	report.OutputFilePath = &key

	report, err = b.reportStore.Update(ctx, report)
	if err != nil {
		return nil, err
	}

	b.logger.Info(
		"successfully generated report",
		"report_id", report.Id,
		"user_id", userId.String(),
		"path", key,
	)

	return report, nil
}
