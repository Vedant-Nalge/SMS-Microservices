package kafka

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	kafkago "github.com/segmentio/kafka-go"
	"github.com/sms/sms-store/config"
	"github.com/sms/sms-store/internal/model"
	"github.com/sms/sms-store/internal/service"
)

// Consumer wraps a kafka-go Reader and dispatches SMS events to SmsService.
type Consumer struct {
	reader *kafkago.Reader
	svc    *service.SmsService
}

// New creates a Kafka consumer configured from cfg.
func New(cfg *config.Config, svc *service.SmsService) *Consumer {
	brokers := strings.Split(cfg.KafkaBrokers, ",")

	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:        brokers,
		Topic:          cfg.KafkaTopic,
		GroupID:        cfg.KafkaGroupID,
		MinBytes:       1,
		MaxBytes:       10e6, // 10 MB
		CommitInterval: time.Second,
		StartOffset:    kafkago.FirstOffset,
		ErrorLogger:    kafkago.LoggerFunc(func(msg string, args ...interface{}) { log.Printf("[kafka-err] "+msg, args...) }),
	})

	return &Consumer{reader: reader, svc: svc}
}

// Start begins consuming messages in a blocking loop.
// Call it in a goroutine; it stops when ctx is cancelled.
func (c *Consumer) Start(ctx context.Context) {
	log.Printf("[kafka] Consumer starting: topic=%s groupId=%s",
		c.reader.Config().Topic, c.reader.Config().GroupID)

	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				log.Println("[kafka] Consumer shutting down gracefully")
				break
			}
			log.Printf("[kafka] Read error: %v", err)
			time.Sleep(500 * time.Millisecond) // brief back-off
			continue
		}

		log.Printf("[kafka] Received message: topic=%s partition=%d offset=%d key=%s",
			msg.Topic, msg.Partition, msg.Offset, string(msg.Key))

		c.processMessage(ctx, msg)
	}

	if err := c.reader.Close(); err != nil {
		log.Printf("[kafka] Error closing reader: %v", err)
	}
}

func (c *Consumer) processMessage(ctx context.Context, msg kafkago.Message) {
	var event model.SmsEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		log.Printf("[kafka] Failed to unmarshal event at offset=%d: %v", msg.Offset, err)
		return // skip malformed message (already committed by auto-commit)
	}

	storeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := c.svc.StoreEvent(storeCtx, &event); err != nil {
		log.Printf("[kafka] Failed to store event messageId=%s: %v", event.MessageID, err)
		// In production consider a dead-letter topic here
		return
	}

	log.Printf("[kafka] Successfully processed SMS event: messageId=%s userId=%s status=%s",
		event.MessageID, event.UserID, event.Status)
}
