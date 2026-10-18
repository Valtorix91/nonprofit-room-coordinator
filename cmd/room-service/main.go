package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"

	"example.com/nonprofit-room-coordinator/internal/infrai"
	"example.com/nonprofit-room-coordinator/internal/matchmaking"
)

type roomAPI interface {
	CreateChannel(ctx context.Context, channel, key string) error
	Publish(ctx context.Context, channel, event string, data any, accountID, key string) error
}

type server struct {
	rooms roomAPI
}

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}

	mux := http.NewServeMux()
	s := server{rooms: infrai.New(key)}
	mux.HandleFunc("POST /matchmaking", s.matchmaking)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	log.Println("room service listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func (s server) matchmaking(w http.ResponseWriter, r *http.Request) {
	var input matchmaking.Request
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	state, err := matchmaking.Match(input)
	if err != nil {
		status := http.StatusUnprocessableEntity
		if !errors.Is(err, matchmaking.ErrNoMatch) {
			status = http.StatusBadRequest
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	if err := s.rooms.CreateChannel(r.Context(), state.Channel, input.IdempotencyKey+":channel"); err != nil {
		writeUpstreamError(w, err)
		return
	}
	if err := s.rooms.Publish(r.Context(), state.Channel, "room.state.ready", state, input.CampaignID, input.IdempotencyKey+":state"); err != nil {
		writeUpstreamError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, state)
}

func writeUpstreamError(w http.ResponseWriter, err error) {
	var apiErr *infrai.APIError
	if errors.As(err, &apiErr) && apiErr.HTTPStatus >= 400 && apiErr.HTTPStatus < 500 {
		writeJSON(w, apiErr.HTTPStatus, map[string]string{"error": apiErr.Message, "code": apiErr.Code})
		return
	}
	writeJSON(w, http.StatusBadGateway, map[string]string{"error": "room provider request failed"})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode response: %v", err)
	}
}
