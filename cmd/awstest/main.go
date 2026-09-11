package main

import (
	"context"
	"fmt"

	"github.com/ArditZubaku/async-api/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

func main() {
	ctx := context.Background()
	sdkConfig, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		panic(err)
	}

	conf, err := config.New()
	if err != nil {
		panic(err)
	}
	s3Client := s3.NewFromConfig(sdkConfig, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(conf.S3LocalStackEndpoint)
		options.UsePathStyle = true
	})
	lbRes, err := s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		panic(err)
	}

	for _, bucket := range lbRes.Buckets {
		fmt.Println(*bucket.Name)
	}

	sqsClient := sqs.NewFromConfig(sdkConfig, func(options *sqs.Options) {
		options.BaseEndpoint = aws.String(conf.SQSLocalStackEndpoint)
	})

	lqRes, err := sqsClient.ListQueues(ctx, &sqs.ListQueuesInput{})
	if err != nil {
		panic(err)
	}

	for _, queue := range lqRes.QueueUrls {
		fmt.Println(queue)
	}
}
