package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sms/sms-store/internal/model"
)

// Repository defines the persistence operations required by SmsService.
type Repository interface {
	Save(ctx context.Context, record *model.SmsRecord) (string, error)
	FindByUserID(ctx context.Context, userID string) ([]*model.SmsRecord, error)
}

// SmsService encapsulates business logic for storing and retrieving SMS records.
type SmsService struct {
	repo Repository
}

// New creates a new SmsService backed by the given repository.
func New(repo Repository) *SmsService {
	return &SmsService{repo: repo}
}

// NewWithRepo is an alias for New — used by tests to inject stubs.
func NewWithRepo(repo Repository) *SmsService { return New(repo) }

// StoreEvent converts a Kafka SmsEvent into a persisted SmsRecord.
func (s *SmsService) StoreEvent(ctx context.Context, event *model.SmsEvent) error {
	if event.MessageID == "" {
		return fmt.Errorf("messageId is required")
	}
	if event.UserID == "" {
		return fmt.Errorf("userId is required")
	}

	sentAt := event.SentAt
	if sentAt.IsZero() {
		sentAt = time.Now().UTC()
	}

	record := &model.SmsRecord{
		MessageID:      event.MessageID,
		UserID:         event.UserID,
		PhoneNumber:    event.PhoneNumber,
		Message:        event.Message,
		Status:         event.Status,
		VendorResponse: event.VendorResponse,
		SentAt:         sentAt,
	}

	id, err := s.repo.Save(ctx, record)
	if err != nil {
		return fmt.Errorf("save event: %w", err)
	}

	log.Printf("[service] Stored SMS event: _id=%s messageId=%s userId=%s status=%s",
		id, event.MessageID, event.UserID, event.Status)
	return nil
}

// GetMessagesByUserID fetches the SMS history for a user.
func (s *SmsService) GetMessagesByUserID(ctx context.Context, userID string) ([]*model.SmsRecord, error) {
	if userID == "" {
		return nil, fmt.Errorf("userId must not be empty")
	}

	records, err := s.repo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("fetch history for userId=%s: %w", userID, err)
	}

	log.Printf("[service] Retrieved %d messages for userId=%s", len(records), userID)
	return records, nil
}
