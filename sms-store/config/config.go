package config

import (
	"log"
	"os"
)

// Config holds all runtime configuration, sourced from environment variables.
type Config struct {
	ServerPort      string
	MongoURI        string
	MongoDB         string
	MongoCollection string
	KafkaBrokers    string
	KafkaTopic      string
	KafkaGroupID    string
}

// Load reads environment variables and returns a Config with sensible defaults.
func Load() *Config {
	cfg := &Config{
		ServerPort:      getEnv("SERVER_PORT", "8081"),
		MongoURI:        getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDB:         getEnv("MONGO_DB", "smsstore"),
		MongoCollection: getEnv("MONGO_COLLECTION", "sms_messages"),
		KafkaBrokers:    getEnv("KAFKA_BROKERS", "localhost:9092"),
		KafkaTopic:      getEnv("KAFKA_TOPIC", "sms-events"),
		KafkaGroupID:    getEnv("KAFKA_GROUP_ID", "sms-store-group"),
	}

	log.Printf("[config] Loaded: port=%s mongo=%s db=%s topic=%s",
		cfg.ServerPort, cfg.MongoURI, cfg.MongoDB, cfg.KafkaTopic)

	return cfg
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
