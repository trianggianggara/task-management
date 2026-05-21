# Task Management API

Multi-user task management API with idempotency support, built with Go and Clean Architecture.

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.22+ |
| HTTP | Echo v4 |
| Database | PostgreSQL 16 |
| SQL | sqlx (raw SQL) |
| Migrations | golang-migrate |
| Auth | JWT (HS256) + bcrypt |
| Docs | Swagger (swag + echo-swagger) |
| Logging | slog (stdlib, JSON) |

## Quick Start

```bash
# Start with Docker Compose
docker compose up -d

# API available at
curl http://localhost:8080/api/v1/health

# Swagger UI
open http://localhost:8080/swagger/index.html
```

```bash
# 1. Start PostgreSQL
# 2. Configure environment
cp .env.example .env.development
# Edit .env.development with your DATABASE_URL

# 3. Run migrations
make migrate-up

# 4. Start the server
make run
```

## API Endpoints

### Auth (public)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/auth/register` | Register new user |
| POST | `/api/v1/auth/login` | Login, returns JWT |

### Tasks (protected — Bearer token)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/tasks` | Create task (requires `Idempotency-Key` header) |
| GET | `/api/v1/tasks` | List tasks (`?status=&search=&page=&limit=`) |
| GET | `/api/v1/tasks/:id` | Get task detail |
| PUT | `/api/v1/tasks/:id` | Update task (`{"status":"completed"}`) |
| DELETE | `/api/v1/tasks/:id` | Soft-delete task |
| POST | `/api/v1/tasks/:id/assign` | Assign to team member (transactional) |

### Teams (protected — Bearer token)

Seeded teams (migration `000001`):

| ID | Name |
|----|------|
| `ENG` | Engineering |
| `DSG` | Design |
| `PDT` | Product |

| Method | Path | Description |
|--------|------|-------------|
| PUT | `/api/v1/auth/team` | Join a team (`{"code": "ENG"}`) |
| DELETE | `/api/v1/auth/team` | Leave current team |

### Health

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/health` | Health check (DB ping) |

## Architecture

```
cmd/api/main.go        Entry point
internal/
├── config/            Environment configuration
├── domain/            Pure entities (User, Task, Team, TaskLog)
├── repository/        Data access contracts + PostgreSQL implementations
├── usecase/           Business logic (Auth, Task)
├── delivery/http/
│   ├── api.go           Echo setup + server lifecycle
│   ├── route.go         Route registration
│   ├── api/             HTTP handlers (auth, task)
│   └── dto/             Request/response DTOs
├── middleware/         Request ID, Logger, Auth, Rate Limiter, Error Handler
└── contract/          Dependency injection (Common, Repository, Usecase, Delivery)
pkg/utils/
├── response/          Generic API envelope + error types
├── autotx/            Database transaction manager
└── password/          Bcrypt hasher
```

## Idempotency

`POST /api/v1/tasks` requires an `Idempotency-Key: <UUID>` header. If the same key is sent within 24 hours, the original response is returned and **no duplicate task is created**. Expired keys are purged automatically every hour.

## Response Format

```json
// Success
{"success":true,"status":200,"message":"Login successful","data":{...},"request_id":"...","timestamp":"..."}

// Error
{"success":false,"status":404,"code":"NOT_FOUND","message":"task not found","timestamp":"...","request_id":"..."}

// Paginated
{"success":true,"status":200,"message":"Tasks retrieved","data":[...],"meta":{"page":1,"page_size":10,"total_items":42,"total_pages":5},"request_id":"...","timestamp":"..."}
```

## Database Design

| Table | Purpose | Key Columns |
|-------|---------|------------|
| `teams` | User groups (join by code) | id PK, code UNIQUE, name |
| `users` | Auth & ownership | id PK, email UNIQUE, password_hash, team_id FK |
| `tasks` | Task data (soft delete) | id PK, user_id FK (reporter), assignee_id FK, title, status ENUM |
| `idempotency_keys` | Safe retry (24h TTL) | key PK, user_id FK, response_body JSONB |
| `task_logs` | Audit trail | id PK, task_id FK, action, old_value JSONB, new_value JSONB |

See [docs/database-design.svg](docs/database-design.svg) for full schema with indexes and relationships.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ENVIRONMENT` | `development` | `development` or `production` |
| `DATABASE_URL` | `postgres://...` | PostgreSQL connection string |
| `JWT_SECRET` | `change-me-in-production` | JWT signing secret |
| `JWT_EXPIRY_HOURS` | `24` | Token expiry in hours |
| `APP_PORT` | `8080` | Server port |
| `DB_MAX_OPEN_CONNS` | `25` | Max open DB connections |
| `DB_MAX_IDLE_CONNS` | `5` | Max idle DB connections |
| `RATE_LIMIT_RPS` | `5` | Auth rate limit (req/sec) |
| `RATE_LIMIT_BURST` | `10` | Auth rate limit burst |

## Background Worker

Runs scheduled jobs — currently purging expired idempotency keys every hour. New jobs are added by implementing the `job.Job` interface.

```
internal/job/
├── job.go                   # Job interface + Runner
├── purge_idempotency_keys.go
└── your_new_job.go          # Add new jobs here
```

```bash
make run-worker                  # Run locally
make build-worker                # Build binary
make docker-build-worker         # Build Docker image

docker run -d --name task-worker \
  -e DATABASE_URL="postgres://user:pass@host:5432/taskmanagement?sslmode=disable" \
  task-management-worker:latest
```

## Makefile Commands

```bash
make run              # Start API server (development)
make run ENV=production  # Start API server (production)
make test             # Run all tests with race detector
make build            # Build binary
make migrate-up       # Run migrations
make migrate-down     # Rollback last migration
make migrate-create name=xxx  # Create new migration
make swagger          # Regenerate Swagger docs
make lint             # Run linter
make run-worker       # Start background worker
make build-worker     # Build worker binary
make docker-build     # Build API Docker image
make docker-build-worker  # Build worker Docker image
```

## Running Tests

```bash
# All tests (no database required)
make test

# With race detection
go test ./... -race

# Specific package
go test ./internal/usecase/ -v -run TestCreateTask_Concurrent
```

## Running Migrations Separately

```bash
# Via docker compose
docker compose run --rm app /bin/api migrate

# Or standalone binary
go run ./cmd/migration
```
