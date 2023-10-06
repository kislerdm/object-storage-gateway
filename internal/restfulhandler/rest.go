package restfulhandler

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strings"

	gateway "github.com/kislerdm/minio-gateway"
)

const defaultPrefix = "/object"

// New initialises new Gateway Restful API handler.
func New(gateway *gateway.Client) *Handler {
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
	if err := validateInputObjectID(objectID); err != nil {
		h.logError(r, http.StatusUnprocessableEntity, err.Error())
		writeErrorMessage(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	switch r.Method {
	case http.MethodGet:

		data, found, err := h.Read(r.Context(), objectID)
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

		defer func() { _ = data.Close() }()

		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		if _, err := io.Copy(w, data); err != nil {
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

var regExpID = regexp.MustCompile("^[a-zA-Z0-9]{1,32}$")

// validateInputObjectID validates the input object ID.
func validateInputObjectID(id string) error {
	if !regExpID.MatchString(id) {
		return errors.New("id is not valid")
	}
	return nil
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
