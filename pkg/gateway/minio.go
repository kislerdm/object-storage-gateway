package gateway

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func newMinioAdapter(ipAddress, accessKeyID, secretAccessKey string) (minioPort, error) {
	const defaultPort = "9000"
	host := ipAddress + ":" + defaultPort
	c, err := minio.New(host, &minio.Options{
		Creds: credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
	})
	if err != nil {
		return nil, err
	}
	if !c.IsOnline() {
		return nil, errors.New("the storage node is offline")
	}
	return &minioAdapter{c}, nil
}

// isNotFoundError defines if the Minion client's error indicated that the obj is not found.
func isNotFoundError(err error) bool {
	switch e := err.(type) {
	case minio.ErrorResponse:
		return e.StatusCode == http.StatusNotFound
	default:
		return false
	}
}

type minioAdapter struct {
	*minio.Client
}

func (m minioAdapter) ReadObject(
	ctx context.Context, bucketName string, objectName string, opts minio.GetObjectOptions,
) (io.ReadCloser, error) {
	return m.GetObject(ctx, bucketName, objectName, opts)
}
