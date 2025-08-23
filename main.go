package main

import (
	"chirpy/internal/database"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// apiConfig holds our server's state, including the fileserver hit count.
type apiConfig struct {
	fileserverHits atomic.Int32
	DB             *database.Queries
	Platform       string // New field for the PLATFORM environment variable
}

// New `createChirpBody` struct for the incoming JSON
type createChirpBody struct {
	Body   string `json:"body"`
	UserID string `json:"user_id"`
}

// New `Chirp` struct for the outgoing JSON response
type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

// createUserBody represents the expected JSON request body for a new user.
type createUserBody struct {
	Email string `json:"email"`
}

// errorResponse represents a generic JSON error response.
type errorResponse struct {
	Error string `json:"error"`
}

// User represents the User data returned to the client.
type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

// sanitizeChirp replaces profane words in a given string.
func sanitizeChirp(s string) string {
	profaneWords := []string{"kerfuffle", "sharbert", "fornax"}
	words := strings.Split(s, " ")

	for i, word := range words {
		cleanedWord := strings.ToLower(word)
		isProfane := slices.Contains(profaneWords, cleanedWord)
		if isProfane {
			words[i] = "****"
		}
	}

	return strings.Join(words, " ")
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

// resetHandler resets the fileserverHits counter to zero and deletes all users if in dev.
func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {
	if cfg.Platform != "dev" {
		respondWithError(w, http.StatusForbidden, "Forbidden: This endpoint is only available in the 'dev' environment")
		return
	}

	// Delete all users from the database
	err := cfg.DB.DeleteUsers(context.Background())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to delete users")
		return
	}

	// Reset fileserver hits
	cfg.fileserverHits.Store(0)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// adminMetricsHandler returns an HTML page with the hit count.
func (cfg *apiConfig) adminMetricsHandler(w http.ResponseWriter, r *http.Request) {
	hits := cfg.fileserverHits.Load()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	// Use fmt.Sprintf to format the HTML string with the hit count
	html := fmt.Sprintf(`<html>
	<body>
	  <h1>Welcome, Chirpy Admin</h1>
	  <p>Chirpy has been visited %d times!</p>
	</body>
</html>`, hits)
	w.Write([]byte(html))
}

// createUserHandler creates a new user in the database.
func (cfg *apiConfig) createUserHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var reqBody createUserBody

	err := decoder.Decode(&reqBody)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Generate a new UUID and get current time for timestamps
	now := time.Now().UTC()
	id := uuid.New()

	// Call the generated SQLC function to create the user
	dbUser, err := cfg.DB.CreateUser(context.Background(), database.CreateUserParams{
		ID:        id,
		CreatedAt: now,
		UpdatedAt: now,
		Email:     reqBody.Email,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	// Map the database.User to the main package's User struct
	user := User{
		ID:        dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email:     dbUser.Email,
	}

	respondWithJSON(w, http.StatusCreated, user)
}

func (cfg *apiConfig) createChirpHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var reqBody createChirpBody

	err := decoder.Decode(&reqBody)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Port all logic from the old `validate_chirp` handler
	if len(reqBody.Body) > 140 {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long")
		return
	}

	cleanedBody := sanitizeChirp(reqBody.Body)
	now := time.Now().UTC()
	id := uuid.New()
	userID, err := uuid.Parse(reqBody.UserID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid User ID")
		return
	}

	// Call the generated SQLC function to create the chirp
	dbChirp, err := cfg.DB.CreateChirp(r.Context(), database.CreateChirpParams{
		ID:        id,
		CreatedAt: now,
		UpdatedAt: now,
		Body:      cleanedBody,
		UserID:    userID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create chirp")
		return
	}

	// Map the database.Chirp to the main package's Chirp struct
	chirp := Chirp{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		UserID:    dbChirp.UserID,
	}

	respondWithJSON(w, http.StatusCreated, chirp)
}

// respondWithError is a helper function to send JSON error responses.
func respondWithError(w http.ResponseWriter, code int, msg string) {
	respondWithJSON(w, code, errorResponse{Error: msg})
}

// respondWithJSON is a helper function to send a JSON response.
func respondWithJSON(w http.ResponseWriter, code int, payload any) {
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
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
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Get the DB_URL from environment variables
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL not found in environment variables")
	}
	platform := os.Getenv("PLATFORM")

	// Open a connection to the database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Error opening database connection: %v", err)
	}
	defer db.Close() // Defer closing the database connection

	// Use the SQLC generated database package to create new queries
	dbQueries := database.New(db)

	mux := http.NewServeMux()
	apiCfg := &apiConfig{
		DB:       dbQueries, // Store the dbQueries in the apiConfig struct
		Platform: platform,
	}

	// API endpoints
	mux.HandleFunc("POST /api/users", apiCfg.createUserHandler)
	mux.HandleFunc("POST /api/chirps", apiCfg.createChirpHandler)
	mux.HandleFunc("GET /api/healthz", healthzHandler)
	mux.HandleFunc("GET /api/metrics", apiCfg.metricsHandler)

	// Admin endpoints
	mux.HandleFunc("GET /admin/metrics", apiCfg.adminMetricsHandler)
	mux.HandleFunc("POST /admin/reset", apiCfg.resetHandler)

	// Fileserver remains at the /app/ path
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
