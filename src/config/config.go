package config

import (
	"context"
	"errors"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

func ConfigAWS() (*dynamodb.Client, error) {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		return nil, errors.New("AWS_REGION is not set")
	}

	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			os.Getenv("AWS_ACCESS_KEY_ID"),
			os.Getenv("AWS_SECRET_ACCESS_KEY"),
			"")),
	)
	if err != nil {
		return nil, err
	}

	return dynamodb.NewFromConfig(cfg), nil
}
