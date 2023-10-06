package gateway

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"

	"github.com/minio/minio-go/v7"
)

type MimioConnectionDetails struct {
	IPAddress       string
	AccessKeyID     string
	SecretAccessKey string
}

func readKeys(m map[string]MimioConnectionDetails) []string {
	var o = make([]string, len(m))
	var i uint8
	for k := range m {
		o[i] = k
	}
	return o
}

type ClusterAccessDetailsReader interface {
	Read(ctx context.Context, prefix string) (map[string]MimioConnectionDetails, error)
}

func New(hostPrefix string, connectionDetailsReader ClusterAccessDetailsReader) (*Client, error) {
	const defaultBucket = "store"

	return &Client{
		StorageHostPrefix: hostPrefix,

		defaultBucket:           defaultBucket,
		connectionDetailsReader: connectionDetailsReader,
		newConnectionFn:         newMinioClientAdapter,
		cacheObjectLocation:     map[string]string{},
	}, nil
}

type Client struct {
	StorageHostPrefix string

	defaultBucket           string
	connectionDetailsReader ClusterAccessDetailsReader
	newConnectionFn         minioConnectionFactory

	// TODO: add invalidation logic. Imagine that a node where an object was written is down.
	//  Next time we try to write to it, the error will be returned,
	//    however we can just write to a another node instead.
	cacheObjectLocation map[string]string
}

func (s *Client) newStorageConnection(ctx context.Context, nodeID string) (minioConnectionPort, error) {
	hostsConnectionDetailsMap, err := s.connectionDetailsReader.Read(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	connectionDetails, ok := hostsConnectionDetailsMap[nodeID]
	if !ok {
		return nil, errors.New("no connection details found for the node with the label " + nodeID)
	}

	return s.newConnectionFn(connectionDetails.IPAddress, connectionDetails.AccessKeyID, connectionDetails.SecretAccessKey)
}

func (s *Client) Read(ctx context.Context, id string) (data io.ReadCloser, found bool, err error) {
	hostID, ok := s.cacheObjectLocation[id]
	if ok {
		data, err = s.read(ctx, id, hostID)

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

	hostsConnectionDetailsMap, err := s.connectionDetailsReader.Read(ctx, s.StorageHostPrefix)
	if err != nil {
		return
	}

	// go round-robin over all hosts
	// TODO: improve performance: cache connections (?)
	for hostID, _ := range hostsConnectionDetailsMap {
		data, err = s.read(ctx, id, hostID)
		if data != nil {
			s.setObjectLocation(id, hostID)
			found = true
			return
		}
		if !isNotFoundError(err) {
			return
		}
	}

	data = nil
	err = nil
	return
}

func (s *Client) read(ctx context.Context, id string, hostName string) (io.ReadCloser, error) {
	conn, err := s.newStorageConnection(ctx, hostName)
	if err != nil {
		return nil, err
	}
	return conn.GetObject(ctx, s.defaultBucket, id, minio.GetObjectOptions{})
}

func (s *Client) Write(ctx context.Context, id string, data io.Reader) error {
	hostName, ok := s.cacheObjectLocation[id]
	if ok {
		return s.write(ctx, id, data, hostName)
	}

	hostsConnectionDetailsMap, err := s.connectionDetailsReader.Read(ctx, s.StorageHostPrefix)
	if err != nil {
		return err
	}

	// go round-robin over all hosts to find if the object is stored to one of storage nodes
	// it's required to ensure the "sticky"-condition: overwrite already existing object
	for hostID, _ := range hostsConnectionDetailsMap {
		found, err, conn := s.objectExists(ctx, id, hostID)
		if err != nil {
			return err
		}
		if found {
			// TODO: add size definition to minimize memory allocation
			_, err = conn.PutObject(ctx, s.defaultBucket, id, data, -1, minio.PutObjectOptions{})
			return err
		}
	}

	// define host to store new object
	host := pickStorage(readKeys(hostsConnectionDetailsMap), id)

	return s.write(ctx, id, data, host)
}

func (s *Client) setObjectLocation(id string, host string) {
	if s.cacheObjectLocation == nil {
		s.cacheObjectLocation = map[string]string{}
	}
	s.cacheObjectLocation[id] = host
}

func (s *Client) write(ctx context.Context, id string, data io.Reader, hostName string) error {
	conn, err := s.newStorageConnection(ctx, hostName)
	if err != nil {
		return err
	}

	exists, err := conn.BucketExists(ctx, s.defaultBucket)
	if err != nil {
		return fmt.Errorf("cannot store the object: %w", err)
	}
	if !exists {
		if err := conn.MakeBucket(ctx, s.defaultBucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("cannot create bucket to store objects %w", err)
		}
	}

	// TODO: add size definition to minimize memory allocation
	_, err = conn.PutObject(ctx, s.defaultBucket, id, data, -1, minio.PutObjectOptions{})
	return err
}

func (s *Client) objectExists(ctx context.Context, id string, hostName string) (bool, error, minioConnectionPort) {
	conn, err := s.newStorageConnection(ctx, hostName)
	if err != nil {
		return false, err, nil
	}
	_, err = conn.GetObjectACL(ctx, s.defaultBucket, id)
	if err != nil {
		if isNotFoundError(err) {
			return false, nil, conn
		}
		return false, err, conn
	}
	return true, err, conn
}

func pickStorage(hosts []string, _ string) string {
	return hosts[rand.Intn(len(hosts))]
}

type minioConnectionFactory func(endpoint, accessKeyID, secretAccessKey string) (minioConnectionPort, error)

type minioConnectionPort interface {
	GetObjectACL(ctx context.Context, bucketName, objectName string) (*minio.ObjectInfo, error)
	BucketExists(ctx context.Context, bucketName string) (bool, error)
	PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error)
	MakeBucket(ctx context.Context, bucketName string, opts minio.MakeBucketOptions) error
	GetObject(ctx context.Context, bucketName string, objectName string, opts minio.GetObjectOptions) (io.ReadCloser, error)
	IsOnline() bool
}
