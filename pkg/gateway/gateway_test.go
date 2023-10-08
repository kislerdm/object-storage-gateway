package gateway

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"strings"
	"testing"
)

const mockClusterPrefix = "myhost"

func TestGateway_Read(t *testing.T) {
	t.Parallel()

	const inputID = "obj"

	t.Run("shall successfully return existing object", func(t *testing.T) {
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

	t.Run("shall fail to establish connection to the node", func(t *testing.T) {
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

	t.Run(`shall successfully return the status "not found"`, func(t *testing.T) {
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

	t.Run("shall fail to read the object", func(t *testing.T) {
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
	const inputID = "obj"
	inputData := strings.NewReader("data")

	t.Parallel()
	t.Run("shall successfully overwrite existing object", func(t *testing.T) {
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
	t.Run("shall successfully create the object", func(t *testing.T) {
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

func (m *mockStorageClient) Detected(_ context.Context, _, _ string) (bool, error) {
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
	if m.err != nil {
		return "", "", "", m.err
	}
	return "192.0.2.10", "foo", "bar", nil
}

func newMockGateway() *Gateway {
	return &Gateway{
		cfg: &Config{
			StorageInstancesSelector:       mockClusterPrefix,
			StorageInstancesFinder:         &mockStorageInstancesFinder{},
			StorageConnectionDetailsReader: &mockStorageConnectionDetailsReader{},
			NewStorageConnectionFn:         mockMinioConnectionFactory(errors.New("undefined"), nil),
		},
		logger: slog.Default(),
	}
}

func Test_readSortedMapKeys(t *testing.T) {
	type args struct {
		m map[string]struct{}
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "1 element",
			args: args{
				m: map[string]struct{}{"foo": {}},
			},
			want: []string{"foo"},
		},
		{
			name: "3 elements",
			args: args{
				m: map[string]struct{}{"foo": {}, "baz": {}, "bar": {}},
			},
			want: []string{"bar", "baz", "foo"},
		},
		{
			name: "empty",
			args: args{
				m: map[string]struct{}{},
			},
			want: []string{},
		},
		{
			name: "nil input",
			args: args{
				m: nil,
			},
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := readSortedMapKeys(tt.args.m); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("readMapKeys() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_hash(t *testing.T) {
	type args struct {
		id string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "1",
			args: args{
				id: "1",
			},
			want: 49,
		},
		{
			name: "foo",
			args: args{
				id: "foo",
			},
			want: 324,
		},
		{
			name: "FOo0",
			args: args{
				id: "FOo0",
			},
			want: 308,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hash(tt.args.id); got != tt.want {
				t.Errorf("hash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_pickStorageInstance(t *testing.T) {
	type args struct {
		storageInstanceIDs map[string]struct{}
		objectID           string
	}
	tests := []struct {
		name   string
		args   args
		wantId string
	}{
		{
			name: "nil input",
			args: args{
				storageInstanceIDs: nil,
			},
			wantId: "",
		},
		{
			name: "empty input",
			args: args{
				storageInstanceIDs: map[string]struct{}{},
			},
			wantId: "",
		},
		{
			name: "single instance",
			args: args{
				storageInstanceIDs: map[string]struct{}{"foo": {}},
			},
			wantId: "foo",
		},
		{
			name: `three instances - obj:"1"`,
			args: args{
				storageInstanceIDs: map[string]struct{}{"foo": {}, "bar": {}, "baz": {}},
				objectID:           "1",
			},
			// 49 % 3 = 1
			wantId: "baz",
		},
		{
			name: "three instances - obj:foo",
			args: args{
				storageInstanceIDs: map[string]struct{}{"foo": {}, "bar": {}, "baz": {}},
				objectID:           "foo",
			},
			// 324 % 3 = 0
			wantId: "bar",
		},
		{
			name: "three instances - obj:FoO0",
			args: args{
				storageInstanceIDs: map[string]struct{}{"foo": {}, "bar": {}, "baz": {}},
				objectID:           "FoO0",
			},
			// 308 % 3 = 2
			wantId: "foo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotId := pickStorageInstance(tt.args.storageInstanceIDs, tt.args.objectID); gotId != tt.wantId {
				t.Errorf("pickStorageInstance() = %v, want %v", gotId, tt.wantId)
			}
		})
	}
}
