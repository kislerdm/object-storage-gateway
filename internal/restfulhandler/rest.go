package restfulhandler

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/kislerdm/minio-gateway/internal/validator"
	"github.com/kislerdm/minio-gateway/pkg/gateway"
)

const defaultPrefix = "/object"

// New initialises new Gateway Restful API handler.
func New(gateway *gateway.Gateway) *Handler {
	var defaultLogger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	return &Handler{
		readWriter:        gateway,
		commonRoutePrefix: defaultPrefix,
		logger:            defaultLogger,
	}
}

// Handler Gateway Restful API handler.
type Handler struct {
	readWriter

	commonRoutePrefix string
	logger            *slog.Logger
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !h.knownRoute(r.URL.Path) {
		h.logError(r, http.StatusBadRequest, "route not found")
		writeErrorMessage(w, http.StatusBadRequest, "route cannot be handled")
		return
	}

	objectID := h.readObjectID(r.URL.Path)
	if err := validator.ValidateInputObjectID(objectID); err != nil {
		h.logError(r, http.StatusUnprocessableEntity, err.Error())
		writeErrorMessage(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	switch r.Method {
	case http.MethodGet:
		// TODO: fix when the key does in fact exist, but the file is big
		readCloser, found, err := h.Read(r.Context(), objectID)

		// TODO: fix the error message: return 404 when storage bucket does not exist
		if err != nil {
			h.logError(r, http.StatusInternalServerError, err.Error())
			writeErrorMessage(w, http.StatusInternalServerError, "failed to read object")
			return
		}

		if !found {
			h.logError(r, http.StatusNotFound, "object not found")
			writeErrorMessage(w, http.StatusNotFound, "object not found")
			return
		}

		defer func() { _ = readCloser.Close() }()

		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		if _, err := io.Copy(w, readCloser); err != nil {
			h.logError(r, http.StatusInternalServerError, err.Error())
			writeErrorMessage(w, http.StatusInternalServerError, "server error")
		}

		return

	case http.MethodPut:
		// TODO: fix upload of big files
		if r.Body == nil {
			h.logError(r, http.StatusBadRequest, "nil request body")
			writeErrorMessage(w, http.StatusBadRequest, "failed to write: request body shall be provided")
			return
		}

		defer func() { _ = r.Body.Close() }()

		if err := h.Write(r.Context(), objectID, r.Body); err != nil {
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
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"error":"` + s + `"}`))
}

type readWriter interface {
	reader
	writer
}

// reader defines the interface to retrieve data from the storage instance.
type reader interface {
	Read(ctx context.Context, id string) (readCloser io.ReadCloser, found bool, err error)
}

// writer defines the interface to store data to the storage instance.
type writer interface {
	Write(ctx context.Context, id string, reader io.Reader) error
}
