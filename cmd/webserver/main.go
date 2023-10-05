package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/kislerdm/gateway"
	"github.com/kislerdm/gateway/internal/httphandler"
)

func main() {
	port := "3000"
	if v, err := strconv.Atoi(os.Getenv("PORT")); err == nil && v > 1000 {
		port = strconv.Itoa(v)
	}

	hostPrefix := "homework-object-storage-amazin-object-storage-node"
	if v := os.Getenv("HOST_PREFIX"); v != "" {
		hostPrefix = v
	}

	gwClint, err := gateway.NewGateway(hostPrefix)
	if err != nil {
		log.Fatalln(err)
	}

	h := httphandler.NewHandler(gwClint)

	if err := http.ListenAndServe(":"+port, h); err != nil {
		log.Fatalln(err)
	}
}
