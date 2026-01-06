package events

import "time"

type Segment struct {
	Type string `json:"type"`
	Min  int    `json:"min,omitempty"`
	Max  int    `json:"max,omitempty"`
}

type OfferCreated struct {
	EventID   string    `json:"eventId"`
	OfferID   string    `json:"offerId"`
	Title     string    `json:"title"`
	Segment   Segment   `json:"segment"`
	CreatedAt time.Time `json:"createdAt"`
}

type NotificationToSend struct {
	NotificationID string    `json:"notificationId"`
	OfferID        string    `json:"offerId"`
	UserID         string    `json:"userId"`
	Template       string    `json:"template"`
	CreatedAt      time.Time `json:"createdAt"`
}

type DLQMessage struct {
	Reason   string `json:"reason"`
	Topic    string `json:"topic"`
	BodyJSON string `json:"bodyJson"`
}
