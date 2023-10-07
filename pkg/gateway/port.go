package gateway

import (
	"context"
	"io"
)

type StorageInstancesFinder interface {
	// Find scans the service discovery records to find instances and return their IDs.
	Find(ctx context.Context, instanceNameFilter string) (map[string]struct{}, error)
}

type StorageConnectionDetailsReader interface {
	// Read reads ip address and authentication details to connect to the instance.
	Read(ctx context.Context, id string) (ipAddress, accessKeyID, secretAccessKey string, err error)
}

type StorageController interface {
	Read(ctx context.Context, bucketName, objectName string) (io.ReadCloser, bool, error)
	Write(ctx context.Context, bucketName, objectName string, reader io.Reader) error
	Detected(ctx context.Context, bucketName, objectName string) (bool, error)
}

type StorageConnectionFactory func(endpoint, accessKeyID, secretAccessKey string) (StorageController, error)
