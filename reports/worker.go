package reports

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"log/slog"
	"time"

	"github.com/ArditZubaku/async-api/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type Worker struct {
	conf        *config.Config
	builder     *ReportBuilder
	logger      *slog.Logger
	sqsClient   *sqs.Client
	channel     chan types.Message
	concurrency uint
}

func NewWorker(
	conf *config.Config,
	builder *ReportBuilder,
	logger *slog.Logger,
	sqsClient *sqs.Client,
	maxConcurrency uint,
) *Worker {
	return &Worker{
		conf:        conf,
		builder:     builder,
		logger:      logger,
		sqsClient:   sqsClient,
		channel:     make(chan types.Message, maxConcurrency),
		concurrency: maxConcurrency,
	}
}

func (w *Worker) Start(ctx context.Context) error {
	queueUrlOutput, err := w.sqsClient.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: aws.String(w.conf.SQSQueue),
	})
	if err != nil {
		return fmt.Errorf("failed to get url for queue %s: %w", w.conf.SQSQueue, err)
	}

	w.logger.Info("starting worker",
		"queue", w.conf.SQSQueue,
		"queueUrl", queueUrlOutput.QueueUrl,
	)

	for i := range w.concurrency {
		go func(id uint) {
			w.logger.Info("starting worker goroutine", "id", id)
			for {
				select {
				case <-ctx.Done():
					w.logger.Warn(
						"worker is shutting down",
						"goroutine_id", id,
						"queue", w.conf.SQSQueue,
						"error", ctx.Err(),
					)
				case msg := <-w.channel:
					if err := w.processMessage(ctx, msg); err != nil {
						w.logger.Error(
							"failed to process message",
							"goroutine_id", id,
							"queue", w.conf.SQSQueue,
							"error", err,
						)
						continue
					}

					if _, err := w.sqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
						QueueUrl:      queueUrlOutput.QueueUrl,
						ReceiptHandle: msg.ReceiptHandle,
					}); err != nil {
						w.logger.Error(
							"failed to delete message from the queue ",
							"message_id", *msg.MessageId,
							"queue_url", *queueUrlOutput.QueueUrl,
							"error", err,
						)
					}

				}
			}
		}(i)
	}

	maxMsgs := min(int32(w.concurrency), 10)
	for {
		receiveMsgOutput, err := w.sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            queueUrlOutput.QueueUrl,
			MaxNumberOfMessages: maxMsgs,
		})
		if err != nil {
			w.logger.Error(
				"failed to receive message from queue",
				"queue", w.conf.SQSQueue,
				"error", err,
			)
			if ctx.Err() != nil {
				return ctx.Err()
			}
		}

		if len(receiveMsgOutput.Messages) == 0 {
			continue
		}

		for _, msg := range receiveMsgOutput.Messages {
			w.channel <- msg
		}
	}
}

func (w *Worker) processMessage(ctx context.Context, msg types.Message) error {
	w.logger.Info("processing message", "messageId", *msg.MessageId)
	if msg.Body == nil || *msg.Body == "" {
		w.logger.Warn("message body is empty", "message_id", msg.MessageId)
		return nil
	}

	var sqsMessage SQSMessage
	if err := json.Unmarshal([]byte(*msg.Body), &sqsMessage); err != nil {
		w.logger.Warn(
			"failed to unmarshal message",
			"message_id", *msg.MessageId,
			"body", *msg.Body,
			"error", err,
		)
		return nil
	}

	builderCtx, builderCancel := context.WithTimeout(ctx, time.Second*10)
	defer builderCancel()

	if _, err := w.builder.Build(builderCtx, sqsMessage.UserId, sqsMessage.ReportId); err != nil {
		return fmt.Errorf("failed to build report: %w", err)
	}

	return nil
}
