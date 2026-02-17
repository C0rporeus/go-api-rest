package config

import (
	"context"
	"errors"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// ConfigAWS initialises a DynamoDB client using the default AWS credential
// chain (env vars → shared credentials → IAM role). Requires AWS_REGION.
func ConfigAWS() (*dynamodb.Client, error) {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		return nil, errors.New("AWS_REGION is not set")
	}

	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion(region),
	)
	if err != nil {
		return nil, err
	}

	return dynamodb.NewFromConfig(cfg), nil
}
