package gateway

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"sort"
)

// New initializes a Gateway.
func New(
	storageInstancesSelector string,
	storageBucket string,
	storageDiscoveryClient StorageConnectionReadFinder,
	newStorageConnectionFn StorageConnectionFn,
	logger *slog.Logger,
) (*Gateway, error) {
	if storageDiscoveryClient == nil {
		return nil, errors.New("storageDiscoveryClient must be not nil")
	}

	if newStorageConnectionFn == nil {
		return nil, errors.New("newStorageConnectionFn must be not nil")
	}

	if storageInstancesSelector == "" {
		return nil, errors.New("storageInstancesSelector must set as not empty string")
	}

	o := &Gateway{
		storageInstancesSelector: storageInstancesSelector,
		storageBucket:            storageBucket,
		storageDiscoveryClient:   storageDiscoveryClient,
		newStorageConnectionFn:   newStorageConnectionFn,
		Logger:                   logger,
	}

	const defaultBucket = "store"
	if o.storageBucket != "" {
		o.storageBucket = defaultBucket
	}

	if o.Logger == nil {
		o.Logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelDebug,
		}))
	}

	o.Logger = o.Logger.WithGroup("gw")

	return o, nil
}

type Gateway struct {
	// storageInstancesSelector selector to identify instances in the storage cluster.
	storageInstancesSelector string
	// storageBucket  bucket for RW operations.
	storageBucket string

	storageDiscoveryClient StorageConnectionReadFinder
	newStorageConnectionFn StorageConnectionFn

	Logger *slog.Logger
}

// Read reads the object given its ID.
func (s *Gateway) Read(ctx context.Context, id string) (io.ReadCloser, bool, error) {
	instances, err := s.storageDiscoveryClient.Find(ctx, s.storageInstancesSelector)
	if err != nil {
		return nil, false, err
	}

	if len(instances) == 0 {
		return nil, false, errors.New("cannot identify storage instances, check if cluster is running")
	}

	// go round-robin over all hosts and try to find requested object.
	for instanceID := range instances {
		conn, err := s.newStorageInstanceConnection(ctx, instanceID)
		if err != nil {
			return nil, false, err
		}

		s.Logger.Debug("searching",
			slog.String("operation", "read"),
			slog.String("instanceID", instanceID),
			slog.String("objectID", id),
		)

		found, err := conn.Find(ctx, s.storageBucket, id)
		if err != nil {
			return nil, false, err
		}

		if found {
			s.Logger.Debug("reading",
				slog.String("operation", "read"),
				slog.String("instanceID", instanceID),
				slog.String("objectID", id),
			)

			dataReadCloser, _, err := conn.Read(ctx, s.storageBucket, id)
			if err != nil {
				return nil, false, err
			}

			return dataReadCloser, found, nil
		}
	}

	return nil, false, nil
}

// Write writes object to the storage.
func (s *Gateway) Write(ctx context.Context, id string, reader io.Reader, objectSizeBytes int64) error {
	instances, err := s.storageDiscoveryClient.Find(ctx, s.storageInstancesSelector)
	if err != nil {
		return err
	}

	if len(instances) == 0 {
		return errors.New("cannot identify storage instances, check if cluster is running")
	}

	// go round-robin over all hosts to find if the object is stored to one of storage nodes
	// it's required to ensure the "sticky"-condition: overwrite already existing object
	for instanceID := range instances {
		s.Logger.Debug("searching",
			slog.String("operation", "write"),
			slog.String("instanceID", instanceID),
			slog.String("objectID", id),
		)

		var conn ObjectReadWriteFinder
		conn, err = s.newStorageInstanceConnection(ctx, instanceID)
		if err != nil {
			return err
		}

		var found bool
		found, err = conn.Find(ctx, s.storageBucket, id)
		if err != nil {
			return err
		}

		if found {
			s.Logger.Debug("overwriting",
				slog.String("operation", "write"),
				slog.String("instanceID", instanceID),
				slog.String("objectID", id),
			)

			return conn.Write(ctx, s.storageBucket, id, reader, objectSizeBytes)
		}
	}

	// define the instance to store new object
	instanceID := pickStorageInstance(instances, id)

	conn, err := s.newStorageInstanceConnection(ctx, instanceID)
	if err != nil {
		return err
	}

	s.Logger.Debug("creating",
		slog.String("operation", "write"),
		slog.String("instanceID", instanceID),
		slog.String("objectID", id),
	)

	return conn.Write(ctx, s.storageBucket, id, reader, objectSizeBytes)
}

func (s *Gateway) newStorageInstanceConnection(ctx context.Context, id string) (ObjectReadWriteFinder, error) {
	ipAddress, accessKeyID, secretAccessKey, err := s.storageDiscoveryClient.Read(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.newStorageConnectionFn(ipAddress, accessKeyID, secretAccessKey)
}

// pickStorageInstance randomly selects the storage instance.
func pickStorageInstance(storageInstanceIDs map[string]struct{}, objectID string) (id string) {
	if len(storageInstanceIDs) == 0 {
		return ""
	}

	ids := readSortedMapKeys(storageInstanceIDs)

	switch cntInstances := len(ids); cntInstances {
	case 1:
		return ids[0]
	default:
		idInx := hash(objectID) % cntInstances
		return ids[idInx]
	}
}

func hash(id string) int {
	var o int32
	for _, r := range id {
		o += r
	}
	return int(o)
}

func readSortedMapKeys(m map[string]struct{}) []string {
	var o = make([]string, len(m))
	var i int
	for k := range m {
		o[i] = k
		i++
	}
	sort.Strings(o)
	return o
}

// StorageConnectionReadFinder defines the port for "service discovery".
type StorageConnectionReadFinder interface {
	// Find scans the "service discovery" records to find instances and return their IDs.
	Find(ctx context.Context, instanceNameFilter string) (map[string]struct{}, error)
	// Read reads ip address and authentication details to connect to the instance.
	Read(ctx context.Context, id string) (ipAddress, accessKeyID, secretAccessKey string, err error)
}

// ObjectReadWriteFinder defines the port to the storage instance.
type ObjectReadWriteFinder interface {
	// Read reads the object.
	Read(ctx context.Context, bucketName, objectName string) (io.ReadCloser, bool, error)

	// Write writes the object.
	Write(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSizeBytes int64) error

	// Find identifies if the object can be found in the instance.
	Find(ctx context.Context, bucketName, objectName string) (bool, error)
}

// StorageConnectionFn defines the factory of ObjectReadWriteFinder.
type StorageConnectionFn func(endpoint, accessKeyID, secretAccessKey string) (ObjectReadWriteFinder, error)
