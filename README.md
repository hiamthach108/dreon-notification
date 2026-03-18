# dreon-notification

Notification service (MVP): send emails (Resend) and SMS (Twilio) via gRPC, with async processing over RabbitMQ.

## Features

- **gRPC API** – Create/send notifications; enqueue for async send. Channel: EMAIL, SMS (PUSH/IN_APP stubbed).
- **Async queue** – Enqueue notification → RabbitMQ (Watermill/AMQP) → consumer processes and sends. Messages are always committed (ack); failed sends are marked `Failed` in DB for later handling/retry.
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

| Group     | Variable                         | Description |
|----------|----------------------------------|-------------|
| App      | `APP_NAME`, `APP_VERSION`       | Application identity |
| Server   | `HTTP_HOST`, `HTTP_PORT`, `GRPC_PORT` | Listen addresses |
| Redis    | `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`, `REDIS_DB` | Cache |
| Postgres | `POSTGRES_*`                     | Database connection |
| RabbitMQ | `RABBITMQ_URL`                   | AMQP URL. Use `127.0.0.1` on host; `rabbitmq` in Docker. |
| Email    | `EMAIL_SENDER`, `EMAIL_RESEND_API_KEY`, `EMAIL_TEMPLATE_DIR` | Resend |
| SMS      | `TWILIO_ACCOUNT_SID`, `TWILIO_AUTH_TOKEN`, `TWILIO_FROM_NUMBER`, `SMS_TEMPLATE_DIR` | Twilio (optional) |
| Logging  | `LOG_LEVEL`                      | e.g. info, debug |

If Twilio is not set, a mock SMS client is used.

## Project Structure

```
.
├── config/                    # App config, env loading
├── internal/
│   ├── aggregate/             # DTOs (e.g. SendNotificationReq, enqueue payload)
│   ├── model/                 # Domain models
│   ├── repository/            # Data access
│   ├── service/               # Notification logic; publish/subscribe (AMQP) in service layer
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
│   ├── events/                # AMQP: LoggerAdapter, publisher/subscriber from config, router, RunRouter (topic subscriptions)
│   ├── grpc/                  # gRPC server, NotiInternal
│   └── http/                  # HTTP server
├── templates/
│   ├── email/                 # MJML (.mjml)
│   └── sms/                   # Text (.txt)
└── examples/
```

## Queue Behavior (MVP)

- **Topic**: `notifications.send` (see `internal/shared/constant/event.go`).
- **Publish**: Service enqueues by publishing to AMQP; consumer runs in the same process (Watermill router).
- **Consume**: One subscriber; `RunRouter` registers topic → handler (service `ProcessNotificationMessage`). Messages are **always acked** (autocommit). On send failure, notification status is set to `Failed` and the message is still acked so the queue keeps moving; failed notifications can be retried or handled later via DB.

## Notification Types and Templates

- **EMAIL**: `WELCOME`, `VERIFY_OTP`, `FORGOT_PASSWORD`, `RESET_PASSWORD` → `templates/email/` (MJML).
- **SMS**: e.g. `VERIFY_OTP` → `templates/sms/` (text). Mapping in `internal/shared/constant/template.go`.

## Testing

```bash
go test ./...
```

## License

Private / internal use.
