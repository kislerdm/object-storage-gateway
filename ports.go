package gateway

import (
	"io"

	"golang.org/x/net/context"
)

// DNSReader defines the interface to resolve node's host by its name pattern.
type DNSReader interface {
	ListHostNames(ctx context.Context, prefix string) ([]string, error)
	ReadHostIP(ctx context.Context, hostName string) (string, error)
}

// CredentialsReader defines the interface to read
type CredentialsReader interface {
	ReadCredentials(ctx context.Context, hostName string) (accessKeyID string, secretAccessKey string, err error)
}

// ReadWriteScanner defines the interface to interact with the storage.
type ReadWriteScanner interface {
	Reader
	Writer
	Scanner
}

// Reader defines the interface to retrieve data from the storage instance.
type Reader interface {
	Read(ctx context.Context, id string) (data io.ReadCloser, err error)
}

// Writer defines the interface to store data to the storage instance.
type Writer interface {
	Write(ctx context.Context, id string, data io.ReadCloser) error
}

// Scanner defines the interface to look up the object in the storage instance.
type Scanner interface {
	ObjectExists(ctx context.Context, id string) (bool, error)
}
