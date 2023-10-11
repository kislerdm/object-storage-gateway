package restfulhandler

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/kislerdm/object-storage-gateway/pkg/gateway"
)

const defaultPrefix = "/object"

// New initialises new Gateway Restful API handler.
func New(gw *gateway.Gateway) (*Handler, error) {
	o := &Handler{
		rw:                gw,
		commonRoutePrefix: defaultPrefix,
		logger: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: false,
			Level:     slog.LevelError,
		})),
	}

	if gw.Logger != nil {
		o.logger = gw.Logger
	}
	o.logger = o.logger.WithGroup("webserver")

	return o, nil
}

// Handler Gateway Restful API handler.
type Handler struct {
	rw readWriter

	commonRoutePrefix string
	logger            *slog.Logger
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("request",
		slog.String("path", r.URL.Path),
		slog.String("method", r.Method),
		slog.Int64("content-length", r.ContentLength),
		slog.String("headers", concatHeaders(r.Header)),
	)

	if !h.knownRoute(r.URL.Path) {
		h.logError(r, http.StatusBadRequest, "route not found")
		writeErrorMessage(w, http.StatusBadRequest, "route cannot be handled")
		return
	}

	objectID := h.readObjectID(r.URL.Path)
	if err := validateInputObjectID(objectID); err != nil {
		h.logError(r, http.StatusUnprocessableEntity, err.Error())
		writeErrorMessage(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	switch r.Method {
	case http.MethodGet:
		readCloser, found, err := h.rw.Read(r.Context(), objectID)
		if err != nil {
			h.logError(r, http.StatusInternalServerError, err.Error())
			writeErrorMessage(w, http.StatusInternalServerError, "failed to read object")
			return
		}

		if !found || readCloser == nil {
			h.logError(r, http.StatusNotFound, "object not found")
			writeErrorMessage(w, http.StatusNotFound, "object not found")
			return
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		defer func() { _ = readCloser.Close() }()
		if _, err := io.Copy(w, readCloser); err != nil {
			h.logError(r, http.StatusInternalServerError, err.Error())
			writeErrorMessage(w, http.StatusInternalServerError, "server error")
		}

		return

	case http.MethodPut:
		if r.Body == nil {
			h.logError(r, http.StatusBadRequest, "nil request body")
			writeErrorMessage(w, http.StatusBadRequest, "failed to write: request body shall be provided")
			return
		}

		defer func() { _ = r.Body.Close() }()
		if err := h.rw.Write(r.Context(), objectID, r.Body, contentSize(r)); err != nil {
			h.logError(r, http.StatusInternalServerError, err.Error())
			writeErrorMessage(w, http.StatusInternalServerError, "failed to write object")
			return
		}

		w.WriteHeader(http.StatusCreated)
		return

	default:
		h.logError(r, http.StatusInternalServerError, "method not allowed")
		writeErrorMessage(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
}

func contentSize(r *http.Request) int64 {
	v, err := strconv.ParseInt(r.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return -1
	}
	return v
}

func concatHeaders(headers http.Header) string {
	if len(headers) == 0 {
		return ""
	}
	var buf strings.Builder
	for k := range headers {
		for _, v := range headers.Values(k) {
			buf.WriteString(k)
			buf.WriteString("=")
			buf.WriteString(v)
			buf.WriteString(",")
		}
	}
	o := buf.String()
	return o[:len(o)-1]
}

func (h Handler) logError(r *http.Request, statusCode int, msg string) {
	h.logger.Error(msg,
		slog.Int("code", statusCode),
		slog.String("path", r.URL.Path),
		slog.String("method", r.Method),
	)
}

func (h Handler) knownRoute(p string) bool {
	return strings.HasPrefix(p, h.commonRoutePrefix)
}

func (h Handler) readObjectID(p string) string {
	s := strings.TrimPrefix(p, h.commonRoutePrefix)

	// removes all trailing slash
	s = strings.TrimRightFunc(
		s, func(r rune) bool {
			return r == '/'
		},
	)

	s = strings.TrimLeftFunc(
		s, func(r rune) bool {
			return r == '/'
		},
	)

	return s
}

func writeErrorMessage(w http.ResponseWriter, statusCode int, s string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, _ = w.Write([]byte(`{"error":"` + s + `"}`))
}

// reader defines the interface to store and retrieve data.
type readWriter interface {
	Read(ctx context.Context, id string) (readCloser io.ReadCloser, found bool, err error)
	Write(ctx context.Context, id string, reader io.Reader, objectSizeBytes int64) error
}
