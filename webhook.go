package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"
)

type webhookBody struct {
	Event string `json:"event"`
	Data  struct {
		UserID string `json:"user_id"`
	} `json:"data"`
}

func (cfg *apiConfig) webhookHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var reqBody webhookBody

	err := decoder.Decode(&reqBody)
	if err != nil {
		log.Printf("Error decoding webhook body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if reqBody.Event != "user.upgraded" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	userID, err := uuid.Parse(reqBody.Data.UserID)
	if err != nil {
		log.Printf("Invalid user ID in webhook: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	_, err = cfg.DB.UpdateUserIsChirpyRed(r.Context(), userID)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		log.Printf("Failed to update user to Chirpy Red: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
