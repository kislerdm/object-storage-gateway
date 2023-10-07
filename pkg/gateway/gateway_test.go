package gateway

import (
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
)

const mockClusterPrefix = "myhost"

func TestGateway_Read(t *testing.T) {
	t.Parallel()

	const inputID = "obj"

	t.Run("shall return data when the node with the object is known", func(t *testing.T) {
		// GIVEN
		gateway := newMockGateway()
		gateway.cacheObjectLocation[inputID] = mockClusterPrefix + "-0"
		storedDataReader := strings.NewReader("foo")
		gateway.cfg.NewStorageConnectionFn = mockMinioConnectionFactory(nil,
			&mockStorageClient{dataReader: storedDataReader})

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
		gateway.cfg.NewStorageConnectionFn = mockMinioConnectionFactory(nil,
			&mockStorageClient{dataReader: storedDataReader})

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
		gateway.cfg.NewStorageConnectionFn = mockMinioConnectionFactory(errors.New("error"), nil)

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
		gateway.cfg.NewStorageConnectionFn = mockMinioConnectionFactory(errors.New("error"), nil)

		// WHEN
		_, _, err := gateway.Read(context.TODO(), inputID)

		// THEN
		if err == nil {
			t.Errorf("error is expected")
			return
		}
	})

	t.Run(`shall succeed, but find no found`, func(t *testing.T) {
		// GIVEN
		gateway := newMockGateway()
		gateway.cfg.NewStorageConnectionFn = mockMinioConnectionFactory(nil, &mockStorageClient{})

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
		gateway.cfg.NewStorageConnectionFn = mockMinioConnectionFactory(nil,
			&mockStorageClient{err: errors.New("foo")})

		// WHEN
		_, _, err := gateway.Read(context.TODO(), inputID)

		// THEN
		if err == nil || err.Error() != "foo" {
			t.Errorf("error expected")
			return
		}
	})
}

func TestGateway_Write(t *testing.T) {
	t.Parallel()

	const inputID = "obj"
	inputData := strings.NewReader("data")

	t.Run("shall overwrite existing object when its location was known already", func(t *testing.T) {
		// GIVEN
		gateway := newMockGateway()
		gateway.cacheObjectLocation[inputID] = "myhost-0"
		gateway.cfg.NewStorageConnectionFn = mockMinioConnectionFactory(nil, &mockStorageClient{})

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
		gateway.cfg.NewStorageConnectionFn = mockMinioConnectionFactory(nil, &mockStorageClient{dataReader: inputData})

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
		gateway.cfg.NewStorageConnectionFn = mockMinioConnectionFactory(nil, &mockStorageClient{})

		// WHEN
		err := gateway.Write(context.TODO(), inputID, inputData)

		// THEN
		if err != nil {
			t.Errorf("no error expected")
			return
		}
	})

}

func mockMinioConnectionFactory(err error, rw StorageController) StorageConnectionFactory {
	return func(endpoint, accessKeyID, secretAccessKey string) (StorageController, error) {
		if err != nil {
			return nil, err
		}
		return rw, nil
	}
}

type mockStorageClient struct {
	err        error
	dataReader io.Reader
}

func (m *mockStorageClient) Read(_ context.Context, _, _ string) (io.ReadCloser, bool, error) {
	if m.err != nil {
		return nil, false, m.err
	}
	return io.NopCloser(m.dataReader), m.dataReader != nil, nil
}

func (m *mockStorageClient) Write(_ context.Context, _, _ string, reader io.Reader) error {
	if m.err != nil {
		return m.err
	}
	m.dataReader = reader
	return nil
}

func (m *mockStorageClient) Detected(ctx context.Context, bucketName, objectName string) (bool, error) {
	return m.dataReader != nil, m.err
}

type mockStorageInstancesFinder struct {
	err error
}

func (m mockStorageInstancesFinder) Find(_ context.Context, instanceNameFilter string) (map[string]struct{}, error) {
	if m.err != nil {
		return nil, m.err
	}
	return map[string]struct{}{instanceNameFilter + "-0": {}}, nil
}

type mockStorageConnectionDetailsReader struct {
	err error
}

func (m mockStorageConnectionDetailsReader) Read(_ context.Context, _ string) (
	ipAddress, accessKeyID, secretAccessKey string, err error,
) {
	if err != nil {
		return "", "", "", err
	}
	return "192.0.2.10", "foo", "bar", nil
}

func newMockGateway() *Gateway {
	return &Gateway{
		cfg: &Config{
			StorageInstancesPrefix:         mockClusterPrefix,
			StorageInstancesFinder:         &mockStorageInstancesFinder{},
			StorageConnectionDetailsReader: &mockStorageConnectionDetailsReader{},
			NewStorageConnectionFn:         mockMinioConnectionFactory(errors.New("undefined"), nil),
		},
		cacheObjectLocation: map[string]string{},
	}
}
