package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"kafka-go-notifications/internal/events"
	kafkah "kafka-go-notifications/internal/kafka"
)

type createOfferReq struct {
	Title   string         `json:"title"`
	Segment events.Segment `json:"segment"`
}

func main() {
	brokers := strings.Split(getenv("KAFKA_BROKERS", "localhost:9092"), ",")
	topic := getenv("OFFERS_TOPIC", "offers.created")
	addr := getenv("HTTP_ADDR", ":8081")

	writer := kafkah.NewWriter(brokers, topic)
	defer writer.Close()

	http.HandleFunc("/offers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req createOfferReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		if req.Title == "" {
			http.Error(w, "title required", http.StatusBadRequest)
			return
		}
		if req.Segment.Type == "" {
			req.Segment.Type = "ALL_USERS"
		}

		ev := events.OfferCreated{
			EventID:   uuid.NewString(),
			OfferID:   uuid.NewString(),
			Title:     req.Title,
			Segment:   req.Segment,
			CreatedAt: time.Now().UTC(),
		}

		if err := kafkah.WriteJSON(context.Background(), writer, ev.OfferID, ev); err != nil {
			http.Error(w, "kafka write failed", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ev)
	})

	log.Printf("offer-api listening on %s, brokers=%v topic=%s", addr, brokers, topic)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
