package config

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func ConfigAWS() (*dynamodb.Client, *s3.Client, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, nil, err
	}

	dbClient := dynamodb.NewFromConfig(cfg)
	s3Client := s3.NewFromConfig(cfg)

	return dbClient, s3Client, nil
}
