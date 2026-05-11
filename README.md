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

- **Docker Desktop** — [docker.com/products/docker-desktop](https://www.docker.com/products/docker-desktop)
- (Optional, for local dev without Docker) Java 17+, Maven 3.9+, Go 1.21+

---

## How to Run the Project

### Step 1 — Download and extract the ZIP

Unzip `sms-service.zip`. You will get a folder called `sms-service/`. Inside it there is another `sms-service/` folder — that inner one is where you need to be.

```
sms-service.zip
└── sms-service/
    └── sms-service/        ← work from here
        ├── docker-compose.yml
        ├── sms-sender/
        └── sms-store/
```

### Step 2 — Open a terminal in the right folder

**Windows (PowerShell):**
```powershell
cd C:\path\to\sms-service\sms-service
```

**Mac / Linux:**
```bash
cd /path/to/sms-service/sms-service
```

Confirm you are in the right place — you must see `docker-compose.yml` in the listing:
```bash
ls        # Mac/Linux
dir       # Windows PowerShell
```

### Step 3 — Start all services

```bash
docker compose up --build
```

This single command will:
- Pull all required Docker images (Kafka, Zookeeper, Redis, MongoDB)
- Build the Java SMS Sender image (~3–5 min on first run)
- Build the Go SMS Store image (~2 min on first run)
- Start all 6 containers

**Wait until you see these lines in the logs:**
```
sms-sender  | Started SmsSenderApplication in X.X seconds
sms-store   | [main] SMS Store listening on :8081
sms-store   | [kafka] Consumer starting: topic=sms-events
```

> MongoDB will continuously print checkpoint messages — that is completely normal, ignore them.

### Step 4 — Verify everything is running

Open a **second terminal** and run:

```bash
curl http://localhost:8080/v1/sms/health
curl http://localhost:8081/health
```

Both should return `{"status":"UP"}`.

> **Windows PowerShell note:** Use `Invoke-WebRequest` instead of `curl`:
> ```powershell
> Invoke-WebRequest -Uri http://localhost:8080/v1/sms/health | Select-Object -ExpandProperty Content
> Invoke-WebRequest -Uri http://localhost:8081/health | Select-Object -ExpandProperty Content
> ```

### Step 5 — Send your first SMS

**Mac / Linux / Git Bash:**
```bash
curl -X POST http://localhost:8080/v1/sms/send \
  -H "Content-Type: application/json" \
  -d '{"userId":"user-123","phoneNumber":"+919876543210","message":"Hello from SMS service!"}'
```

**Windows PowerShell:**
```powershell
Invoke-WebRequest -Uri http://localhost:8080/v1/sms/send `
  -Method POST `
  -Headers @{"Content-Type"="application/json"} `
  -Body '{"userId":"user-123","phoneNumber":"+919876543210","message":"Hello from SMS service!"}' |
  Select-Object -ExpandProperty Content
```

Expected response:
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

> The status will be randomly `SUCCESS` or `FAILED` (90/10 split) to simulate a real vendor.

### Step 6 — Check SMS history

Wait 2–3 seconds for Kafka to deliver the event, then:

**Mac / Linux / Git Bash:**
```bash
curl http://localhost:8081/v1/user/user-123/messages
```

**Windows PowerShell:**
```powershell
Invoke-WebRequest -Uri http://localhost:8081/v1/user/user-123/messages |
  Select-Object -ExpandProperty Content
```

You will see the stored SMS record retrieved from MongoDB.

### Step 7 — Test the block list

```bash
# Block a user (runs redis-cli inside the Redis container)
docker exec redis redis-cli SADD blocked:users user-123

# Try sending — response will have status: BLOCKED
curl -X POST http://localhost:8080/v1/sms/send \
  -H "Content-Type: application/json" \
  -d '{"userId":"user-123","phoneNumber":"+919876543210","message":"Test"}'

# Unblock the user
docker exec redis redis-cli SREM blocked:users user-123
```

### Step 8 — Shut down

```bash
# Stop all containers
docker compose down

# Stop and also wipe stored MongoDB data
docker compose down -v
```

---

## Troubleshooting

| Problem | Fix |
|---------|-----|
| `docker compose` not found | Try `docker-compose` (with hyphen) — older Docker versions use this |
| Port 8080 or 8081 already in use | Stop whatever is using that port, or change port mapping in `docker-compose.yml` |
| SMS Sender can't connect to Kafka at startup | Kafka takes ~30 sec to be ready; Spring Boot retries automatically — just wait |
| `curl` not working on Windows | Use `Invoke-WebRequest` as shown above, or install Git Bash |
| `version` attribute warning in docker-compose | Safe to ignore — it is just an obsolete field warning |
| MongoDB flooding logs with checkpoint messages | Normal behaviour — ignore it |

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

---

### SMS Sender — `GET /v1/sms/health`

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

The block list is a Redis Set at key `blocked:users`. Use Docker exec since Redis runs inside a container:

```bash
# Block a user
docker exec redis redis-cli SADD blocked:users user-123

# Unblock a user
docker exec redis redis-cli SREM blocked:users user-123

# List all blocked users
docker exec redis redis-cli SMEMBERS blocked:users
```

---

## End-to-End Demo Script

On Mac/Linux or Git Bash on Windows:

```bash
bash demo.sh
```

The script automatically:
1. Checks health of both services
2. Sends a successful SMS for `user-demo-001`
3. Blocks `user-blocked-999` via Redis
4. Attempts to send to the blocked user (receives `BLOCKED`)
5. Sends a second SMS for `user-demo-001`
6. Waits for Kafka consumer to store events
7. Retrieves message history for `user-demo-001` from the SMS Store
8. Retrieves history for the blocked user (the BLOCKED event is also stored)
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

---

## What I Learned

I was given this project as an assignment to refactor a monolithic SMS service into two separate microservices — one in Java and one in Go. Honestly, at the start I didn't fully understand why you would split one working service into two, or why you'd bring in Kafka just to pass data between them. By the end, it started making a lot more sense.

### Microservices — why split things up?
Before this I thought microservices were just a fancy way of complicating things. But working through this made me understand the real reason — each service has one job. The Java service only cares about receiving the request, checking if the user is blocked, and calling the vendor. The Go service only cares about storing and retrieving messages. If one goes down, the other can still function. That separation felt theoretical before, but actually building it made it click.

### Kafka — I didn't get it at first
Kafka was the most confusing part initially. My first instinct was — why not just have the Java service call the Go service directly over HTTP? It would be simpler. But I slowly understood the point — if the Go service is slow or temporarily down, the Java service shouldn't have to wait or fail. Kafka sits in the middle and holds the message until the Go service is ready. The Java service publishes and moves on. That async decoupling is something I'll remember.

### Redis for the block list
This one was straightforward but satisfying. Instead of hitting a database on every single SMS request just to check if a user is blocked, we keep that list in Redis which lives in memory and responds in microseconds. I also learned about "failing open" — if Redis itself goes down, we let the request through rather than blocking all SMS sending. That's a deliberate design decision, not laziness.

### MongoDB
I had used databases before but always relational ones. Working with MongoDB here showed me how you think differently — you design your document around how you'll query it, not around normalisation rules. Creating indexes (compound index on userId + sentAt for history queries, unique index on messageId to prevent duplicates) also made me realise that schema design and performance go hand in hand even in NoSQL.

### Docker — from confusing to actually useful
I had seen Dockerfiles before but never really understood them. Writing multi-stage builds for both services — one stage to build, a smaller one to run — and then wiring everything together in docker-compose.yml gave me a real appreciation for what Docker actually solves. I also spent a good amount of time debugging build errors (the missing go.sum file being a good example) which taught me more than any tutorial would have.

### Go — a new language mid-project
I had never written Go before this. The standard library HTTP server, the explicit error handling, the interface-based design — it's very different from Java. What I liked is that Go forces you to be deliberate. There's no magic, no framework doing things behind the scenes. You write a handler, you register it, you handle errors explicitly. It felt verbose at first but I understand now why Go codebases tend to be readable.

### Spring Boot — more than just annotations
I knew Spring Boot at a surface level — put an annotation, it works. This project made me go deeper into how Kafka producers are configured (idempotence, acks=all, retries), how RedisTemplate works, and how to write a proper global exception handler so validation errors return clean JSON instead of a stack trace.

### Testing — actually writing them, not just knowing they exist
I knew unit testing was important. This project made me actually write them. Mocking dependencies in Java with Mockito, and using a stub repository in Go to test the service layer without needing a real MongoDB running — these taught me that good code is code that can be tested in isolation. The Repository interface in Go exists purely to make testing possible, and that's a legitimate architectural decision.

### The thing that surprised me most
The hardest part wasn't the code. It was understanding *why* each decision was made — why Kafka instead of HTTP, why Redis instead of a database, why fail open, why async. The code is just the expression of those decisions. That's probably the most valuable thing I'm taking away from this.
