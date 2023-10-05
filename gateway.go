package gateway

import (
	"context"
	"io"
	"math/rand"
)

func NewGateway(storageHostPrefix string) (*Gateway, error) {
	panic("todo")
}

type StorageConnectionFactory func(endpoint, accessKeyID, secretAccessKey string) (ReadWriteScanner, error)

type Gateway struct {
	StorageHostPrefix string

	dnsReader                DNSReader
	credentialsReader        CredentialsReader
	storageConnectionFactory StorageConnectionFactory

	// TODO: add invalidation logic. Imagine that a node where an object was written is down.
	//  Next time we try to write to it, the error will be returned,
	//    however we can just write to a another node instead.
	cacheObjectLocation map[string]string
}

func (s *Gateway) newStorageConnection(ctx context.Context, hostName string) (ReadWriteScanner, error) {
	host, err := s.dnsReader.ReadHostIP(ctx, hostName)
	if err != nil {
		return nil, err
	}
	accessKeyID, secretAccessKey, err := s.credentialsReader.ReadCredentials(ctx, hostName)
	if err != nil {
		return nil, err
	}
	return s.storageConnectionFactory(host, accessKeyID, secretAccessKey)
}

func (s *Gateway) Read(ctx context.Context, id string) (data io.ReadCloser, err error) {
	hostName, ok := s.cacheObjectLocation[id]
	if ok {
		return s.read(ctx, id, hostName)
	}

	hosts, err := s.dnsReader.ListHostNames(ctx, s.StorageHostPrefix)
	if err != nil {
		return nil, err
	}

	// go round-robin over all hosts
	// TODO: improve performance: cache connections (?)
	var o io.ReadCloser
	for _, host := range hosts {
		o, err = s.read(ctx, id, hostName)
		if o != nil {
			s.setObjectLocation(id, host)
			return o, nil
		}
		if !IsNotFoundError(err) {
			return nil, err
		}
	}

	return nil, err
}

func (s *Gateway) read(ctx context.Context, id string, hostName string) (io.ReadCloser, error) {
	conn, err := s.newStorageConnection(ctx, hostName)
	if err != nil {
		return nil, err
	}
	return conn.Read(ctx, id)
}

func (s *Gateway) Write(ctx context.Context, id string, data io.ReadCloser) error {
	hostName, ok := s.cacheObjectLocation[id]
	if ok {
		return s.write(ctx, id, data, hostName)
	}

	hosts, err := s.dnsReader.ListHostNames(ctx, s.StorageHostPrefix)
	if err != nil {
		return err
	}

	// go round-robin over all hosts to find if the object is stored to one of storage nodes
	// it's required to ensure the "sticky"-condition: overwrite already existing object
	for _, host := range hosts {
		found, err, con := s.objectExists(ctx, id, host)
		if err != nil {
			return err
		}
		if found {
			return con.Write(ctx, id, data)
		}
	}

	// define host to store new object
	host := pickStorage(hosts, id)

	return s.write(ctx, id, data, host)
}

func pickStorage(hosts []string, _ string) string {
	return hosts[rand.Intn(len(hosts))]
}

func (s *Gateway) setObjectLocation(id string, host string) {
	if s.cacheObjectLocation == nil {
		s.cacheObjectLocation = map[string]string{}
	}
	s.cacheObjectLocation[id] = host
}

func (s *Gateway) write(ctx context.Context, id string, data io.ReadCloser, hostName string) error {
	conn, err := s.newStorageConnection(ctx, hostName)
	if err != nil {
		return err
	}
	return conn.Write(ctx, id, data)
}

func (s *Gateway) objectExists(ctx context.Context, id string, hostName string) (bool, error, ReadWriteScanner) {
	conn, err := s.newStorageConnection(ctx, hostName)
	if err != nil {
		return false, err, nil
	}
	exists, err := conn.ObjectExists(ctx, id)
	if err != nil {
		return false, err, nil
	}
	return exists, nil, conn
}
