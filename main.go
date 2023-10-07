//go:build !unittest
// +build !unittest

package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/kislerdm/minio-gateway/internal/docker"
	"github.com/kislerdm/minio-gateway/internal/minio"
	"github.com/kislerdm/minio-gateway/pkg/gateway"
	"github.com/kislerdm/minio-gateway/pkg/gateway/restfulhandler"
)

func main() {
	port := "3000"
	if v, err := strconv.Atoi(os.Getenv("PORT")); err == nil && v > 1000 {
		port = strconv.Itoa(v)
	}

	storagePrefix := "amazin-object-storage-node"
	if v := os.Getenv("STORAGE_INSTANCES_PREFIX"); v != "" {
		storagePrefix = v
	}

	cl, err := docker.NewClient()
	if err != nil {
		log.Fatalln(err)
	}

	gwConfig := gateway.Config{
		StorageInstancesPrefix:         storagePrefix,
		DefaultBucket:                  "store",
		StorageInstancesFinder:         cl,
		StorageConnectionDetailsReader: cl,
		NewStorageConnectionFn:         minio.NewClient,
	}

	gw, err := restfulhandler.FromConfig(gwConfig)
	if err != nil {
		log.Fatalln(err)
	}

	if err := http.ListenAndServe(":"+port, gw); err != nil {
		log.Fatalln(err)
	}
}
