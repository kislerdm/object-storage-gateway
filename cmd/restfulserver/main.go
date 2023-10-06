package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	gateway "github.com/kislerdm/minio-gateway"
	"github.com/kislerdm/minio-gateway/internal/docker"
	"github.com/kislerdm/minio-gateway/internal/restfulhandler"
)

func main() {
	port := "3000"
	if v, err := strconv.Atoi(os.Getenv("PORT")); err == nil && v > 1000 {
		port = strconv.Itoa(v)
	}

	nodeID := "amazin-object-storage-node"
	if v := os.Getenv("STORAGE_NODE_ID"); v != "" {
		nodeID = v
	}

	dockerAdapter, err := docker.NewClient()
	if err != nil {
		log.Fatalln(err)
	}

	gwClient, err := gateway.New(nodeID, dockerAdapter)
	if err != nil {
		log.Fatalln(err)
	}

	if err := http.ListenAndServe(":"+port, restfulhandler.New(gwClient)); err != nil {
		log.Fatalln(err)
	}
}
