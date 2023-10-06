package gateway

import (
	"golang.org/x/net/context"
)

// DNSReader defines the interface to resolve node's host by its name pattern.
type DNSReader interface {
	ListHostNames(ctx context.Context, prefix string) ([]string, error)
	ReadHostIP(ctx context.Context, hostName string) (string, error)
}

// CredentialsReader defines the interface to read.
type CredentialsReader interface {
	ReadCredentials(ctx context.Context, hostName string) (accessKeyID string, secretAccessKey string, err error)
}
