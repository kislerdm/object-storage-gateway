package minio

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/kislerdm/minio-gateway/pkg/gateway"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func NewClient(ipAddress, accessKeyID, secretAccessKey string) (gateway.ObjectReadWriteFinder, error) {
	const defaultPort = "9000"
	host := ipAddress + ":" + defaultPort
	c, err := minio.New(host, &minio.Options{
		Creds: credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
	})
	if err != nil {
		return nil, err
	}
	if !c.IsOnline() {
		return nil, errors.New("the storage node is offline")
	}
	return &Client{c}, nil
}

type Client struct {
	*minio.Client
}

func (c *Client) Read(ctx context.Context, bucketName, objectName string) (io.ReadCloser, bool, error) {
	exists, _ := c.BucketExists(ctx, bucketName)
	if !exists {
		return nil, false, nil
	}
	reader, err := c.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		if isNotFoundError(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return reader, true, nil
}

func (c *Client) Write(ctx context.Context, bucketName, objectName string, reader io.Reader) error {
	exists, err := c.BucketExists(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("cannot store the object: %w", err)
	}
	if !exists {
		if err = c.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("cannot create bucket to store objects %w", err)
		}
	}

	// TODO: find the way to identify the object size to optimize the process
	_, err = c.PutObject(ctx, bucketName, objectName, reader, -1, minio.PutObjectOptions{})
	return err
}

func (c *Client) Find(ctx context.Context, bucketName, objectName string) (bool, error) {
	_, err := c.GetObjectACL(ctx, bucketName, objectName)
	if err != nil {
		if isNotFoundError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// isNotFoundError defines if the Minion client's error indicated that the obj is not found.
func isNotFoundError(err error) bool {
	switch e := err.(type) { //nolint:errorlint // no wrapped is expected
	case minio.ErrorResponse:
		return e.StatusCode == http.StatusNotFound
	default:
		return false
	}
}
