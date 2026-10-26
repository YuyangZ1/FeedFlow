package main

import (
	"feedflow/backend"
	"feedflow/handler"
	"fmt"
	"log"
	"net/http"
)

func main() {
	fmt.Println("started-service")

	// Build the router first so missing config (JWT_SECRET) fails fast,
	// before connecting to Elasticsearch or storage.
	router := handler.InitRouter()

	backend.InitElasticsearchBackend()
	backend.InitGCSBackend()

	log.Fatal(http.ListenAndServe(":8080", router))
}
