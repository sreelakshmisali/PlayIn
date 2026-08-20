# PlayHub

Sports community and turf booking platform.

**Status: Phase 2.** The scaffold, authentication, and player profiles. People
can register, sign in and build a sports profile that other players can view.
Turfs, bookings, teams and tournaments are not built yet. See
[What is not built](#what-is-not-built).

## Stack

| Part | Technology |
|---|---|
| Frontend | React 19, TypeScript 7, Vite 8, Tailwind CSS 4 |
| Backend | Go 1.25, standard library HTTP, pgx |
| Database | PostgreSQL 16 |
| Migrations | golang-migrate, run from Docker |
| Infrastructure | Docker, Docker Compose, nginx |

## Architecture

```
browser --> nginx (web:80) --> Go API (api:8080) --> PostgreSQL (postgres:5432)
              serves the SPA
              proxies /api
```

nginx serves the built bundle and forwards `/api` to the Go service, so the
browser talks to a single origin. In development the Vite dev server does the
same job with its own proxy.

The backend has four layers, each depending only on the one below it.

```
cmd/api                   composition root: reads config, builds everything, starts the server
  +- internal/server         router and HTTP server
     +- internal/middleware    request id, logging, recovery, CORS
     +- internal/httpx         JSON responses and the error envelope
     +- internal/health        one feature package: handler (thin) + service (logic)
        +- internal/database   pgx connection pool
           internal/config     environment configuration, loaded once
           internal/logging    structured logger
```

`internal/health` is the reference shape for every feature added later:
`handler.go` does HTTP only, `service.go` holds the logic and depends on narrow
interfaces. A repository layer arrives with the first feature that touches a
table, inside that feature's own package.

Full reasoning is in [docs/architecture.md](docs/architecture.md).

## Directory layout

```
PlayHub/
├── backend/            Go API
│   ├── cmd/api/          entry point
│   ├── internal/         application packages
│   └── migrations/       SQL migrations
├── frontend/           React SPA
│   └── src/
│       ├── components/   layout and shared UI
│       ├── lib/          API client and config
│       ├── pages/        route targets
│       └── routes/       route table
├── docker/nginx/       nginx site config
├── docs/               architecture notes
├── docker-compose.yml
└── .env.example
```

## Prerequisites

- Docker and Docker Compose. That is enough to run everything.
- Go 1.25 and Node 22.12+ only if you want to run a service directly on the host.

## Quick start

Everything in Docker. Three commands.

```bash
cp .env.example .env
```

Edit `.env` and set `DB_PASSWORD` to a local value.

```bash
docker compose up --build -d
```

```bash
curl http://localhost:8080/api/v1/health
```

Then open **http://localhost:3000** and go to the Status page. It calls the API
through nginx and shows what the backend reports.

`docker compose up` starts PostgreSQL, waits for it to become healthy, runs
pending migrations, then starts the API and the web server.

Stop it:

```bash
docker compose down
```

Stop it and delete the database volume:

```bash
docker compose down -v
```

## Running the parts separately

### PostgreSQL

Start only the database:

```bash
docker compose up -d postgres
```

It publishes port 5432 to the host, so the backend and the migration tool can
reach it from outside Docker. Data lives in the `playhub_postgres_data` volume
and survives `docker compose down`.

Open a psql shell:

```bash
docker compose exec postgres psql -U playhub -d playhub
```

### Migrations

Migrations run through the `migrate` service, so golang-migrate does not need to
be installed locally. Run these from the repository root.

Apply all pending migrations:

```bash
docker compose run --rm migrate up
```

Roll back the most recent migration:

```bash
docker compose run --rm migrate down 1
```

Show the current schema version:

```bash
docker compose run --rm migrate version
```

Create a new migration pair in `backend/migrations/`:

```bash
docker compose run --rm migrate create -ext sql -dir /migrations -seq add_venues
```

Conventions and the recovery procedure for a failed migration are in
[backend/migrations/README.md](backend/migrations/README.md).

### Backend

In Docker:

```bash
docker compose up -d --build api
```

On the host, with PostgreSQL already running, first create the backend env file:

```bash
cd backend && cp .env.example .env
```

Then load it and run, using bash or Git Bash:

```bash
cd backend && set -a && . ./.env && set +a && go run ./cmd/api
```

Tests:

```bash
cd backend && go test ./... -race
```

Coverage, which must stay at or above 80%:

```bash
cd backend && go test ./... -coverprofile=coverage.out -covermode=atomic && go tool cover -func=coverage.out | tail -1
```

Vet:

```bash
cd backend && go vet ./...
```

### Frontend

In Docker, served by nginx on port 3000:

```bash
docker compose up -d --build web
```

On the host with hot reload on port 5173, with the API running on 8080:

```bash
cd frontend && npm install
```

```bash
cd frontend && npm run dev
```

The dev server proxies `/api` to `VITE_DEV_API_PROXY`, which defaults to
`http://localhost:8080`. Copy `frontend/.env.example` to `frontend/.env.local`
to change it.

Type-check and build:

```bash
cd frontend && npm run build
```

Type-check only:

```bash
cd frontend && npm run typecheck
```

## API

Base path: `/api/v1`. Breaking changes get a new prefix.

### `GET /api/v1/health`

Pings PostgreSQL and reports the result.

Returns `200` when everything responds, `503` when a dependency is down.

```json
{
  "status": "ok",
  "service": "playhub-api",
  "version": "dev",
  "env": "development",
  "timestamp": "2026-08-18T09:45:12.482Z",
  "checks": {
    "database": { "status": "ok", "latency_ms": 1 }
  }
}
```

### Authentication

| Endpoint | Auth | Purpose |
|---|---|---|
| `POST /api/v1/auth/register` | none | Create an account, returns a session. `201` |
| `POST /api/v1/auth/login` | none | Exchange credentials for a session. `200` |
| `POST /api/v1/auth/refresh` | refresh token in body | Rotate the token pair. `200` |
| `POST /api/v1/auth/logout` | refresh token in body | Revoke the refresh token. `204` |
| `GET /api/v1/auth/me` | bearer access token | The signed-in user. `200` |
| `GET /api/v1/admin/ping` | bearer, ADMIN | Authorisation probe. `200` |

Register and login return the same body:

```json
{
  "user": {
    "id": "164cb079-194d-4cd0-b0ab-f9007776bc79",
    "email": "player@playhub.test",
    "full_name": "Test Player",
    "role": "PLAYER",
    "is_active": true,
    "created_at": "2026-08-18T09:52:09.636025Z"
  },
  "tokens": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "token_type": "Bearer",
    "expires_in": 900
  }
}
```

Send the access token as `Authorization: Bearer <token>`.

Roles are `PLAYER`, `OWNER` and `ADMIN`. Registration accepts `PLAYER` and
`OWNER`; `ADMIN` is granted out of band.

Refresh tokens rotate. Redeeming one revokes it, so a refresh response is the
only place the replacement exists. Logout revokes the token server side and is
idempotent.

Validation failures return `422` with a `details` array naming each field:

```json
{
  "error": {
    "code": "validation_failed",
    "message": "The request contains invalid fields.",
    "request_id": "a28e3dadec9e6f4372326832e2ffceab",
    "details": [
      { "field": "email", "message": "Email is not a valid address." },
      { "field": "password", "message": "Password must be at least 10 characters." }
    ]
  }
}
```

### Player profiles and sports

| Endpoint | Auth | Purpose |
|---|---|---|
| `GET /api/v1/sports` | none | The sports catalogue with their positions. `200` |
| `GET /api/v1/players/{userId}` | none | A player's public profile. `200` |
| `GET /api/v1/players/me` | bearer, PLAYER | Your own profile. `200`, `404` before one exists |
| `PUT /api/v1/players/me` | bearer, PLAYER | Create or replace it. `201` then `200` |
| `PATCH /api/v1/players/me` | bearer, PLAYER | Change some of it. `200` |
| `POST /api/v1/players/me/sports` | bearer, PLAYER | Add a preferred sport, or change its position. `200` |
| `DELETE /api/v1/players/me/sports/{sportId}` | bearer, PLAYER | Drop a preferred sport. `200` |

Managing a profile is PLAYER-only. An OWNER or ADMIN token is authenticated but
gets `403 forbidden`, and writes nothing.

A profile carries only profile data. No email address, role, account flag or
credential appears in any of these responses, for the owner or for a visitor:

```json
{
  "user_id": "78f01aa4-7a15-4e52-88c8-79bee9c0634f",
  "display_name": "Meera Joseph",
  "image_url": "https://cdn.example.com/meera.jpg",
  "bio": "Goalkeeper on Tuesdays, setter on Fridays.",
  "location": "Thrissur, Kerala",
  "sports": [
    { "sport": { "id": "...", "slug": "football", "name": "Football",
                 "positions": ["Goalkeeper", "Defender", "Midfielder", "Forward"] },
      "position": "Goalkeeper" },
    { "sport": { "id": "...", "slug": "badminton", "name": "Badminton", "positions": [] } }
  ],
  "created_at": "2026-08-18T11:55:56.448392Z",
  "updated_at": "2026-08-18T12:01:04.117820Z"
}
```

Six sports are seeded: Football, Cricket, Badminton, Basketball, Volleyball and
Tennis. Each carries the positions it offers; Badminton and Tennis carry none,
so a client shows a position picker for the others and omits it for those two.
A position the sport does not offer is refused with `422` and a message naming
the valid choices.

`PUT` is a full representation: a field left out is cleared. `PATCH` merges, and
distinguishes an absent key from an explicit `null`, which clears the field.

Every response carries an `X-Request-ID` header. Send your own to trace a
request across services.

Every error uses one envelope:

```json
{
  "error": {
    "code": "not_found",
    "message": "The requested resource does not exist.",
    "request_id": "3f9a1c8e2b7d4a015c6e8f2a9b3d7c41"
  }
}
```

## Configuration

All configuration comes from environment variables. `config.Load` reads them
once at startup, validates them, and reports every problem at once. No other
code calls `os.Getenv`.

`.env.example` at the root documents every variable used by Docker Compose.
`backend/.env.example` documents the full backend set, including the timeout and
pool settings that Compose leaves at their defaults.

`DB_USER`, `DB_PASSWORD` and `DB_NAME` have no defaults and the API refuses to
start without them.

`.env` files are git-ignored. Only `.env.example` is committed, and it contains
no real credentials.

## Optional: make

A `Makefile` wraps the commands above. `make help` lists them. Every target has
a plain equivalent in this README, so make is never required.

## What is built

- Root project structure: `backend/`, `frontend/`, `docker/`, `docs/`.
- Go module, entry point, and graceful shutdown on SIGINT and SIGTERM.
- Configuration package: environment-driven, validated, injected.
- PostgreSQL connection pool with startup retry and backoff.
- HTTP server with explicit read, write, idle and shutdown timeouts.
- Router on the standard library `ServeMux`, versioned under `/api/v1`.
- Middleware: panic recovery, request id, structured request logging, CORS.
- `GET /api/v1/health`, split across a thin handler and a service.
- Structured logging with `log/slog`.
- Authentication: `users` and `refresh_tokens` tables, bcrypt hashing at cost
  12, HS256 access and refresh tokens, rotation on refresh, server-side
  revocation, `RequireAuth` and `RequireRole` middleware, and per-field
  validation errors. Split across handler, service and repository.
- Player profiles: the `sports`, `player_profiles` and `player_sports` tables,
  six seeded sports with per-sport positions, profile create, replace and
  partial update, preferred sport selection, and a public profile view.
  PLAYER-only authorisation on everything that writes.
- Backend tests, 88% statement coverage. The repository tests run against a
  real PostgreSQL and skip without one.
- Vite + React + TypeScript frontend with Tailwind CSS 4.
- Route table, application layout, Home, Status, Login, Register, Account,
  Profile, Profile edit, public Player and 404 pages, mobile first.
- Session context, protected routing, and an API client that attaches the
  bearer token and refreshes once on a 401.
- Docker Compose: PostgreSQL with a persistent volume, health check and
  environment-based credentials.
- Backend and frontend Dockerfiles, both multi-stage. The backend runs as a
  non-root user.
- Migration infrastructure with a baseline migration and documented conventions.
- `.env.example` at the root and for the backend. No secrets committed.

## What is not built

Deliberately absent. Adding any of it now would mean guessing at a domain that
does not exist yet.

- Any product feature beyond player profiles: turfs, venues, bookings, slots,
  payments, teams, tournaments, search, notifications.
- Derived statistics of any kind: matches played, wins, ratings, form. They are
  computed from source data once that data exists, never stored.
- Social features: following, activity feeds, player discovery.
- Image uploads. A profile image is a URL the player supplies.
- Business tables beyond `users`, `refresh_tokens`, `sports`,
  `player_profiles` and `player_sports`.
- Email verification, password reset and rate limiting on the credential
  endpoints.
- An admin surface. `GET /api/v1/admin/ping` exists only to exercise the role
  middleware.
- Frontend state management and data-fetching libraries. The pages call the API
  client directly, which is enough at this size.
- Frontend tests. `tsc` in strict mode covers what matters at this stage.
- ESLint. `tsc` runs in strict mode and covers what matters at this stage.
- CI pipelines, observability beyond structured logs, and rate limiting.
