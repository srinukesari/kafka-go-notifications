# Kafka Go Notifications – Event-Driven Notification System (Go + Kafka)

Kafka Go Notifications is a backend-only, event-driven notification system built using **Golang** and **Apache Kafka**.  
The project is designed to understand **Kafka fundamentals**, **consumer groups**, **offset management**, **fanout patterns**, **idempotent consumers**, and **failure handling (DLQ)**.

---

## High-Level Overview

The system models a real-world scenario where a platform creates notification campaigns (offers, alerts, reminders), and the backend processes them asynchronously using Kafka.

Key goals:
- Decouple producers and consumers
- Handle large fanout safely
- Avoid duplicate notifications
- Handle failures without blocking processing
- Scale horizontally using Kafka partitions and consumer groups

---

## Architecture
    Offer API
        |
        | (OfferCreated)
        v
    Kafka Topic: offers.created
        |
        v
    Fanout Service(dispatcher srv)
        |
        | (NotificationTask per user)
        v
    Kafka Topic: notifications.send
        |
        v
    Worker Service/push-notification-srv (Consumer Group)
        |
        ├── Success → Commit Offset
        └── Failure → Retry → DLQ


---

## Services

### 1. Offer API (`offer-api-service`)
- Exposes an HTTP endpoint to create notification campaigns.
- Publishes a high-level `OfferCreated` event to Kafka.
- Does **not** handle user-level delivery.

**Responsibility:**  
Create campaigns, not deliver notifications.

---

### 2. Fanout Service (`dispatcher-service`)
- Consumes `offers.created`.
- Resolves user segments (currently mocked).
- Expands **one campaign event into many per-user notification tasks**.
- Publishes tasks to `notifications.send`.

**Responsibility:**  
Convert **1 → N** events (fanout).

---

### 3. Worker Service (`push-notifications-service`)
- Consumes `notifications.send` using a consumer group.
- Sends notifications (mocked).
- Implements:
  - retry with backoff
  - in-memory idempotency (dedup)
  - Dead Letter Queue (DLQ)

**Responsibility:**  
Process notification tasks reliably and at scale.

---

## Kafka Topics

| Topic Name | Purpose |
|------------|---------|
| `offers.created` | High-level campaign events |
| `notifications.send` | Per-user notification tasks |
| `notifications.dlq` | Messages that failed after retries |

---

## Kafka Concepts Used

- **Partitions** for parallelism
- **Consumer Groups** for scaling workers
- **Offsets per partition per group**
- **At-least-once delivery**
- **Idempotent consumers**
- **Dead Letter Queue (DLQ)**
- **Retention-based message deletion**

---

## Idempotency & Duplicate Handling

Kafka guarantees **at-least-once delivery**, so duplicate message delivery is possible.

To prevent duplicate notifications:
- Each task has a deterministic `notificationId` (e.g. `campaignId:userId`)
- Worker checks an in-memory dedup store
- A notification is marked as sent **only after successful delivery**

This provides **effectively-once behavior while the service is running**.

> Note: In-memory dedup state is lost on restart.  
> A production system would use Redis cache or a database.

---

## Failure Handling & DLQ

- Each notification task is retried up to **3 times**
- Exponential backoff between retries
- After max retries:
  - Message is sent to `notifications.dlq`
  - Offset is committed
  - Processing continues

This prevents **poison messages** from blocking partitions.

---

## Running the Project

### Start Kafka
```bash
docker compose up -d
```

### Start all 3 Services
```bash
go run ./cmd/offer-api-service/main.go

go run ./cmd/dispatcher-service/main.go

go run ./cmd/push-notification-service/main.go
```

### Create an offer
```bash
curl -X POST http://localhost:8081/offers \
  -H "Content-Type: application/json" \
  -d '{"title":"New Year Sale","segment":{"type":"ALL_USERS"}}'
```








