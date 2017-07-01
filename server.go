package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	f, _ := os.Create("/var/log/leecher-server.log")
	defer f.Close()
	log.SetOutput(f)

	http.HandleFunc("/", Handler)

	log.Printf("Listening on port %s\n\n", port)
	http.ListenAndServe(":"+port, nil)
}
