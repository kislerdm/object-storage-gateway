//go:build !unittest
// +build !unittest

package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/kislerdm/minio-gateway/internal/restfulhandler"
	"github.com/kislerdm/minio-gateway/pkg/gateway/config"
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

	gwConfig := config.Config{
		StorageInstancesPrefix:  storagePrefix,
		DefaultBucket:           "store",
		InstancesFinder:         nil,
		ConnectionDetailsReader: nil,
		ConnectionFactory:       nil,
	}

	if err := http.ListenAndServe(":"+port, restfulhandler.New(gwConfig)); err != nil {
		log.Fatalln(err)
	}
}
