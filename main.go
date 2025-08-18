package main

import (
	"log"
	"net/http"
)

func main() {
	// 1. Create a new http.ServeMux
	mux := http.NewServeMux()

	// 2. Use a standard http.FileServer as the handler and http.Dir to set the root directory
	// The path `.` represents the current directory.
	fs := http.FileServer(http.Dir("."))

	// 3. Use the http.NewServeMux's .Handle() method to add a handler for the root path (`/`).
	mux.Handle("/", fs)

	// 4. Create and start the server
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println("Fileserver starting on :8080...")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed to start: %v", err)
	}
}
