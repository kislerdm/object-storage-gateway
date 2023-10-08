//go:build !unittest
// +build !unittest

package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/kislerdm/minio-gateway/internal/docker"
	"github.com/kislerdm/minio-gateway/internal/minio"
	"github.com/kislerdm/minio-gateway/pkg/gateway"
	"github.com/kislerdm/minio-gateway/pkg/restfulhandler"
)

func main() {
	port := "3000"
	if v, err := strconv.Atoi(os.Getenv("PORT")); err == nil && v > 1000 {
		port = strconv.Itoa(v)
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

	cl, err := docker.NewClient()
	if err != nil {
		log.Fatalln(err)
	}

	gwConfig := gateway.Config{
		StorageInstancesSelector:       storageInstanceSelector,
		DefaultBucket:                  "store",
		StorageInstancesFinder:         cl,
		StorageConnectionDetailsReader: cl,
		NewStorageConnectionFn:         minio.NewClient,
		Logger: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: true,
			Level:     loggerLevel,
		})),
	}

	gw, err := restfulhandler.FromConfig(gwConfig)
	if err != nil {
		log.Fatalln(err)
	}

	server := &http.Server{
		Addr:         ":" + port,
		ReadTimeout:  -1,
		WriteTimeout: -1,
		Handler:      gw,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalln(err)
	}
}
