package gateway

import (
	"context"
	"errors"
	"io"
	"math/rand"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/kislerdm/minio-gateway/internal/docker"
	"github.com/kislerdm/minio-gateway/pkg/gateway/config"
	"github.com/minio/minio-go/v7"
)

func New(storageInstancesIDPrefix string) (*Config, error) {
	const defaultBucket = "store"

	cl, err := docker.NewClient()
	if err != nil {
		return nil, err
	}

	return &Gateway{
		StorageInstancesPrefix: storageInstancesIDPrefix,

		DefaultBucket:           defaultBucket,
		connectionDetailsReader: cl,
		connectionFactory:       newMinioAdapter,
		cacheObjectLocation:     map[string]string{},
		mu:                      &sync.RWMutex{},
	}, nil
}

type Gateway struct {
	cfg *config.Config

	// TODO: add invalidation logic. Imagine that a node where an object was written is down.
	//  Next time we try to write to it, the error will be returned,
	//    however we can just write to a another node instead.
	// Maps the object ID to the instance ID where it's stored.
	cacheObjectLocation map[string]string
	mu                  *sync.RWMutex
}

// Read reads the object given its ID.
func (s *Gateway) Read(ctx context.Context, id string) (dataReadCloser io.ReadCloser, found bool, err error) {
	instanceID, ok := s.cacheObjectLocation[id]
	if ok {
		conn, err := newStorageInstanceConnection(ctx, s.cfg.ConnectionDetailsReader, s.cfg.ConnectionFactory,
			instanceID)

		dataReadCloser, err = conn.ReadObject(ctx, s.defaultBucket, id, minio.GetObjectOptions{})
		if err != nil {
			if isNotFoundError(err) {
				err = nil
				return
			}
			return
		}

		found = true
		return
	}

	instances, err := findStorageInstances(ctx, s.connectionDetailsReader, s.StorageInstancesPrefix)
	if err != nil {
		return
	}

	if len(instances) == 0 {
		err = errors.New("cannot identify storage instances, check if cluster is running")
		return
	}

	// go round-robin over all hosts and try to find requested object.
	for instanceID, _ = range instances {
		conn, err := newStorageInstanceConnection(ctx, s.connectionDetailsReader, s.connectionFactory, instanceID)
		if err != nil {
			return
		}

		dataReadCloser, err = conn.ReadObject(ctx, s.defaultBucket, id, minio.GetObjectOptions{})
		if err != nil {
			if isNotFoundError(err) {
				continue
			}
			return
		}

		if dataReadCloser != nil {
			s.cacheObjectLocation[id] = instanceID
			found = true
			return
		}
	}

	err = nil
	return
}

// Write writes object to the storage.
func (s *Gateway) Write(ctx context.Context, id string, reader io.Reader) error {
	instanceID, ok := s.cacheObjectLocation[id]
	if ok {
		conn, err := newStorageInstanceConnection(ctx, s.connectionDetailsReader, s.connectionFactory, instanceID)
		if err != nil {
			return err
		}

		_, err = conn.PutObject(ctx, s.defaultBucket, id, reader, -1, minio.PutObjectOptions{})
		return err
	}

	instances, err := findStorageInstances(ctx, s.connectionDetailsReader, s.StorageInstancesPrefix)
	if err != nil {
		return err
	}

	if len(instances) == 0 {
		return errors.New("cannot identify storage instances, check if cluster is running")
	}

	// go round-robin over all hosts to find if the object is stored to one of storage nodes
	// it's required to ensure the "sticky"-condition: overwrite already existing object
	for instanceID, _ = range instances {
		conn, err := newStorageInstanceConnection(ctx, s.connectionDetailsReader, s.connectionFactory, instanceID)
		if err != nil {
			return err
		}
		found, err := s.objectExists(ctx, conn, id)
		if err != nil {
			return err
		}
		if found {
			s.cacheObjectLocation[id] = instanceID
			_, err = conn.PutObject(ctx, s.defaultBucket, id, reader, -1, minio.PutObjectOptions{})
			return err
		}
	}

	// define the instance to store new object
	instanceID = pickStorageInstance(instances, id)

	conn, err := newStorageInstanceConnection(ctx, s.connectionDetailsReader, s.connectionFactory, instanceID)
	if err != nil {
		return err
	}

	if _, err = conn.PutObject(ctx, s.defaultBucket, id, reader, -1, minio.PutObjectOptions{}); err != nil {
		return err
	}

	s.cacheObjectLocation[id] = instanceID
	return nil
}

func newStorageInstanceConnection(
	ctx context.Context, storageDetailsReader config.StorageConnectionDetailsReader,
	connectionFactory config.StorageConnectionFactory,
	id string,
) (config.StorageController, error) {
	// TODO: improve performance: cache connections
	ipAddress, accessKeyID, secretAccessKey, err := storageDetailsReader.Read(ctx, id)
	if err != nil {
		return nil, err
	}

	return connectionFactory(ipAddress, accessKeyID, secretAccessKey)
}

func (s *Gateway) objectExists(ctx context.Context, conn minioPort, id string) (bool, error) {
	_, err := conn.GetObjectACL(ctx, s.defaultBucket, id)
	if err != nil {
		if isNotFoundError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// pickStorageInstance randomly selects the storage instance.
func pickStorageInstance(storageInstanceIDs map[string]struct{}, _ string) (id string) {
	pick := rand.Intn(len(storageInstanceIDs))
	var i int
	for id = range storageInstanceIDs {
		if i == pick {
			return
		}
		i++
	}
	return
}

type minioPort interface {
	GetObjectACL(ctx context.Context, bucketName, objectName string) (*minio.ObjectInfo, error)
	BucketExists(ctx context.Context, bucketName string) (bool, error)
	PutObject(
		ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64,
		opts minio.PutObjectOptions,
	) (minio.UploadInfo, error)
	MakeBucket(ctx context.Context, bucketName string, opts minio.MakeBucketOptions) error
	ReadObject(ctx context.Context, bucketName string, objectName string, opts minio.GetObjectOptions) (
		io.ReadCloser, error,
	)
	IsOnline() bool
}

type ConnectionDetailsReader interface {
	ContainerList(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error)
	ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error)
}
