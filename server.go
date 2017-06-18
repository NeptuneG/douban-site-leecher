package main

import (
	"net/http"
	"os"

	"github.com/NeptuneG/douban-site-leecher/leecher"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	http.HandleFunc("/", leecher.Handler)
	http.ListenAndServe(":"+port, nil)
}
