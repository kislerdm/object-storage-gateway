package gateway

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"math/rand"
	"os"
)

// New initializes a Gateway using the Config.
func New(cfg Config) (*Gateway, error) {
	const defaultBucket = "store"

	c := cfg
	if c.DefaultBucket == "" {
		c.DefaultBucket = defaultBucket
	}

	o := &Gateway{
		cfg: &c,
		logger: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelDebug,
		})),
	}

	if c.Logger != nil {
		o.logger = c.Logger
	}
	o.logger = o.logger.WithGroup("gw")

	return o, nil
}

type Gateway struct {
	cfg *Config

	logger *slog.Logger
}

// Read reads the object given its ID.
func (s *Gateway) Read(ctx context.Context, id string) (io.ReadCloser, bool, error) {
	instances, err := s.cfg.StorageInstancesFinder.Find(ctx, s.cfg.StorageInstancesSelector)
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

		s.logger.Debug("reading",
			slog.String("operation", "read"),
			slog.String("instanceID", instanceID),
			slog.String("objectID", id),
		)

		dataReadCloser, found, err := conn.Read(ctx, s.cfg.DefaultBucket, id)
		if err != nil {
			return nil, false, err
		}

		if found {
			s.logger.Debug("found",
				slog.String("operation", "read"),
				slog.String("instanceID", instanceID),
				slog.String("objectID", id),
			)
			return dataReadCloser, found, err
		}
	}

	return nil, false, nil
}

// Write writes object to the storage.
func (s *Gateway) Write(ctx context.Context, id string, reader io.Reader) error {
	instances, err := s.cfg.StorageInstancesFinder.Find(ctx, s.cfg.StorageInstancesSelector)
	if err != nil {
		return err
	}

	if len(instances) == 0 {
		return errors.New("cannot identify storage instances, check if cluster is running")
	}

	// go round-robin over all hosts to find if the object is stored to one of storage nodes
	// it's required to ensure the "sticky"-condition: overwrite already existing object
	for instanceID := range instances {
		s.logger.Debug("searching",
			slog.String("operation", "write"),
			slog.String("instanceID", instanceID),
			slog.String("objectID", id),
		)

		var conn StorageController
		conn, err = s.newStorageInstanceConnection(ctx, instanceID)
		if err != nil {
			return err
		}

		var found bool
		found, err = conn.Detected(ctx, s.cfg.DefaultBucket, id)
		if err != nil {
			return err
		}

		if found {
			s.logger.Debug("overwriting",
				slog.String("operation", "write"),
				slog.String("instanceID", instanceID),
				slog.String("objectID", id),
			)

			return conn.Write(ctx, s.cfg.DefaultBucket, id, reader)
		}
	}

	// define the instance to store new object
	instanceID := pickStorageInstance(instances, id)

	conn, err := s.newStorageInstanceConnection(ctx, instanceID)
	if err != nil {
		return err
	}

	s.logger.Debug("creating",
		slog.String("operation", "write"),
		slog.String("instanceID", instanceID),
		slog.String("objectID", id),
	)

	return conn.Write(ctx, s.cfg.DefaultBucket, id, reader)
}

func (s *Gateway) newStorageInstanceConnection(ctx context.Context, id string) (StorageController, error) {
	ipAddress, accessKeyID, secretAccessKey, err := s.cfg.StorageConnectionDetailsReader.Read(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.cfg.NewStorageConnectionFn(ipAddress, accessKeyID, secretAccessKey)
}

// pickStorageInstance randomly selects the storage instance.
func pickStorageInstance(storageInstanceIDs map[string]struct{}, _ string) (id string) {
	pick := rand.Intn(len(storageInstanceIDs)) //nolint:gosec // skip for now
	var i int
	for id = range storageInstanceIDs {
		if i == pick {
			return
		}
		i++
	}
	return
}
