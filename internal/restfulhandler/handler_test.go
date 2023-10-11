package restfulhandler

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

type mockResponseWriter struct {
	Headers    http.Header
	Body       []byte
	StatusCode int
}

func (m *mockResponseWriter) Header() http.Header {
	return m.Headers
}

func (m *mockResponseWriter) Write(bytes []byte) (int, error) {
	m.Body = bytes
	return len(bytes), nil
}

func (m *mockResponseWriter) WriteHeader(statusCode int) {
	m.StatusCode = statusCode
}

type mockReadWriter struct {
	err        error
	readCloser io.Reader
}

func (m *mockReadWriter) Read(_ context.Context, _ string) (readCloser io.ReadCloser, found bool, err error) {
	if m.err != nil {
		return nil, false, m.err
	}
	return io.NopCloser(m.readCloser), m.readCloser != nil, nil
}

func (m *mockReadWriter) Write(_ context.Context, _ string, reader io.Reader, _ int64) error {
	if m.err != nil {
		return m.err
	}
	m.readCloser = reader
	return nil
}

func TestHandler_ServeHTTP(t *testing.T) {
	type fields struct {
		readWriter        readWriter
		commonRoutePrefix string
		logger            *slog.Logger
	}
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}

	const contentTypeJSON = "application/json"

	var tests = []struct {
		name            string
		fields          fields
		args            args
		wantStatusCode  int
		wantContentType string
	}{
		{
			name: "shall successfully read the object",
			fields: fields{
				readWriter: &mockReadWriter{
					readCloser: strings.NewReader("obj"),
				},
				commonRoutePrefix: defaultPrefix,
				logger:            slog.Default(),
			},
			args: args{
				w: &mockResponseWriter{Headers: map[string][]string{}},
				r: &http.Request{
					Method: http.MethodGet,
					URL:    &url.URL{Path: "/object/bAr1"},
				},
			},
			wantStatusCode:  http.StatusOK,
			wantContentType: "application/octet-stream",
		},
		{
			name: "shall successfully write the object",
			fields: fields{
				readWriter:        &mockReadWriter{},
				commonRoutePrefix: defaultPrefix,
				logger:            slog.Default(),
			},
			args: args{
				w: &mockResponseWriter{Headers: map[string][]string{}},
				r: &http.Request{
					Method: http.MethodPut,
					URL:    &url.URL{Path: "/object/bAr1"},
					Body:   io.NopCloser(strings.NewReader("jeanmichel")),
				},
			},
			wantStatusCode: http.StatusCreated,
		},
		{
			name: "shall successfully write the object - empty request body",
			fields: fields{
				readWriter:        &mockReadWriter{},
				commonRoutePrefix: defaultPrefix,
				logger:            slog.Default(),
			},
			args: args{
				w: &mockResponseWriter{Headers: map[string][]string{}},
				r: &http.Request{
					Method: http.MethodPut,
					URL:    &url.URL{Path: "/object/bAr1"},
					Body:   io.NopCloser(nil),
				},
			},
			wantStatusCode: http.StatusCreated,
		},
		{
			name: "shall fail to read the object - object not found",
			fields: fields{
				readWriter:        &mockReadWriter{},
				commonRoutePrefix: defaultPrefix,
				logger:            slog.Default(),
			},
			args: args{
				w: &mockResponseWriter{Headers: map[string][]string{}},
				r: &http.Request{
					Method: http.MethodGet,
					URL:    &url.URL{Path: "/object/bAr1"},
				},
			},
			wantStatusCode:  http.StatusNotFound,
			wantContentType: contentTypeJSON,
		},
		{
			name: "shall fail to read the object - storage error",
			fields: fields{
				readWriter:        &mockReadWriter{err: errors.New("error")},
				commonRoutePrefix: defaultPrefix,
				logger:            slog.Default(),
			},
			args: args{
				w: &mockResponseWriter{Headers: map[string][]string{}},
				r: &http.Request{
					Method: http.MethodGet,
					URL:    &url.URL{Path: "/object/bAr1"},
				},
			},
			wantStatusCode:  http.StatusInternalServerError,
			wantContentType: contentTypeJSON,
		},
		{
			name: "shall fail to write the object - storage error",
			fields: fields{
				readWriter:        &mockReadWriter{err: errors.New("error")},
				commonRoutePrefix: defaultPrefix,
				logger:            slog.Default(),
			},
			args: args{
				w: &mockResponseWriter{Headers: map[string][]string{}},
				r: &http.Request{
					Method: http.MethodPut,
					URL:    &url.URL{Path: "/object/bAr1"},
					Body:   io.NopCloser(strings.NewReader("data-to-write")),
				},
			},
			wantStatusCode:  http.StatusInternalServerError,
			wantContentType: contentTypeJSON,
		},
		{
			name: "shall fail to write the object - nil body",
			fields: fields{
				readWriter:        &mockReadWriter{},
				commonRoutePrefix: defaultPrefix,
				logger:            slog.Default(),
			},
			args: args{
				w: &mockResponseWriter{Headers: map[string][]string{}},
				r: &http.Request{
					Method: http.MethodPut,
					URL:    &url.URL{Path: "/object/bAr1"},
				},
			},
			wantStatusCode:  http.StatusBadRequest,
			wantContentType: contentTypeJSON,
		},
		{
			name: "shall fail - the route is unknown",
			fields: fields{
				commonRoutePrefix: defaultPrefix,
				logger:            slog.Default(),
			},
			args: args{
				w: &mockResponseWriter{Headers: map[string][]string{}},
				r: &http.Request{
					Method: http.MethodGet,
					URL:    &url.URL{Path: "/foo/bar1"},
				},
			},
			wantStatusCode:  http.StatusBadRequest,
			wantContentType: contentTypeJSON,
		},
		{
			name: "shall fail - unsupported object id",
			fields: fields{
				commonRoutePrefix: defaultPrefix,
				logger:            slog.Default(),
			},
			args: args{
				w: &mockResponseWriter{Headers: map[string][]string{}},
				r: &http.Request{
					Method: http.MethodGet,
					URL:    &url.URL{Path: "/object////-!#///"},
				},
			},
			wantStatusCode:  http.StatusUnprocessableEntity,
			wantContentType: contentTypeJSON,
		},
		{
			name: "shall fail - unsupported method",
			fields: fields{
				commonRoutePrefix: defaultPrefix,
				logger:            slog.Default(),
			},
			args: args{
				w: &mockResponseWriter{Headers: map[string][]string{}},
				r: &http.Request{
					Method: http.MethodPatch,
					URL:    &url.URL{Path: "/object/fOo0"},
				},
			},
			wantStatusCode:  http.StatusMethodNotAllowed,
			wantContentType: contentTypeJSON,
		},
	}

	t.Parallel()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := Handler{
				rw:                tt.fields.readWriter,
				commonRoutePrefix: tt.fields.commonRoutePrefix,
				logger:            tt.fields.logger,
			}
			h.ServeHTTP(tt.args.w, tt.args.r)

			responseWriter := tt.args.w.(*mockResponseWriter)
			if responseWriter.StatusCode != tt.wantStatusCode {
				t.Errorf("wrong StatuCode, want: %d, got: %d", tt.wantStatusCode, responseWriter.StatusCode)
				return
			}

			gotContentType := responseWriter.Headers.Get("Content-Type")
			if gotContentType != tt.wantContentType {
				t.Errorf("wrong Content-Type header, want: %s, got: %s", tt.wantContentType, gotContentType)
				return
			}
		})
	}
}
