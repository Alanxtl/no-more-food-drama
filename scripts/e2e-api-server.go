package main

import (
	"log"
	"net/http"

	handler "github.com/Alanxtl/no-more-food-drama/api"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/rooms", handler.Handler)
	mux.HandleFunc("/api/rooms/", handler.Handler)

	log.Fatal(http.ListenAndServe("127.0.0.1:3002", mux))
}
