# dreon-notification

Notification service that sends emails (Resend) and SMS (Twilio) via gRPC. Supports template-based messages for channels EMAIL and SMS.

## Features

- **Email** – MJML templates (welcome, verify-otp, forgot-password, reset-password) rendered to HTML and sent via [Resend](https://resend.com).
- **SMS** – Text templates (e.g. verify-otp) sent via [Twilio](https://www.twilio.com). Uses a mock client when Twilio is not configured.
- **gRPC API** – `SendNotification` with channel (EMAIL, SMS, PUSH, IN_APP), type, recipients, and params.
- **HTTP** – Health or admin endpoints (if configured).

## Prerequisites

- Go 1.25+
- Redis (for cache)
- PostgreSQL (for notification records)
- Resend API key (for email)
- Twilio account (optional; for SMS)

## Setup

1. Clone and install dependencies:

```bash
go mod download
```

2. Copy env example and set values:

```bash
cp .env.example .env
# Edit .env with your Redis, Postgres, Resend, and optionally Twilio credentials.
```

3. Run:

```bash
go run .
```

- HTTP server: `http://localhost:8080` (see `HTTP_HOST`, `HTTP_PORT`)
- gRPC server: `localhost:9090` (see `GRPC_PORT`)

## Environment Variables

| Group      | Variable                    | Description |
|-----------|-----------------------------|-------------|
| App       | `APP_NAME`, `APP_VERSION`  | Application identity |
| Server    | `HTTP_HOST`, `HTTP_PORT`, `GRPC_PORT` | Listen addresses |
| Redis     | `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`, `REDIS_DB` | Cache |
| Postgres  | `POSTGRES_*`               | Database connection |
| Email     | `EMAIL_SENDER`             | From address (Resend verified domain or onboarding@resend.dev) |
| Email     | `EMAIL_TEMPLATE_DIR`       | MJML template directory (default: `templates/email`) |
| Email     | `EMAIL_RESEND_API_KEY`     | Resend API key |
| SMS       | `TWILIO_ACCOUNT_SID`       | Twilio Account SID (optional) |
| SMS       | `TWILIO_AUTH_TOKEN`        | Twilio Auth Token (optional) |
| SMS       | `TWILIO_FROM_NUMBER`       | Twilio sender number, E.164 (e.g. +15551234567) (optional) |
| SMS       | `SMS_TEMPLATE_DIR`         | SMS text template directory (default: `templates/sms`) |
| Logging   | `LOG_LEVEL`                | Log level (e.g. info, debug) |

If `TWILIO_ACCOUNT_SID` and `TWILIO_AUTH_TOKEN` are not set, the service uses a mock SMS client; SMS send requests will fail with "client not configured".

## Project Structure

```
.
├── config/                 # App config and env loading
├── internal/
│   ├── aggregate/          # Request/response DTOs
│   ├── model/              # Domain models
│   ├── repository/         # Data access
│   ├── service/            # Notification business logic
│   ├── shared/constant/    # Template maps (email, SMS)
│   └── errorx/             # Error types
├── pkg/
│   ├── email/              # Email client (Resend), MJML renderer
│   ├── sms/                # SMS client (Twilio), body renderer, mock
│   ├── logger/             # Logger interface and impl
│   ├── cache/              # Redis cache
│   ├── database/           # Postgres client
│   └── validator/          # Validation
├── presentation/
│   ├── grpc/               # gRPC server and proto
│   └── http/               # HTTP server
├── templates/
│   ├── email/              # MJML templates (.mjml)
│   └── sms/                # SMS text templates (.txt)
└── examples/               # Example scripts (e.g. send email)
```

## Notification Types and Templates

- **EMAIL**: `WELCOME`, `VERIFY_OTP`, `FORGOT_PASSWORD`, `RESET_PASSWORD` → MJML templates in `templates/email/`.
- **SMS**: `VERIFY_OTP` (and others as added) → text templates in `templates/sms/`.

Template name mapping is in `internal/shared/constant/template.go`.

## Testing

```bash
go test ./...
```

## License

Private / internal use.
