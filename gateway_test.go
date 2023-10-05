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
		assignator := newMockAssignator()
		assignator.cacheObjectLocation[inputID] = "myhost-0"

		want := io.NopCloser(strings.NewReader("qux"))
		assignator.storageConnectionFactory = mockReadWriterFactory(nil, &readWritScannerMock{Data: want})

		// WHEN
		got, err := assignator.Read(context.TODO(), inputID)

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

	t.Run("shall return data when the node with the object is not known", func(t *testing.T) {
		// GIVEN
		want := io.NopCloser(strings.NewReader("qux"))

		assignator := newMockAssignator()
		assignator.storageConnectionFactory = mockReadWriterFactory(nil, &readWritScannerMock{Data: want})

		// WHEN
		got, err := assignator.Read(context.TODO(), inputID)

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

		assignator := newMockAssignator()
		assignator.cacheObjectLocation[inputID] = "myhost-0"
		assignator.storageConnectionFactory = mockReadWriterFactory(errors.New("error"), nil)

		// WHEN
		_, err := assignator.Read(context.TODO(), inputID)

		// THEN
		if err == nil {
			t.Errorf("error is expected")
			return
		}
	})

	t.Run("shall fail to get a connection to the node the object's location is not known", func(t *testing.T) {
		// GIVEN
		assignator := newMockAssignator()
		assignator.storageConnectionFactory = mockReadWriterFactory(errors.New("error"), nil)

		// WHEN
		_, err := assignator.Read(context.TODO(), inputID)

		// THEN
		if err == nil {
			t.Errorf("error is expected")
			return
		}
	})

	t.Run(`shall return "not found" error`, func(t *testing.T) {
		// GIVEN
		assignator := newMockAssignator()
		assignator.storageConnectionFactory = mockReadWriterFactory(nil, &readWritScannerMock{Err: minio.ErrorResponse{StatusCode: http.StatusNotFound}})

		// WHEN
		_, err := assignator.Read(context.TODO(), inputID)

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
		assignator := newMockAssignator()
		assignator.storageConnectionFactory = mockReadWriterFactory(nil, &readWritScannerMock{Err: errors.New("foo")})

		// WHEN
		_, err := assignator.Read(context.TODO(), inputID)

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
	inputData := io.NopCloser(strings.NewReader("data"))

	t.Run("shall overwrite existing object when its location was known already", func(t *testing.T) {
		// GIVEN
		assignator := newMockAssignator()
		assignator.cacheObjectLocation[inputID] = "myhost-0"
		assignator.storageConnectionFactory = mockReadWriterFactory(nil, &readWritScannerMock{})

		// WHEN
		err := assignator.Write(context.TODO(), inputID, inputData)

		// THEN
		if err != nil {
			t.Errorf("no error expected")
			return
		}
	})
	t.Run("shall overwrite existing object when its location was not known", func(t *testing.T) {
		// GIVEN
		assignator := newMockAssignator()
		assignator.storageConnectionFactory = mockReadWriterFactory(nil, &readWritScannerMock{Data: inputData})

		// WHEN
		err := assignator.Write(context.TODO(), inputID, inputData)

		// THEN
		if err != nil {
			t.Errorf("no error expected")
			return
		}
	})
	t.Run("shall write the object which is not present in the storage cluster yet", func(t *testing.T) {
		// GIVEN
		assignator := newMockAssignator()
		assignator.storageConnectionFactory = mockReadWriterFactory(nil, &readWritScannerMock{})

		// WHEN
		err := assignator.Write(context.TODO(), inputID, inputData)

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

type readWritScannerMock struct {
	Err  error
	Data io.ReadCloser
}

func (r *readWritScannerMock) ObjectExists(_ context.Context, _ string) (bool, error) {
	if r.Err != nil {
		return false, r.Err
	}
	return r.Data != nil, nil
}

func (r *readWritScannerMock) Read(_ context.Context, _ string) (io.ReadCloser, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	return r.Data, nil
}

func (r *readWritScannerMock) Write(_ context.Context, _ string, data io.ReadCloser) error {
	if r.Err != nil {
		return r.Err
	}
	r.Data = data
	return nil
}

func mockReadWriterFactory(err error, rw ReadWriteScanner) StorageConnectionFactory {
	return func(endpoint, accessKeyID, secretAccessKey string) (ReadWriteScanner, error) {
		if err != nil {
			return nil, err
		}
		return rw, nil
	}
}

func newMockAssignator() Gateway {
	return Gateway{
		StorageHostPrefix: "myhost",
		dnsReader: &mockDNSReader{
			HostCnt: 3,
		},
		credentialsReader: &mockCredentialsReader{
			AccessKeyID:     "foo",
			SecretAccessKey: "bar",
		},
		storageConnectionFactory: mockReadWriterFactory(errors.New("undefined"), nil),
		cacheObjectLocation:      map[string]string{},
	}
}
