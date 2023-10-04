package main

import (
	"io"
	"log/slog"
	"net/http"
	"strings"
)

type HTTPHandler struct {
	io           ReadWriter
	commonPrefix string
	logger       slog.Logger
}

func (h HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !h.knownRoute(r.URL.Path) {
		writeErrorMessage(w, http.StatusBadRequest, "route cannot be handled")
		return
	}

	objectID := h.readObjectID(r.URL.Path)
	if err := ValidateObjectID(objectID); err != nil {
		writeErrorMessage(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	switch r.Method {
	case http.MethodGet:
		data, err := h.io.Read(r.Context(), objectID)
		defer func() { _ = data.Close() }()

		if IsNotFoundError(err) {
			h.logger.Error("READING DATA", "error", err)
			writeErrorMessage(w, http.StatusNotFound, "object not found")
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/octet-stream")

		if _, err := io.Copy(w, data); err != nil {
			h.logger.Error("WRITING OUTPUT", "error", err)
			writeErrorMessage(w, http.StatusInternalServerError, "server error")
		}

		return

	case http.MethodPut:
		defer func() { _ = r.Body.Close() }()
		err := h.io.Write(r.Context(), objectID, r.Body)
		if err != nil {
			h.logger.Error("WRITING DATA", "error", err)
			writeErrorMessage(w, http.StatusInternalServerError, "server error")
			return
		}

	default:
		writeErrorMessage(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
}

func (h HTTPHandler) knownRoute(p string) bool {
	return strings.HasPrefix(h.commonPrefix, p)
}

func (h HTTPHandler) readObjectID(p string) string {
	s := strings.TrimPrefix(p, h.commonPrefix)

	// removes all trailing slash
	s = strings.TrimRightFunc(s, func(r rune) bool {
		return r == '/'
	})

	s = strings.TrimRightFunc(s, func(r rune) bool {
		return r == '/'
	})

	return s
}

func writeErrorMessage(w http.ResponseWriter, statusCode int, s string) {
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"error":"` + s + `"}`))
}
