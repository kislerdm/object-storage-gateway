package gateway

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"

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
		gateway.newConnectionFn = mockMinioConnectionFactory(nil, &mockMinioClient{Data: storedDataReader})

		// WHEN
		got, err := gateway.Read(context.TODO(), inputID)

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
		gateway.newConnectionFn = mockMinioConnectionFactory(nil, &mockMinioClient{Data: storedDataReader})

		// WHEN
		got, err := gateway.Read(context.TODO(), inputID)

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
		gateway.newConnectionFn = mockMinioConnectionFactory(errors.New("error"), nil)

		// WHEN
		_, err := gateway.Read(context.TODO(), inputID)

		// THEN
		if err == nil {
			t.Errorf("error is expected")
			return
		}
	})

	t.Run("shall fail to get a connection to the node the object's location is not known", func(t *testing.T) {
		// GIVEN
		gateway := newMockGateway()
		gateway.newConnectionFn = mockMinioConnectionFactory(errors.New("error"), nil)

		// WHEN
		_, err := gateway.Read(context.TODO(), inputID)

		// THEN
		if err == nil {
			t.Errorf("error is expected")
			return
		}
	})

	t.Run(`shall return "not found" error`, func(t *testing.T) {
		// GIVEN
		gateway := newMockGateway()
		gateway.newConnectionFn = mockMinioConnectionFactory(nil, &mockMinioClient{Err: minio.ErrorResponse{StatusCode: http.StatusNotFound}})

		// WHEN
		_, err := gateway.Read(context.TODO(), inputID)

		// THEN
		if err == nil {
			t.Errorf("error is expected")
			return
		}
		if err.(minio.ErrorResponse).StatusCode != http.StatusNotFound {
			t.Errorf("not found error expected")
			return
		}
	})

	t.Run(`shall fail to read object`, func(t *testing.T) {
		// GIVEN
		gateway := newMockGateway()
		gateway.newConnectionFn = mockMinioConnectionFactory(nil, &mockMinioClient{Err: errors.New("foo")})

		// WHEN
		_, err := gateway.Read(context.TODO(), inputID)

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

type mockDNSReader struct {
	HostCnt          uint
	CntListHostNames uint
	Err              error
	cache            map[string]string
}

func (m *mockDNSReader) ListHostNames(_ context.Context, prefix string) ([]string, error) {
	m.CntListHostNames++
	if m.Err != nil {
		return nil, m.Err
	}
	l := m.HostCnt
	if l == 0 {
		l = 1
	}
	var o = make([]string, l)
	m.cache = map[string]string{}
	for i := range o {
		id := strconv.Itoa(i)
		o[i] = prefix + "-" + id
		m.cache[o[i]] = "192.0.2." + id
	}
	return o, nil
}

func (m *mockDNSReader) ReadHostIP(_ context.Context, hostName string) (string, error) {
	if m.Err != nil {
		return "", m.Err
	}

	const defaultIP = "192.0.2.0"
	if m.cache == nil {
		return defaultIP, nil
	}

	ip, ok := m.cache[hostName]
	if !ok {
		return defaultIP, nil
	}
	return ip, nil
}

type mockCredentialsReader struct {
	Err                          error
	AccessKeyID, SecretAccessKey string

	cntHit uint
}

func (m *mockCredentialsReader) ReadCredentials(_ context.Context, _ string) (string, string, error) {
	m.cntHit++
	if m.Err != nil {
		return "", "", m.Err
	}
	return m.AccessKeyID, m.SecretAccessKey, nil
}

func mockMinioConnectionFactory(err error, rw minioConnectionPort) minioConnectionFactory {
	return func(endpoint, accessKeyID, secretAccessKey string) (minioConnectionPort, error) {
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

func (m *mockMinioClient) GetObject(ctx context.Context, _, _ string, _ minio.GetObjectOptions) (io.ReadCloser, error) {
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

func (m *mockMinioClient) PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, _ int64, _ minio.PutObjectOptions) (minio.UploadInfo, error) {
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

func newMockGateway() Client {
	return Client{
		StorageHostPrefix: "myhost",
		dnsReader: &mockDNSReader{
			HostCnt: 3,
		},
		credentialsReader: &mockCredentialsReader{
			AccessKeyID:     "foo",
			SecretAccessKey: "bar",
		},
		newConnectionFn:     mockMinioConnectionFactory(errors.New("undefined"), nil),
		cacheObjectLocation: map[string]string{},
	}
}
