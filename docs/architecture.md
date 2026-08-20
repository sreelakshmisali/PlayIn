# Architecture

Phase 0 scaffold, Phase 1 authentication, Phase 2 player profiles. This
document describes what exists and why it is shaped the way it is. It does not
describe planned features.

## Shape

```
browser ──▶ nginx (web) ──▶ Go API ──▶ PostgreSQL
              serves the SPA
              proxies /api
```

In development the Vite dev server replaces nginx and proxies `/api` to the API
on the host. The browser sees one origin in both cases, so CORS is only a
fallback path, not the normal one.

## Backend layers

Four layers, each depending only on the one below it.

| Layer | Location | Responsibility |
|---|---|---|
| Entry point | `cmd/api` | Composition root. Reads config, builds every dependency, starts the server. |
| Transport | `internal/server`, `internal/middleware`, `internal/httpx` | Routing, cross-cutting HTTP concerns, response shapes. |
| Feature | `internal/health`, `internal/auth`, `internal/players` | Handler, service, repository. One package per bounded feature. |
| Infrastructure | `internal/config`, `internal/database`, `internal/logging` | Configuration, connection pool, logger. |

`internal/health` is the reference shape for every feature package added later:

- `service.go` holds the logic and depends on narrow interfaces, not concrete types.
- `handler.go` translates HTTP to service calls and back. It holds no logic.
- `Routes()` registers the package's own endpoints, so the router stays a list
  of mounts rather than a growing list of paths.

`internal/auth` is the first package to add the third file the shape allows:

- `repository.go` is the only place in the package that writes SQL. It reports
  storage facts (`ErrEmailTaken`, `ErrUserNotFound`) and decides nothing.
- The service depends on a `Store` interface declared in `service.go`, not on
  `*Repository`. The dependency points inward, so the rules are testable
  without PostgreSQL and the repository is free to change shape.

## Decisions

**Standard library router.** Go 1.22 added method and wildcard patterns to
`http.ServeMux`, which covers everything the API needs. A third-party router
would be a dependency with no matching benefit.

Registering `"/"` for a JSON 404 shadows ServeMux's built-in 405 handling, so
each path is registered twice: once with its method, once without. The
method-less pattern answers unsupported methods with a JSON 405.

**pgx over database/sql.** `pgxpool` is the PostgreSQL-native driver. It gives
better type handling and a purpose-built pool. The application is not going to
target another database engine.

**No ORM.** Migrations are the schema's source of truth and queries are written
as SQL. This keeps generated queries out of the critical path and makes the
index strategy visible.

**Config is loaded once, in one place.** `config.Load` reads the environment,
validates it, and reports every problem at once. Nothing else calls
`os.Getenv`, so no component can quietly depend on an undeclared variable.

**Errors are a single JSON envelope.** Every failure returns
`{"error": {"code", "message", "request_id"}}`. Clients branch on `code`, not on
message text. `internal/httpx` is the only place that writes a response body.

**Correlation ids.** `X-Request-ID` is accepted from the caller or generated,
carried on the request context, attached to every log line, echoed on the
response, and included in error bodies.

**Middleware order.** Recovery is outermost so it catches panics from everything
below it. Request id runs before the logger so every line carries a correlation
id. CORS is innermost of the four, closest to the routes.

**Graceful shutdown.** SIGINT and SIGTERM cancel the root context. In-flight
requests drain within `HTTP_SHUTDOWN_TIMEOUT` before the process exits.

**Database retry at startup.** PostgreSQL is often still starting when the API
container comes up, so `ConnectWithRetry` backs off instead of leaving the
container runtime to crash-loop the process.

**Health returns 503 when degraded.** The endpoint pings PostgreSQL. A failed
dependency produces `status: "degraded"` and HTTP 503, so orchestrators can act
on the status code without parsing the body.

## Authentication

**Two token types, one signing key.** The access token is a short-lived HS256
JWT carrying the subject and role. The refresh token is also a JWT, but its
`jti` is a row in `refresh_tokens`, which is what makes it revocable. Every
token carries a `kind` claim and every verifier states the kind it expects, so
a month-long refresh token cannot be presented as an access token.

**Rotation on refresh.** Redeeming a refresh token revokes it before the
replacement is issued, so a stolen token is single use: whichever party redeems
it first invalidates it for the other.

**The account is read on every authenticated request.** `Authenticate` loads
the user rather than trusting the claims, so a deactivated or deleted account
stops working immediately instead of at the end of the access token lifetime.
It costs one indexed primary key lookup.

**Login failures are indistinguishable.** An unknown email and a wrong password
return the same code and message, and the unknown-email path still verifies
against a decoy hash so it costs the same. Without both, the endpoint is an
account enumeration oracle.

**ADMIN is not self-assignable.** Registration accepts PLAYER and OWNER.
ADMIN is granted out of band.

**Guards are middleware, not handler code.** `RequireAuth` and `RequireRole`
wrap routes at the mount point, so a protected route cannot be added without
its check. `RequireRole` refuses a request that arrives without a principal
rather than letting it through, because that combination is a wiring mistake
and the safe reading of a mistake is denial.

**`JWT_SECRET` has no default.** An application that falls back to a built-in
signing key issues forgeable tokens the moment it reaches an environment nobody
configured, so the API refuses to start without one.

**Passwords are bcrypt at cost 12.** Input is capped at 72 bytes in validation
because bcrypt silently truncates there, which would otherwise make two long
passwords sharing a prefix interchangeable.

## Player profiles

**Positions are an array on the sport, not a fourth table.** A position has no
attributes of its own and exists only as a choice one sport offers. Membership
is still enforced by the database rather than by Go: `player_sports` rows are
written by `INSERT ... SELECT ... FROM sports WHERE id = $2 AND ($3 = '' OR $3 =
ANY(s.positions))`, so an invalid position inserts zero rows against the live
sport row, not against a copy the service read earlier.

Badminton and Tennis seed with an empty array. That is what makes "position
where applicable" a property of the data instead of a rule the client has to
know.

**One profile per account, enforced by a unique index.** `PUT /players/me` is an
upsert on `user_id`, so two concurrent first-time saves cannot both create a
profile: the loser updates. `xmax = 0` in the RETURNING clause distinguishes the
insert from the update, which is what the handler answers 201 rather than 200
on.

**PUT replaces, PATCH merges.** A partial update needs three states from each
field, and a `*string` only carries two: absent and null both decode to nil.
`httpx.Optional[T]` records whether the key was present, so `{"bio": null}`
clears the bio and a body without the key leaves it alone.

**The response shape carries no account data.** `Profile` is built from the
profile tables alone; the repository projection never joins `users`. There is no
field an owner sees that a stranger does not, which is why one handler serves
both `/players/me` and `/players/{userId}`. The profile's own primary key is not
exposed either: callers address a player by user id, the identifier the rest of
the API already uses.

**Image URLs are restricted to http and https**, in the validator and again in a
CHECK constraint. A `javascript:` or `data:` URL stored here becomes stored
cross-site scripting the first time it is rendered into an attribute.

**Nothing derived is stored.** Matches played, wins and ratings are computed
from source data when that data exists. A column holding them now would be a
column guaranteed to drift.

**Sports are retired, not deleted.** `player_sports.sport_id` references them
with `ON DELETE RESTRICT`, because deleting a sport players have chosen would
silently edit their profiles. `is_active` is the supported way to withdraw one.

**Routing note.** `GET /players/{userId}` and a method-less `/players/me` cannot
both be registered: against each other neither pattern is more specific, one
being narrower by method and the other by path, and ServeMux rejects the pair at
registration. The single method-less `/players/{userId}` covers the 405 case for
both.

## Frontend

- **`src/lib/api.ts`** is the only module that calls `fetch`. It attaches the
  bearer token, and on a 401 it refreshes once and retries. Concurrent 401s
  share one in-flight refresh, because rotation would invalidate all but one of
  several parallel attempts.
- **`ApiError`** carries the HTTP status and the server's error code, so pages
  branch on structured fields instead of matching strings.
- **`src/lib/config.ts`** is the only module that reads `import.meta.env`.
- **Routing** uses a single route table in `src/routes/router.tsx`. Every route
  is a child of `AppLayout`, so the shell mounts once and pages swap inside it.
- **Path alias `@/`** maps to `src/`, which keeps imports stable when files move.
- **Tailwind v4** is configured in CSS. Design tokens live in the `@theme` block
  in `src/index.css`, so a colour change is one edit in one file.
- **`src/lib/tokens.ts`** is the only module that touches storage, so moving
  from localStorage to cookies later is one file.
- **`AuthProvider`** resolves stored tokens against `GET /auth/me` on mount
  rather than trusting them, and treats "still checking" as a state distinct
  from "signed out" so a hard refresh does not bounce a signed-in user to the
  login screen.
- **`ProtectedRoute`** is a layout route. Pages are grouped behind it rather
  than each checking for itself. It takes a `roles` list, so the profile screens
  refuse an OWNER or ADMIN in the same way the API does.
- **Sports selection writes per change** rather than batching behind a save
  button, because the API has no endpoint that replaces the whole list. A
  client-side diff to simulate one would only add a way for the two to disagree.

## Testing

Unit tests run without infrastructure. The auth repository tests are the
exception: what they check is the schema itself, the unique index on email, the
CHECK constraints, the cascade from users to refresh tokens, and a fake would
only re-test the fake. They skip unless `PLAYHUB_TEST_DATABASE_URL` points at a
migrated database:

```bash
PLAYHUB_TEST_DATABASE_URL=postgres://playhub:pass@localhost:5432/playhub?sslmode=disable go test ./...
```

## Deliberate omissions

Business tables, domain models and every product feature beyond player
profiles: turfs, slots, bookings, teams, tournaments and payments.

Derived statistics of any kind: matches played, wins, ratings, form.

Social features: following, activity feeds, search and discovery beyond a
direct link to a profile.

Image uploads. A profile image is a URL the player supplies; there is no
storage, no resizing and no content validation beyond the scheme check.

Also absent, and deliberately: email verification, password reset, rate
limiting on the credential endpoints, and an admin surface beyond the single
`GET /api/v1/admin/ping` probe that exists to exercise `RequireRole`.
