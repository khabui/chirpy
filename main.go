package main

import (
	"log"
	"net/http"
	"strconv"
	"sync/atomic"
)

// apiConfig holds our server's state, including the fileserver hit count.
type apiConfig struct {
	fileserverHits atomic.Int32
}

// middlewareMetricsInc is a middleware that increments the fileserverHits counter.
func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

// metricsHandler writes the current hit count to the response.
func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, r *http.Request) {
	hits := cfg.fileserverHits.Load()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	// Convert the integer to a string using strconv.Itoa()
	w.Write([]byte("Hits: " + strconv.Itoa(int(hits))))
}

// resetHandler resets the fileserverHits counter to zero.
func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	cfg.fileserverHits.Store(0)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// healthzHandler handles requests to the /healthz endpoint
func healthzHandler(w http.ResponseWriter, r *http.Request) {
	// Set the Content-Type header
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	// Write the status code
	w.WriteHeader(http.StatusOK)

	// Write the body text
	w.Write([]byte(http.StatusText(http.StatusOK)))
}

func main() {
	mux := http.NewServeMux()
	apiCfg := &apiConfig{}

	// Move API endpoints to the /api namespace
	mux.HandleFunc("GET /api/healthz", healthzHandler)
	mux.HandleFunc("GET /api/metrics", apiCfg.metricsHandler)
	mux.HandleFunc("POST /api/reset", apiCfg.resetHandler)

	// Create the fileserver handler and wrap it with the middleware.
	fsHandler := http.StripPrefix("/app/", http.FileServer(http.Dir(".")))
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(fsHandler))

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println("Server starting on :8080...")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed to start: %v", err)
	}
}
