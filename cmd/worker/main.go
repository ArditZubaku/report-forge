package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ArditZubaku/async-api/config"
	"github.com/ArditZubaku/async-api/reports"
	"github.com/ArditZubaku/async-api/store"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	conf, err := config.New()
	if err != nil {
		return err
	}

	jsonHandler := slog.NewJSONHandler(os.Stdout, nil)
	logger := slog.New(jsonHandler)

	db, err := store.NewPG(conf.DatabaseURL())
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("failed to close database", "error", err)
		}
	}()

	dataStore := store.New(db)

	awsConf, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}

	s3Client := s3.NewFromConfig(awsConf, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(conf.S3LocalStackEndpoint)
		options.UsePathStyle = true
	})

	sqsClient := sqs.NewFromConfig(awsConf, func(options *sqs.Options) {
		options.BaseEndpoint = aws.String(conf.SQSLocalStackEndpoint)
	})

	lozClient := reports.NewLozClient(&http.Client{Timeout: time.Second * 10})
	builder := reports.NewReportBuilder(conf, dataStore.Reports, lozClient, s3Client, logger)

	worker := reports.NewWorker(conf, builder, logger, sqsClient, 2)

	return worker.Start(ctx)
}
