package config

import (
	"context"
	"errors"
	"os"

	"cloud.google.com/go/firestore"
)

// ConfigFirestore initialises a Firestore client using Application Default
// Credentials. Requires GCP_PROJECT_ID env var.
func ConfigFirestore() (*firestore.Client, error) {
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		return nil, errors.New("GCP_PROJECT_ID is not set")
	}

	client, err := firestore.NewClient(context.Background(), projectID)
	if err != nil {
		return nil, err
	}

	return client, nil
}
