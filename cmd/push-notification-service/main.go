package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	store "kafka-go-notifications/internal/cache"
	"kafka-go-notifications/internal/events"
	kafkah "kafka-go-notifications/internal/kafka"
)

func main() {
	dedupe := store.NewMemDedupe(30*time.Minute, 200_000)

	brokers := strings.Split(getenv("KAFKA_BROKERS", "localhost:9092"), ",")
	inTopic := getenv("SEND_TOPIC", "notifications.send")
	dlqTopic := getenv("DLQ_TOPIC", "notifications.dlq")
	groupID := getenv("GROUP_ID", "worker-svc")

	reader := kafkah.NewReader(brokers, inTopic, groupID)
	defer reader.Close()

	dlqWriter := kafkah.NewWriter(brokers, dlqTopic)
	defer dlqWriter.Close()

	log.Printf("worker-svc started: brokers=%v in=%s dlq=%s group=%s", brokers, inTopic, dlqTopic, groupID)

	for {
		msg, err := reader.FetchMessage(context.Background())
		if err != nil {
			log.Fatalf("fetch: %v", err)
		}

		var task events.NotificationToSend
		if err := json.Unmarshal(msg.Value, &task); err != nil {
			log.Printf("bad NotificationToSend json: %v", err)
			_ = reader.CommitMessages(context.Background(), msg)
			continue
		}

		if !dedupe.MarkSeen(task.NotificationID) {
			_ = reader.CommitMessages(context.Background(), msg)
			continue
		}

		if err := processWithRetry(task, 3); err != nil {
			log.Printf("failed after retries notifId=%s err=%v -> DLQ", task.NotificationID, err)
			dlq := events.DLQMessage{
				Reason:   err.Error(),
				Topic:    inTopic,
				BodyJSON: string(msg.Value),
			}
			_ = kafkah.WriteJSON(context.Background(), dlqWriter, task.UserID, dlq)
			_ = reader.CommitMessages(context.Background(), msg)
			continue
		}

		_ = reader.CommitMessages(context.Background(), msg)
	}
}

func processWithRetry(task events.NotificationToSend, attempts int) error {
	var err error
	backoff := 200 * time.Millisecond
	for i := 1; i <= attempts; i++ {
		err = mockSend(task)
		if err == nil {
			return nil
		}
		time.Sleep(backoff)
		backoff *= 2
	}
	return err
}

func mockSend(task events.NotificationToSend) error {
	log.Printf("SENT offerId=%s to user=%s notifId=%s template=%s", task.OfferID, task.UserID, task.NotificationID, task.Template)
	return nil
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
