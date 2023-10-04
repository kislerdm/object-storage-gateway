package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// NewMinioClient initialises a new Minio client and checks if it's online.
func NewMinioClient(ctx context.Context, endpoint, accessKeyID, secretAccessKey string) (*MinioClient, error) {
	const defaultBucket = "store"

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds: credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
	})
	if err != nil {
		return nil, err
	}

	if !minioClient.IsOnline() {
		return nil, errors.New("the storage node is offline")
	}

	return &MinioClient{
		defaultBucket: defaultBucket,
		c:             minioClient,
	}, nil
}

type MinioClient struct {
	defaultBucket string
	c             *minio.Client
}

// Read reads an object given its id.
func (r MinioClient) Read(ctx context.Context, id string) (io.ReadCloser, error) {
	return r.c.GetObject(ctx, r.defaultBucket, id, minio.GetObjectOptions{})
}

// Write writes data as an object with id.
func (r MinioClient) Write(ctx context.Context, id string, data io.ReadCloser) error {
	// TODO: consider default bucket creation on creation stage.
	exists, err := r.c.BucketExists(ctx, r.defaultBucket)
	if err != nil {
		return fmt.Errorf("cannot store the object: %w", err)
	}
	if !exists {
		if err := r.c.MakeBucket(ctx, r.defaultBucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("cannot create bucket to store objects %w", err)
		}
	}

	// TODO: add size definition to minimize memory allocation
	_, err = r.c.PutObject(ctx, r.defaultBucket, id, data, -1, minio.PutObjectOptions{})
	return err
}

// IsNotFoundError defines if the error corresponds to obj not found.
func IsNotFoundError(err error) bool {
	switch e := err.(type) {
	case minio.ErrorResponse:
		return e.StatusCode == http.StatusNotFound
	default:
		return false
	}
}
