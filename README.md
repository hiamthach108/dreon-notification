# dreon-notification

Notification service (MVP): send emails (Resend) and SMS (Twilio) via gRPC, with async processing over RabbitMQ.

## Features

- **gRPC API** – Create/send notifications; enqueue for async send. Channel: EMAIL, SMS (PUSH/IN_APP stubbed).
- **Async primary queue** – Enqueue notification → RabbitMQ topic `notifications.send` (Watermill/AMQP) → consumer sends. Messages are always acked; failed sends increment `attempt_count`, set exponential **backoff** via `next_retry_at`, and mark **FAILED** when `attempt_count` reaches `max_attempts`.
- **Retry queue (multi-pod friendly)** – A background worker periodically **claims** due `PENDING` rows in Postgres (`FOR UPDATE SKIP LOCKED`), extends `next_retry_at` by a short **publish lease**, and publishes lightweight messages to `notifications.retry`. Any pod’s consumer can process retries; idempotency relies on DB state (`PENDING` / `COMPLETED` / `FAILED`).
- **Email** – MJML templates (welcome, verify-otp, forgot-password, reset-password) via [Resend](https://resend.com).
- **SMS** – Text templates via [Twilio](https://www.twilio.com); mock client when Twilio is not configured.
- **HTTP** – Health/admin endpoints.

## Prerequisites

- Go 1.25+
- Redis (cache)
- PostgreSQL (notification records)
- RabbitMQ (queue; e.g. `docker-compose up -d rabbitmq`)
- Resend API key (email)
- Twilio (optional; for SMS)

## Setup

1. Install dependencies:

```bash
go mod download
```

2. Copy env and set values:

```bash
cp .env.example .env
# Set Redis, Postgres, RabbitMQ, Resend; optionally Twilio.
```

3. Start RabbitMQ (if using Docker):

```bash
docker-compose up -d rabbitmq
```

Use `RABBITMQ_URL=amqp://guest:guest@127.0.0.1:5672/` in `.env` when running the app on your host (avoids IPv6 connection issues).

4. Run the app:

```bash
go run .
```

- HTTP: `http://localhost:8080` (`HTTP_HOST`, `HTTP_PORT`)
- gRPC: `localhost:9090` (`GRPC_PORT`)

## Environment Variables

| Group         | Variable | Description |
|---------------|----------|-------------|
| App           | `APP_NAME`, `APP_VERSION` | Application identity |
| Server        | `HTTP_HOST`, `HTTP_PORT`, `GRPC_PORT` | Listen addresses |
| Redis         | `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`, `REDIS_DB` | Cache |
| Postgres      | `POSTGRES_*` | Database connection |
| RabbitMQ      | `RABBITMQ_URL` | AMQP URL. Use `127.0.0.1` on host; `rabbitmq` in Docker. |
| Notification  | `NOTIFICATION_MAX_ATTEMPTS` | Max send attempts before status `FAILED` (default in code: 3). |
| Notification  | `NOTIFICATION_RETRY_INTERVAL_SEC` | How often the retry scheduler runs (default: 60). |
| Notification  | `NOTIFICATION_RETRY_BATCH_SIZE` | Max rows claimed per tick (default: 10). |
| Notification  | `NOTIFICATION_RETRY_BACKOFF_INITIAL_SEC` | Base backoff after a failure in seconds (default: 30). |
| Notification  | `NOTIFICATION_RETRY_BACKOFF_MAX_SEC` | Backoff cap in seconds (default: 3600). |
| Notification  | `NOTIFICATION_RETRY_PUBLISH_LEASE_SEC` | After claim, `next_retry_at` is bumped by this lease so other pods do not double-enqueue the same row until the lease expires or the consumer updates the row (default: 300). |
| Email         | `EMAIL_SENDER`, `EMAIL_RESEND_API_KEY`, `EMAIL_TEMPLATE_DIR` | Resend |
| SMS           | `TWILIO_ACCOUNT_SID`, `TWILIO_AUTH_TOKEN`, `TWILIO_FROM_NUMBER`, `SMS_TEMPLATE_DIR` | Twilio (optional) |
| Logging       | `LOG_LEVEL` | e.g. info, debug |

If Twilio is not set, a mock SMS client is used.

## Project Structure

```
.
├── config/                    # App config, env loading
├── internal/
│   ├── aggregate/             # DTOs (e.g. SendNotificationReq, enqueue/retry payloads)
│   ├── model/                 # Domain models
│   ├── repository/            # Data access (transactions, locking, RecordSendFailure SQL)
│   ├── service/               # Notification logic; publish to AMQP from service layer
│   ├── shared/constant/       # Template maps, event topics
│   └── errorx/                # Error types
├── pkg/
│   ├── email/                 # Resend client, MJML renderer
│   ├── sms/                   # Twilio + mock, body renderer
│   ├── logger/                # Logger (Zap)
│   ├── cache/                 # Redis
│   ├── database/              # Postgres
│   └── validator/
├── presentation/
│   ├── events/                # AMQP: LoggerAdapter, publisher/subscriber, router, consumers
│   ├── worker/                # Retry scheduler ticker (enqueue due rows to retry topic)
│   ├── grpc/                  # gRPC server, NotiInternal
│   └── http/                  # HTTP server
├── templates/
│   ├── email/                 # MJML (.mjml)
│   └── sms/                   # Text (.txt)
└── examples/
```

## Queue and Retry Behavior

### Topics

| Topic | Role |
|-------|------|
| `notifications.send` | Initial async send after enqueue. Payload includes full `SendNotificationReq`. |
| `notifications.retry` | Retry attempts for rows that already failed at least once and are still `PENDING` under `max_attempts`. Payload is `{ "notificationId": "..." }`; the consumer loads the row from Postgres. |

Constants live in `internal/shared/constant/event.go`.

### Primary send (`notifications.send`)

- Consumer: `ProcessNotificationFromQueue`.
- On success: `COMPLETED`, `sent_at` set.
- On failure: `RecordSendFailure` atomically increments `attempt_count`, may set `FAILED` if attempts are exhausted, otherwise sets `next_retry_at` using exponential backoff: `min(initial × 2^attempt_count, max)` (see repository SQL).

### Retry scheduler and consumer

- **Scheduler** (`presentation/worker/retry.go`): on an interval, runs `EnqueuePendingRetries`, which opens a transaction, selects eligible rows with `SKIP LOCKED`, sets `next_retry_at` to **now + publish lease**, commits, then publishes one message per row to `notifications.retry`.
- **Retry consumer**: `ProcessNotificationRetryFromQueue` loads the notification by ID; if it is no longer a valid retry target, the message is acked without sending. Otherwise it sends like the primary path and updates success or calls `RecordSendFailure` again.

Messages are **always acked** so the broker does not block; correctness depends on DB state and the lease/backoff fields.

Ensure RabbitMQ topology binds both topics like your existing `notifications.send` setup (Watermill durable queue config).

## Notification Types and Templates

- **EMAIL**: `WELCOME`, `VERIFY_OTP`, `FORGOT_PASSWORD`, `RESET_PASSWORD` → `templates/email/` (MJML).
- **SMS**: e.g. `VERIFY_OTP` → `templates/sms/` (text). Mapping in `internal/shared/constant/template.go`.

## Testing

```bash
go test ./...
```

Some service tests use an in-memory SQLite database to exercise transactional retry claim logic; production uses PostgreSQL.

## License

Private / internal use.
