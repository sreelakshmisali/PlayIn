# PlayHub

Sports community and turf booking platform.

**Status: Phase 2.** The scaffold, authentication, and player profiles. People
can register, sign in and build a sports profile that other players can view.
Turfs, bookings, teams and tournaments are not built yet. See
[What is not built](#what-is-not-built).

## Stack

| Part | Technology |
|---|---|
| Mobile app | React Native, Expo (managed), TypeScript — **the primary player/owner client** |
| Web frontend | React 19, TypeScript 7, Vite 8, Tailwind CSS 4 — being superseded by mobile; see [Mobile app](#mobile-app-react-native--expo) |
| Backend | Go 1.25, standard library HTTP, pgx |
| Database | PostgreSQL 16 |
| Migrations | golang-migrate, run from Docker |
| Infrastructure | Docker, Docker Compose, nginx |

## Architecture

```
React Native app (Expo)  --\
                             +--> Go API (api:8080) --> PostgreSQL (postgres:5432)
browser --> nginx (web:80) -/
              serves the web SPA (admin; player/owner web being retired)
              proxies /api
```

The mobile app and the web frontend are two independent clients of the same
Go REST API under `/api/v1`; neither talks to PostgreSQL directly, and the
API and database are unchanged by the mobile app's existence. See
[Mobile app](#mobile-app-react-native--expo) for how the mobile app reaches
the API from an emulator or a physical device, where `localhost` does not
mean what it means in a browser.

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
├── mobile/             React Native + Expo app — the primary client
│   ├── app/               root composition (providers, RootNavigator)
│   ├── components/        shared UI primitives
│   ├── screens/            auth/, public/, player/, owner/ screens
│   ├── navigation/        AuthNavigator, PlayerNavigator, OwnerNavigator
│   ├── services/          typed API client and per-domain functions
│   ├── hooks/             AuthProvider / useAuth
│   ├── types/             shared request/response types
│   ├── storage/           SecureStore-backed token storage
│   └── theme/             colours, spacing, typography
├── frontend/           React SPA — see "Mobile app" for its current status
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

## Mobile app (React Native + Expo)

`mobile/` is the primary PlayHub client for players and owners. It is an
Expo-managed React Native app in TypeScript, talking to the same Go API under
`/api/v1` that the web frontend uses — it does not add, duplicate or change
any backend endpoint.

### Install and start

```bash
cd mobile && npm install
```

```bash
cd mobile && npm start
```

This starts the Expo dev server (Metro) and prints a QR code. From there,
press `a` for Android, `i` for iOS (macOS only), `w` for web, or scan the QR
code with the Expo Go app on a physical device.

### Run on Android

An emulator, already running:

```bash
cd mobile && npm run android
```

A physical Android device: install **Expo Go** from the Play Store, run
`npm start`, and scan the terminal's QR code — the device must be on the same
Wi-Fi network as the development machine (see API URL below).

### Configuring the API URL

The API's base URL is never hardcoded. It is read from the `EXPO_PUBLIC_API_URL`
environment variable at build/start time (`mobile/services/config.ts`), which
Expo inlines automatically — no extra config library involved. Copy the
example file and edit it:

```bash
cd mobile && cp .env.example .env
```

**`localhost` means something different depending on where the app runs, and
getting this wrong is the most common reason "it works on my machine" fails
on a phone:**

| Target | `EXPO_PUBLIC_API_URL` | Why |
|---|---|---|
| Android emulator | `http://10.0.2.2:8080` (the built-in default — works with no `.env` at all) | `10.0.2.2` is the emulator's own alias for the host machine's `localhost`; the emulator is a separate virtual device, so its `localhost` means itself |
| iOS Simulator | `http://localhost:8080` | The Simulator shares the Mac's network stack, so this one case is the exception |
| Physical device (Expo Go or a dev build) | `http://<your-machine's-LAN-IP>:8080` | The phone is a separate device on the network; `localhost` on it means the phone itself, not your computer |

**To find your development machine's LAN IP:**

- Windows: `ipconfig` — use the `IPv4 Address` under your active adapter (Wi-Fi or Ethernet).
- macOS/Linux: `ifconfig` or `ip addr` — look for the address on your active network interface (often `en0` or `wlan0`).

The phone and the development machine must be on the same network, and the
Go API must actually be reachable on that IP and port — the Docker Compose
`api` service already publishes `8080` to the host, so this is normally just
a firewall check away from working.

### Verifying and building

```bash
cd mobile && npm run typecheck
```

```bash
cd mobile && npx expo-doctor
```

`expo-doctor` checks the whole project setup (config, dependencies, native
module compatibility) without needing a device. To produce and sanity-check
an actual JS bundle for a platform:

```bash
cd mobile && npx expo export --platform android
```

### Architecture

The app follows the same layered discipline as the Go backend: screens never
call `fetch` directly.

```
screens/   -- one screen per route, owns its own local state and loading/error/empty UI
  uses  hooks/useAuth()          -- session: user, status, login, register, logout
  uses  services/*.ts            -- one typed function per API call (auth, players, owners)
        services/api.ts          -- the one place that builds requests: headers, JSON,
                                     the ApiError type, and automatic refresh-and-retry on 401
        storage/tokens.ts        -- expo-secure-store, never AsyncStorage or any
                                     unencrypted store, for the access/refresh token pair
navigation/ -- RootNavigator picks Auth / Player / Owner by session status and role
```

`services/api.ts` is the mobile equivalent of the web app's `lib/api.ts`: a
single `api.get/post/put/patch/delete` surface, a typed `ApiError` with
`.status`, `.code` and `.fieldErrors()`, and a single-flight refresh so N
requests that all hit a 401 at once only trigger one `/auth/refresh` call.
The one structural difference from the web client is that mobile token
storage (`expo-secure-store`) is asynchronous, so every read goes through
`await readTokens()` rather than a synchronous `localStorage` call.

### Mobile UX notes

- `components/Screen.tsx` is the one place every screen gets safe-area insets,
  optional scrolling, and (for forms) `KeyboardAvoidingView` so the keyboard
  never covers the field being typed into.
- Every list (turfs, an owner's own turfs) is a `FlatList` with pull-to-refresh,
  not a `ScrollView` of mapped items.
- Every screen that loads data has an explicit loading, error and empty state;
  none of them silently render nothing.
- `components/Button.tsx` enforces a minimum 44pt touch target on every
  button, regardless of label length.

### Known limitations of this foundation phase

- No turf slot/availability screens yet. Slot and availability endpoints exist
  only on the not-yet-merged `feature/slot-availability` branch; this mobile
  branch was started from `main` deliberately (see the git log), so those
  endpoints are not present in the backend this branch runs against. The
  typed client and screens for this can be added once that branch merges.
  It was not in this phase's required screen list.
  ADMIN accounts see a "not on mobile yet" screen (no admin UI in this phase).
- No sports picker on the player profile screen (add/remove a preferred
  sport). The profile screen displays existing preferred sports; editing them
  is deferred to keep this foundation phase's scope to what proves the
  architecture.
- No turf sports/amenities/photo management on the owner turf form — only the
  core turf fields (name, address, city, hours, capacity). The web app's
  richer editor for these sub-resources was not ported.
- Public guest browsing (turfs visible before signing in) is not implemented;
  the three navigation flows are exactly auth / player / owner, matching the
  brief. A future phase can add a guest-accessible public stack if wanted.
- Not tested on a physical device or Android emulator in this environment —
  none was available. What *was* verified: `tsc --noEmit`, `expo-doctor`
  (21/21 checks), a real Metro bundle build for Android (`expo export` and a
  live bundle request against a running dev server), and every API contract
  the app's `services/` layer depends on (register, login, `/auth/me`,
  refresh-token rotation, unauthorized handling, player profile, owner
  profile, turf creation) exercised directly against the real Go API with the
  exact request/response shapes the TypeScript types declare.

## Web frontend: current status

`frontend/` is **not deleted and not modified** by the mobile migration. It
remains in the repository, working, for two reasons:

1. **It is still the admin web interface.** Turf moderation and user
   management (`/admin/...`) are explicitly out of scope for React Native in
   this phase (see CLAUDE.md) and have no mobile equivalent — `frontend/` is
   the only place they exist.
2. Its player- and owner-facing pages are now **superseded, not removed**.
   They still build and run, but React Native + Expo (`mobile/`) is the
   approved primary client for those two roles going forward; new
   player/owner work belongs in `mobile/`, not `frontend/`.

This is a deliberate, temporary state, not an oversight: `frontend/` continues
to serve admin needs today, while carrying player/owner pages that are no
longer the primary surface for those roles. A future phase should decide
between trimming `frontend/` down to an admin-only app or replacing it
entirely; that decision is out of scope here and is left open on purpose.

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
