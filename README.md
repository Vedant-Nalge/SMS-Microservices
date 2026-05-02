# Polyglot Distributed SMS Service

A microservices-based SMS notification system split across two services:

| Service | Language | Port | Role |
|---------|----------|------|------|
| **SMS Sender** | Java / Spring Boot | `8080` | Gateway — validates requests, checks block list, calls mock vendor, publishes Kafka event |
| **SMS Store** | GoLang / net/http | `8081` | Persistence — consumes Kafka events, stores in MongoDB, exposes history API |

---

## Architecture

```
Client
  │
  ▼ POST /v1/sms/send
┌─────────────────────┐
│    SMS Sender       │──── 2. Check Redis block list ────► Redis
│    (Java/Spring)    │
│                     │──── 3. Call mock 3P vendor ────────► Mock (random SUCCESS/FAIL)
│                     │
│                     │──── 4. Publish to Kafka ───────────► Kafka (topic: sms-events)
└─────────────────────┘                                           │
                                                                  │ 5. Consume
                                                        ┌─────────▼────────────┐
                                                        │    SMS Store         │──► MongoDB
                                                        │    (GoLang/net/http) │
                                                        │                      │◄── GET /v1/user/{id}/messages
                                                        └──────────────────────┘
```

---

## Prerequisites

- Docker & Docker Compose
- (Optional for local dev) Java 17+, Maven 3.9+, Go 1.21+, Redis CLI

---

## Running with Docker Compose

```bash
# Clone / enter the repo root
cd sms-service

# Build and start everything
docker compose up --build

# Services will be available once healthy:
#   SMS Sender  → http://localhost:8080
#   SMS Store   → http://localhost:8081
#   Kafka       → localhost:9092
#   Redis       → localhost:6379
#   MongoDB     → localhost:27017
```

---

## Running Locally (without Docker)

### Infrastructure (still via Docker)
```bash
docker compose up zookeeper kafka redis mongodb
```

### SMS Sender (Java)
```bash
cd sms-sender
mvn spring-boot:run
# Listens on :8080
```

### SMS Store (Go)
```bash
cd sms-store
go run ./cmd/server
# Listens on :8081
```

---

## API Reference

### SMS Sender — `POST /v1/sms/send`

Send an SMS to a user.

**Request body:**
```json
{
  "userId":      "user-123",
  "phoneNumber": "+919876543210",
  "message":     "Your OTP is 4567"
}
```

**Responses:**

| HTTP | Status field | Meaning |
|------|-------------|---------|
| 200 | `SUCCESS` | Vendor accepted the message |
| 403 | `BLOCKED` | User is on the Redis block list |
| 502 | `FAILED` | Mock vendor returned a failure |

**Example — success:**
```bash
curl -X POST http://localhost:8080/v1/sms/send \
  -H "Content-Type: application/json" \
  -d '{"userId":"user-123","phoneNumber":"+919876543210","message":"Hello!"}'
```

```json
{
  "messageId": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "status": "SUCCESS",
  "userId": "user-123",
  "phoneNumber": "+919876543210",
  "errorMessage": null,
  "timestamp": "2024-01-15T10:30:00Z"
}
```

---

### SMS Sender — `GET /v1/sms/health`

Health check.

```bash
curl http://localhost:8080/v1/sms/health
# {"status":"UP","service":"sms-sender"}
```

---

### SMS Store — `GET /v1/user/{userId}/messages`

Retrieve all SMS records for a user (newest first).

```bash
curl http://localhost:8081/v1/user/user-123/messages
```

```json
{
  "success": true,
  "count": 2,
  "data": [
    {
      "id": "65a1b2c3d4e5f6a7b8c9d0e1",
      "messageId": "3fa85f64-...",
      "userId": "user-123",
      "phoneNumber": "+919876543210",
      "message": "Hello!",
      "status": "SUCCESS",
      "vendorResponse": "",
      "sentAt": "2024-01-15T10:30:00Z",
      "storedAt": "2024-01-15T10:30:01Z"
    }
  ]
}
```

---

### SMS Store — `GET /health`

```bash
curl http://localhost:8081/health
# {"status":"UP","service":"sms-store"}
```

---

## Block List Management

The block list is a Redis Set at key `blocked:users`.

```bash
# Block a user
redis-cli SADD blocked:users user-123

# Unblock a user
redis-cli SREM blocked:users user-123

# List all blocked users
redis-cli SMEMBERS blocked:users
```

---

## End-to-End Demo

```bash
# Make sure all services are running first
bash demo.sh
```

The script:
1. Checks health of both services
2. Sends a successful SMS for `user-demo-001`
3. Blocks `user-blocked-999` via Redis
4. Attempts to send to the blocked user (receives `BLOCKED`)
5. Sends a second SMS for `user-demo-001`
6. Waits for the Kafka consumer to store events
7. Retrieves message history for `user-demo-001` from the SMS Store
8. Retrieves history for the blocked user (the BLOCKED event is stored)
9. Unblocks the user

---

## Running Tests

### Java (SMS Sender)
```bash
cd sms-sender
mvn test
```

### Go (SMS Store)
```bash
cd sms-store
go test ./internal/service/...
```

---

## Project Structure

```
sms-service/
├── docker-compose.yml
├── demo.sh
├── README.md
│
├── sms-sender/                        # Java / Spring Boot
│   ├── Dockerfile
│   ├── pom.xml
│   └── src/
│       ├── main/
│       │   ├── java/com/sms/sender/
│       │   │   ├── SmsSenderApplication.java
│       │   │   ├── config/
│       │   │   │   ├── AppConfig.java          # RestTemplate, RedisTemplate beans
│       │   │   │   └── KafkaProducerConfig.java
│       │   │   ├── controller/
│       │   │   │   ├── SmsController.java
│       │   │   │   └── GlobalExceptionHandler.java
│       │   │   ├── kafka/
│       │   │   │   └── SmsEventProducer.java
│       │   │   ├── model/
│       │   │   │   └── Models.java             # SendSmsRequest/Response, SmsEvent, VendorResult
│       │   │   └── service/
│       │   │       ├── BlockListService.java   # Redis block-list logic
│       │   │       ├── SmsVendorService.java   # Mock 3P vendor
│       │   │       └── SmsService.java         # Orchestration
│       │   └── resources/
│       │       └── application.yml
│       └── test/
│           └── java/com/sms/sender/service/
│               └── SmsServiceTest.java
│
└── sms-store/                         # GoLang / net/http
    ├── Dockerfile
    ├── go.mod
    ├── cmd/server/
    │   └── main.go                    # Entrypoint + graceful shutdown
    ├── config/
    │   └── config.go                  # Env-var config
    └── internal/
        ├── handler/
        │   └── sms_handler.go         # HTTP routes
        ├── kafka/
        │   └── consumer.go            # Kafka consumer loop
        ├── model/
        │   └── model.go               # SmsRecord, SmsEvent, APIResponse
        ├── repository/
        │   └── sms_repository.go      # MongoDB operations
        └── service/
            ├── sms_service.go         # Business logic + Repository interface
            └── sms_service_test.go    # Unit tests with stub repo
```

---

## Configuration Reference

### SMS Sender (environment variables / `application.yml`)

| Variable | Default | Description |
|----------|---------|-------------|
| `SPRING_KAFKA_BOOTSTRAP_SERVERS` | `localhost:9092` | Kafka brokers |
| `SPRING_DATA_REDIS_HOST` | `localhost` | Redis host |
| `SPRING_DATA_REDIS_PORT` | `6379` | Redis port |
| `APP_SMS_MOCK_SUCCESS_RATE` | `0.9` | Probability of mock vendor success |

### SMS Store (environment variables)

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | `8081` | HTTP listen port |
| `MONGO_URI` | `mongodb://localhost:27017` | MongoDB connection URI |
| `MONGO_DB` | `smsstore` | Database name |
| `MONGO_COLLECTION` | `sms_messages` | Collection name |
| `KAFKA_BROKERS` | `localhost:9092` | Kafka brokers (comma-separated) |
| `KAFKA_TOPIC` | `sms-events` | Topic to consume |
| `KAFKA_GROUP_ID` | `sms-store-group` | Consumer group ID |
