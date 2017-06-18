package main

import (
	"log"
	"net/http"
	"os"

	"github.com/NeptuneG/douban-site-leecher/leecher"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}
	f, _ := os.Create("douban-site-leecher.log")
	defer f.Close()
	log.SetOutput(f)

	http.HandleFunc("/", leecher.Handler)

	log.Printf("Listening on port %s\n\n", port)
	http.ListenAndServe(":"+port, nil)
}
