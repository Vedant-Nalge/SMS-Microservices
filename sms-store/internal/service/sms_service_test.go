package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sms/sms-store/internal/model"
	"github.com/sms/sms-store/internal/service"
)

type stubRepo struct {
	saved   []*model.SmsRecord
	byUser  map[string][]*model.SmsRecord
	saveErr error
	findErr error
}

func newStubRepo() *stubRepo {
	return &stubRepo{byUser: make(map[string][]*model.SmsRecord)}
}

func (s *stubRepo) Save(_ context.Context, r *model.SmsRecord) (string, error) {
	if s.saveErr != nil {
		return "", s.saveErr
	}
	s.saved = append(s.saved, r)
	s.byUser[r.UserID] = append(s.byUser[r.UserID], r)
	return "fake-object-id", nil
}

func (s *stubRepo) FindByUserID(_ context.Context, userID string) ([]*model.SmsRecord, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	return s.byUser[userID], nil
}

func TestStoreEvent_Success(t *testing.T) {
	repo := newStubRepo()
	svc := service.NewWithRepo(repo)
	ev := &model.SmsEvent{MessageID: "msg-001", UserID: "user-abc", PhoneNumber: "+919876543210", Message: "Hello", Status: model.StatusSuccess, SentAt: time.Now()}
	if err := svc.StoreEvent(context.Background(), ev); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(repo.saved) != 1 {
		t.Fatalf("expected 1 saved record, got %d", len(repo.saved))
	}
}

func TestStoreEvent_MissingMessageID(t *testing.T) {
	if err := service.NewWithRepo(newStubRepo()).StoreEvent(context.Background(), &model.SmsEvent{UserID: "u1"}); err == nil {
		t.Fatal("expected error for missing messageId")
	}
}

func TestStoreEvent_MissingUserID(t *testing.T) {
	if err := service.NewWithRepo(newStubRepo()).StoreEvent(context.Background(), &model.SmsEvent{MessageID: "m1"}); err == nil {
		t.Fatal("expected error for missing userId")
	}
}

func TestStoreEvent_RepoError(t *testing.T) {
	repo := newStubRepo()
	repo.saveErr = errors.New("mongo unavailable")
	ev := &model.SmsEvent{MessageID: "m3", UserID: "u1", Status: model.StatusSuccess, SentAt: time.Now()}
	if err := service.NewWithRepo(repo).StoreEvent(context.Background(), ev); err == nil {
		t.Fatal("expected error when repo.Save fails")
	}
}

func TestGetMessagesByUserID_ReturnsMessages(t *testing.T) {
	repo := newStubRepo()
	svc := service.NewWithRepo(repo)
	ctx := context.Background()
	_ = svc.StoreEvent(ctx, &model.SmsEvent{MessageID: "m1", UserID: "u1", Status: model.StatusSuccess, SentAt: time.Now()})
	_ = svc.StoreEvent(ctx, &model.SmsEvent{MessageID: "m2", UserID: "u1", Status: model.StatusFailed, SentAt: time.Now()})
	records, err := svc.GetMessagesByUserID(ctx, "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("expected 2 records, got %d", len(records))
	}
}

func TestGetMessagesByUserID_EmptyUserID(t *testing.T) {
	if _, err := service.NewWithRepo(newStubRepo()).GetMessagesByUserID(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty userId")
	}
}

func TestGetMessagesByUserID_RepoError(t *testing.T) {
	repo := newStubRepo()
	repo.findErr = errors.New("mongo timeout")
	if _, err := service.NewWithRepo(repo).GetMessagesByUserID(context.Background(), "u1"); err == nil {
		t.Fatal("expected error when FindByUserID fails")
	}
}
