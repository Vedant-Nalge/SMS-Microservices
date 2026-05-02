package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SmsStatus represents the delivery status of an SMS.
type SmsStatus string

const (
	StatusSuccess SmsStatus = "SUCCESS"
	StatusFailed  SmsStatus = "FAILED"
	StatusBlocked SmsStatus = "BLOCKED"
)

// SmsRecord is the MongoDB document stored for every SMS event.
type SmsRecord struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"         json:"id,omitempty"`
	MessageID      string             `bson:"message_id"            json:"messageId"`
	UserID         string             `bson:"user_id"               json:"userId"`
	PhoneNumber    string             `bson:"phone_number"          json:"phoneNumber"`
	Message        string             `bson:"message"               json:"message"`
	Status         SmsStatus          `bson:"status"                json:"status"`
	VendorResponse string             `bson:"vendor_response"       json:"vendorResponse,omitempty"`
	SentAt         time.Time          `bson:"sent_at"               json:"sentAt"`
	StoredAt       time.Time          `bson:"stored_at"             json:"storedAt"`
}

// SmsEvent mirrors the Kafka event published by the Java SMS Sender.
type SmsEvent struct {
	MessageID      string    `json:"messageId"`
	UserID         string    `json:"userId"`
	PhoneNumber    string    `json:"phoneNumber"`
	Message        string    `json:"message"`
	Status         SmsStatus `json:"status"`
	VendorResponse string    `json:"vendorResponse"`
	SentAt         time.Time `json:"sentAt"`
}

// APIResponse is a generic JSON envelope for HTTP responses.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Count   *int        `json:"count,omitempty"`
}
