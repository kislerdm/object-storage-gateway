//go:build !unittest
// +build !unittest

package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/kislerdm/object-storage-gateway/internal/docker"
	"github.com/kislerdm/object-storage-gateway/internal/minio"
	"github.com/kislerdm/object-storage-gateway/internal/restfulhandler"
	"github.com/kislerdm/object-storage-gateway/pkg/gateway"
)

func main() {
	cl, err := docker.NewClient()
	if err != nil {
		log.Fatalln(err)
	}

	storageInstanceSelector := "amazin-object-storage-node"
	if v := os.Getenv("STORAGE_INSTANCES_SELECTOR"); v != "" {
		storageInstanceSelector = v
	}

	debug, _ := strconv.ParseBool(os.Getenv("LOG_DEBUG"))

	loggerLevel := slog.LevelError
	if debug {
		loggerLevel = slog.LevelDebug
	}

	const storageBucket = "store"

	gw, err := gateway.New(storageInstanceSelector, storageBucket, cl, cl, minio.NewClient,
		slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: true,
			Level:     loggerLevel,
		})),
	)
	if err != nil {
		log.Fatalln(err)
	}

	gwHandler, err := restfulhandler.New(gw)
	if err != nil {
		log.Fatalln(err)
	}

	server := &http.Server{
		Addr:         ":8000",
		ReadTimeout:  -1,
		WriteTimeout: -1,
		Handler:      gwHandler,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalln(err)
	}
}
