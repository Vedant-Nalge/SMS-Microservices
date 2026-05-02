package repository

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sms/sms-store/config"
	"github.com/sms/sms-store/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// SmsRepository handles all MongoDB operations for SMS records.
type SmsRepository struct {
	collection *mongo.Collection
}

// New creates a connected SmsRepository and ensures required indexes.
func New(cfg *config.Config) (*SmsRepository, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOpts := options.Client().ApplyURI(cfg.MongoURI)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("mongo connect: %w", err)
	}

	if err = client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("mongo ping: %w", err)
	}

	log.Println("[repository] Connected to MongoDB")

	col := client.Database(cfg.MongoDB).Collection(cfg.MongoCollection)

	repo := &SmsRepository{collection: col}
	if err = repo.ensureIndexes(ctx); err != nil {
		log.Printf("[repository] Warning: could not create indexes: %v", err)
	}

	return repo, nil
}

// Save inserts a new SmsRecord and returns the generated ObjectID string.
func (r *SmsRepository) Save(ctx context.Context, record *model.SmsRecord) (string, error) {
	record.ID = primitive.NewObjectID()
	record.StoredAt = time.Now().UTC()

	result, err := r.collection.InsertOne(ctx, record)
	if err != nil {
		return "", fmt.Errorf("insert sms record: %w", err)
	}

	insertedID := result.InsertedID.(primitive.ObjectID).Hex()
	log.Printf("[repository] Saved SMS record: _id=%s messageId=%s userId=%s status=%s",
		insertedID, record.MessageID, record.UserID, record.Status)

	return insertedID, nil
}

// FindByUserID returns all SMS records for a given userId, newest first.
func (r *SmsRepository) FindByUserID(ctx context.Context, userID string) ([]*model.SmsRecord, error) {
	filter := bson.M{"user_id": userID}
	opts := options.Find().SetSort(bson.D{{Key: "sent_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("find by userId=%s: %w", userID, err)
	}
	defer cursor.Close(ctx)

	var records []*model.SmsRecord
	if err = cursor.All(ctx, &records); err != nil {
		return nil, fmt.Errorf("decode records: %w", err)
	}

	return records, nil
}

// ensureIndexes creates compound and single-field indexes for common query patterns.
func (r *SmsRepository) ensureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "sent_at", Value: -1}},
			Options: options.Index().SetName("idx_user_id_sent_at"),
		},
		{
			Keys:    bson.D{{Key: "message_id", Value: 1}},
			Options: options.Index().SetName("idx_message_id").SetUnique(true),
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	return err
}
