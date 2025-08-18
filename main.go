package main

import (
	"log"
	"net/http"
)

func main() {
	// 1. Create a new http.ServeMux
	mux := http.NewServeMux()

	// 2. Create a new http.Server struct.
	// We'll pass our mux to the server as its handler.
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// 3. Use the server's ListenAndServe method to start the server.
	// This is a blocking call, so the program will stay running here.
	log.Println("Server starting on :8080...")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed to start: %v", err)
	}
}
