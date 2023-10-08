package gateway

import (
	"context"
	"io"
	"log/slog"
)

// Config defines Gateway configuration.
type Config struct {
	// StorageInstancesSelector selector to identify instances in the storage cluster.
	StorageInstancesSelector string

	// DefaultBucket bucket for RW operations.
	DefaultBucket string

	StorageInstancesFinder         StorageInstancesFinder
	StorageConnectionDetailsReader StorageConnectionDetailsReader
	NewStorageConnectionFn         StorageConnectionFactory
	Logger                         *slog.Logger
}

// StorageInstancesFinder defines the port to the "service discovery" controller.
type StorageInstancesFinder interface {
	// Find scans the "service discovery" records to find instances and return their IDs.
	Find(ctx context.Context, instanceNameFilter string) (map[string]struct{}, error)
}

// StorageConnectionDetailsReader defines the port to the "service discovery" controller.
type StorageConnectionDetailsReader interface {
	// Read reads ip address and authentication details to connect to the instance.
	Read(ctx context.Context, id string) (ipAddress, accessKeyID, secretAccessKey string, err error)
}

// StorageController defines the port to the storage instance.
type StorageController interface {
	// Read reads the object.
	Read(ctx context.Context, bucketName, objectName string) (io.ReadCloser, bool, error)

	// Write writes the object.
	Write(ctx context.Context, bucketName, objectName string, reader io.Reader) error

	// Detected identifies if the object can be found in the instance.
	Detected(ctx context.Context, bucketName, objectName string) (bool, error)
}

// StorageConnectionFactory defines the factory of StorageController.
type StorageConnectionFactory func(endpoint, accessKeyID, secretAccessKey string) (StorageController, error)
