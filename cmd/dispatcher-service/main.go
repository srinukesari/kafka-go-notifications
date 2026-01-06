package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"

	"kafka-go-notifications/internal/events"
	kafkah "kafka-go-notifications/internal/kafka"
)

func main() {
	brokers := strings.Split(getenv("KAFKA_BROKERS", "localhost:9092"), ",")
	// fmt.Println(brokers)
	inTopic := getenv("OFFERS_TOPIC", "offers.created")
	outTopic := getenv("SEND_TOPIC", "notifications.send")
	groupID := getenv("GROUP_ID", "fanout-svc")

	reader := kafkah.NewReader(brokers, inTopic, groupID)
	defer reader.Close()

	writer := kafkah.NewWriter(brokers, outTopic)
	defer writer.Close()

	// "Mock users in memory”
	users := []string{"u1", "u2", "u3", "u4", "u5", "u6", "u7", "u8", "u9", "u10"}

	log.Printf("dispatcher-service started: brokers=%v in=%s out=%s group=%s", brokers, inTopic, outTopic, groupID)

	for {
		msg, err := reader.FetchMessage(context.Background())
		if err != nil {
			log.Fatalf("fetch: %v", err)
		}

		var ev events.OfferCreated
		if err := json.Unmarshal(msg.Value, &ev); err != nil {
			log.Printf("bad OfferCreated json, send to DLQ later: %v", err)
			_ = reader.CommitMessages(context.Background(), msg)
			continue
		}

		if ev.Segment.Type != "ALL_USERS" {
			log.Printf("segment not supported yet, skipping offerId=%s segment=%s", ev.OfferID, ev.Segment.Type)
			_ = reader.CommitMessages(context.Background(), msg)
			continue
		}

		for _, userID := range users {
			task := events.NotificationToSend{
				NotificationID: uuid.NewString(),
				OfferID:        ev.OfferID,
				UserID:         userID,
				Template:       "OFFER_PUSH",
				CreatedAt:      time.Now().UTC(),
			}
			if err := kafkah.WriteJSON(context.Background(), writer, userID, task); err != nil {
				log.Printf("produce notifications.send failed: %v", err)

				goto retryOffer
			}
		}

		if err := reader.CommitMessages(context.Background(), msg); err != nil {
			log.Printf("commit failed: %v", err)
		}
		continue

	retryOffer:
		time.Sleep(500 * time.Millisecond)
		_ = kafkago.LoggerFunc(log.Printf)
	}
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
