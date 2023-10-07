package minio

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func NewClient(ipAddress, accessKeyID, secretAccessKey string) (*Client, error) {
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
	// TODO: find the way to identify the object size to optimize the process
	_, err := c.PutObject(ctx, bucketName, objectName, reader, -1, minio.PutObjectOptions{})
	return err
}

func (c *Client) Detected(ctx context.Context, bucketName, objectName string) (bool, error) {
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
	switch e := err.(type) {
	case minio.ErrorResponse:
		return e.StatusCode == http.StatusNotFound
	default:
		return false
	}
}
