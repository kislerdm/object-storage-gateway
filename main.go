package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
)

func main() {
	port := "3000"
	if v, err := strconv.Atoi(os.Getenv("PORT")); err == nil && v > 1000 {
		port = strconv.Itoa(v)
	}

	h := HTTPHandler{}

	if err := http.ListenAndServe(":"+port, h); err != nil {
		log.Fatalln(err)
	}
}
