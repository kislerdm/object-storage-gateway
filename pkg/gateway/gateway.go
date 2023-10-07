package gateway

import (
	"context"
	"errors"
	"io"
	"math/rand"
)

// New initializes a Gateway using the Config.
func New(cfg Config) (*Gateway, error) {
	const defaultBucket = "store"

	if cfg.DefaultBucket == "" {
		cfg.DefaultBucket = defaultBucket
	}

	return &Gateway{
		cfg:                 &cfg,
		cacheObjectLocation: map[string]string{},
	}, nil
}

type Gateway struct {
	cfg *Config

	// TODO: add invalidation logic. Imagine that a node where an object was written is down.
	//  Next time we try to write to it, the error will be returned,
	//    however we can just write to a another node instead.
	// Maps the object ID to the instance ID where it's stored.
	cacheObjectLocation map[string]string

	// TODO: think how to lock Write operation to make sure that two requests with identical objectID
	//  won't result to uncertain location of the object.
	//  Example. Two simultaneous write request with the same ObjectID = 'foo'.
	//  The object with that ID was not present in the cluster before the requests. Without the lock,
	//  two objects with the same ID may end up in two different nodes.
}

// Read reads the object given its ID.
func (s *Gateway) Read(ctx context.Context, id string) (dataReadCloser io.ReadCloser, found bool, err error) {
	var conn StorageController

	instanceID, ok := s.cacheObjectLocation[id]
	if ok {
		conn, err = s.newStorageInstanceConnection(ctx, instanceID)
		if err != nil {
			return
		}
		return conn.Read(ctx, s.cfg.DefaultBucket, id)
	}

	instances, err := s.cfg.StorageInstancesFinder.Find(ctx, s.cfg.StorageInstancesPrefix)
	if err != nil {
		return
	}

	if len(instances) == 0 {
		err = errors.New("cannot identify storage instances, check if cluster is running")
		return
	}

	// go round-robin over all hosts and try to find requested object.
	for instanceID, _ = range instances {
		conn, err = s.newStorageInstanceConnection(ctx, instanceID)
		if err != nil {
			return
		}
		dataReadCloser, found, err = conn.Read(ctx, s.cfg.DefaultBucket, id)
		if found {
			s.cacheObjectLocation[id] = instanceID
		}
	}
	return
}

// Write writes object to the storage.
func (s *Gateway) Write(ctx context.Context, id string, reader io.Reader) error {
	instanceID, ok := s.cacheObjectLocation[id]
	if ok {
		conn, err := s.newStorageInstanceConnection(ctx, instanceID)
		if err != nil {
			return err
		}
		return conn.Write(ctx, s.cfg.DefaultBucket, id, reader)
	}

	instances, err := s.cfg.StorageInstancesFinder.Find(ctx, s.cfg.StorageInstancesPrefix)
	if err != nil {
		return err
	}

	if len(instances) == 0 {
		return errors.New("cannot identify storage instances, check if cluster is running")
	}

	// go round-robin over all hosts to find if the object is stored to one of storage nodes
	// it's required to ensure the "sticky"-condition: overwrite already existing object
	for instanceID, _ = range instances {
		conn, err := s.newStorageInstanceConnection(ctx, instanceID)
		if err != nil {
			return err
		}
		found, err := conn.Detected(ctx, s.cfg.DefaultBucket, id)
		if err != nil {
			return err
		}
		if found {
			s.cacheObjectLocation[id] = instanceID
			return conn.Write(ctx, s.cfg.DefaultBucket, id, reader)
		}
	}

	// define the instance to store new object
	instanceID = pickStorageInstance(instances, id)

	conn, err := s.newStorageInstanceConnection(ctx, instanceID)
	if err != nil {
		return err
	}

	if err := conn.Write(ctx, s.cfg.DefaultBucket, id, reader); err != nil {
		return err
	}

	s.cacheObjectLocation[id] = instanceID
	return nil
}

func (s *Gateway) newStorageInstanceConnection(ctx context.Context, id string) (StorageController, error) {
	// TODO: improve performance: cache connections
	ipAddress, accessKeyID, secretAccessKey, err := s.cfg.StorageConnectionDetailsReader.Read(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.cfg.NewStorageConnectionFn(ipAddress, accessKeyID, secretAccessKey)
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
