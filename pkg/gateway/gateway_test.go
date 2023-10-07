package gateway

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/minio/minio-go/v7"
)

func TestReadWriterStickyAssignator_Read(t *testing.T) {
	t.Parallel()

	const inputID = "obj"

	t.Run("shall return data when the node with the object is known", func(t *testing.T) {
		// GIVEN
		gateway := newMockGateway()
		gateway.cacheObjectLocation[inputID] = "myhost-0"
		storedDataReader := strings.NewReader("foo")
		gateway.connectionFactory = mockMinioConnectionFactory(nil, &mockMinioClient{Data: storedDataReader})

		// WHEN
		got, _, err := gateway.Read(context.TODO(), inputID)

		// THEN
		if err != nil {
			t.Errorf("no error expected")
			return
		}

		want := io.NopCloser(storedDataReader)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("unexpected data output want: %#v, got: %#v", want, got)
			return
		}
	})

	t.Run("shall return data when the node with the object is not known", func(t *testing.T) {
		// GIVEN

		storedDataReader := strings.NewReader("qux")
		gateway := newMockGateway()
		gateway.connectionFactory = mockMinioConnectionFactory(nil, &mockMinioClient{Data: storedDataReader})

		// WHEN
		got, _, err := gateway.Read(context.TODO(), inputID)

		want := io.NopCloser(storedDataReader)
		// THEN
		if err != nil {
			t.Errorf("no error expected")
			return
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("unexpected data output want: %#v, got: %#v", want, got)
			return
		}
	})

	t.Run("shall fail to get a connection to the node with the object when its known", func(t *testing.T) {
		// GIVEN

		gateway := newMockGateway()
		gateway.cacheObjectLocation[inputID] = "myhost-0"
		gateway.connectionFactory = mockMinioConnectionFactory(errors.New("error"), nil)

		// WHEN
		_, _, err := gateway.Read(context.TODO(), inputID)

		// THEN
		if err == nil {
			t.Errorf("error is expected")
			return
		}
	})

	t.Run("shall fail to get a connection to the node the object's location is not known", func(t *testing.T) {
		// GIVEN
		gateway := newMockGateway()
		gateway.connectionFactory = mockMinioConnectionFactory(errors.New("error"), nil)

		// WHEN
		_, _, err := gateway.Read(context.TODO(), inputID)

		// THEN
		if err == nil {
			t.Errorf("error is expected")
			return
		}
	})

	t.Run(`shall succeed without error, but return obj not exists`, func(t *testing.T) {
		// GIVEN
		gateway := newMockGateway()
		gateway.connectionFactory = mockMinioConnectionFactory(nil,
			&mockMinioClient{Err: minio.ErrorResponse{StatusCode: http.StatusNotFound}})

		// WHEN
		_, exists, err := gateway.Read(context.TODO(), inputID)

		// THEN
		if err != nil {
			t.Errorf("error is not expected")
			return
		}
		if exists {
			t.Errorf("object is not expected to be found")
			return
		}
	})

	t.Run(`shall fail to read object`, func(t *testing.T) {
		// GIVEN
		gateway := newMockGateway()
		gateway.newConnectionFn = mockMinioConnectionFactory(nil, &mockMinioClient{Err: errors.New("foo")})

		// WHEN
		_, _, err := gateway.Read(context.TODO(), inputID)

		// THEN
		if err == nil || err.Error() != "foo" {
			t.Errorf("error expected")
			return
		}
	})
}

func TestReadWriterStickyAssignator_Write(t *testing.T) {
	t.Parallel()

	const inputID = "obj"
	inputData := strings.NewReader("data")

	t.Run("shall overwrite existing object when its location was known already", func(t *testing.T) {
		// GIVEN
		gateway := newMockGateway()
		gateway.cacheObjectLocation[inputID] = "myhost-0"
		gateway.newConnectionFn = mockMinioConnectionFactory(nil, &mockMinioClient{})

		// WHEN
		err := gateway.Write(context.TODO(), inputID, inputData)

		// THEN
		if err != nil {
			t.Errorf("no error expected")
			return
		}
	})
	t.Run("shall overwrite existing object when its location was not known", func(t *testing.T) {
		// GIVEN
		gateway := newMockGateway()
		gateway.newConnectionFn = mockMinioConnectionFactory(nil, &mockMinioClient{Data: inputData})

		// WHEN
		err := gateway.Write(context.TODO(), inputID, inputData)

		// THEN
		if err != nil {
			t.Errorf("no error expected")
			return
		}
	})
	t.Run("shall write the object which is not present in the storage cluster yet", func(t *testing.T) {
		// GIVEN
		gateway := newMockGateway()
		gateway.newConnectionFn = mockMinioConnectionFactory(nil, &mockMinioClient{})

		// WHEN
		err := gateway.Write(context.TODO(), inputID, inputData)

		// THEN
		if err != nil {
			t.Errorf("no error expected")
			return
		}
	})

}

func mockMinioConnectionFactory(err error, rw minioPort) StorageConnectionFactory {
	return func(endpoint, accessKeyID, secretAccessKey string) (minioPort, error) {
		if err != nil {
			return nil, err
		}
		return rw, nil
	}
}

type mockMinioClient struct {
	Err  error
	Data io.Reader
}

func (m *mockMinioClient) GetObjectACL(_ context.Context, _, objectName string) (*minio.ObjectInfo, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return &minio.ObjectInfo{
		Key: objectName,
	}, nil
}

func (m *mockMinioClient) ReadObject(ctx context.Context, _, _ string, _ minio.GetObjectOptions) (
	io.ReadCloser,
	error,
) {
	if m.Err != nil {
		return nil, m.Err
	}
	return io.NopCloser(m.Data), nil
}

func (m *mockMinioClient) BucketExists(ctx context.Context, _ string) (bool, error) {
	if m.Err != nil {
		return false, m.Err
	}
	return m.Data != nil, nil
}

func (m *mockMinioClient) PutObject(
	ctx context.Context, bucketName, objectName string, reader io.Reader, _ int64, _ minio.PutObjectOptions,
) (minio.UploadInfo, error) {
	if m.Err != nil {
		return minio.UploadInfo{}, m.Err
	}
	m.Data = reader
	return minio.UploadInfo{
		Bucket: bucketName,
		Key:    objectName,
	}, nil
}

func (m *mockMinioClient) MakeBucket(_ context.Context, _ string, _ minio.MakeBucketOptions) error {
	return m.Err
}

func (m *mockMinioClient) IsOnline() bool {
	if m.Err != nil {
		return false
	}
	return true
}

type mockDocker struct {
	err error
}

func (m mockDocker) ContainerList(_ context.Context, _ types.ContainerListOptions) ([]types.Container, error) {
	if m.err != nil {
		return nil, m.err
	}

}

func (m mockDocker) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	// TODO implement me
	panic("implement me")
}

func (m mockClusterAccessDetailsReader) Read(_ context.Context, prefix string) (
	map[string]MimioConnectionDetails, error,
) {
	if m.err != nil {
		return nil, m.err
	}
	return map[string]MimioConnectionDetails{
		prefix: {
			IPAddress:       "192.0.2.10",
			AccessKeyID:     "foo",
			SecretAccessKey: "bar",
		},
	}, nil
}

func newMockGateway() Gateway {
	return Gateway{
		StorageInstancesPrefix:  "myhost",
		connectionDetailsReader: &mockDocker{},
		connectionFactory:       mockMinioConnectionFactory(errors.New("undefined"), nil),
		cacheObjectLocation:     map[string]string{},
	}
}
