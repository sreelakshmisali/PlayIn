# PlayHub

PlayHub is a production-quality **mobile-first sports community and turf booking platform**.

## Stack

- Frontend: React + TypeScript + Vite + Tailwind CSS
- Backend: Go + REST API
- Database: PostgreSQL
- Infrastructure: Docker + Docker Compose
- API prefix: `/api/v1`

## Product

Players can discover/book turfs, create teams, join tournaments, and build sports profiles from real participation.

Turf owners can manage turfs, slots, bookings, and tournaments.

Admins can approve/manage turfs, users, and platform content.

## Critical Rules

- Mobile-first. Design for mobile screens first; desktop is an enhancement.
- Keep handlers/controllers thin.
- Business logic belongs in services/use-cases.
- Database access belongs in repositories/data-access.
- Use dependency injection where appropriate.
- PostgreSQL schema must use migrations.
- Use UUIDs and consistent timestamps.
- Add proper foreign keys, unique constraints, and indexes.
- Derive statistics from source data; do not duplicate derived values.
- Use environment variables for configuration/secrets.
- Never commit secrets.
- Use structured logging and proper error handling.
- Reuse existing architecture before introducing new abstractions.
- Do not add dependencies, technologies, or features without a clear need.
- Do not introduce microservices, Redis, GraphQL, Kubernetes, or an ORM unless explicitly approved.
- Do not rewrite working code unnecessarily.
- Inspect existing code before modifying it.
- Do not implement features outside the requested phase.

## Development

Build features incrementally and keep the application runnable.

Before finishing a task:
1. Run relevant tests/builds.
2. Verify the feature works end-to-end where practical.
3. Report changed files and verification results.
4. Clearly state what is complete and what remains.

## Mobile-First Product Rule

PlayHub is a mobile-first application.

- Mobile UI is the primary product experience.
- Design and implement mobile screens first.
- Test important flows at 375x812 or similar mobile viewport.
- Desktop layouts must not compromise the mobile experience.
- Do not treat mobile as a responsive afterthought.

## Git Workflow

Every completed feature must be isolated in its own Git branch.

Before implementing a feature:
1. Inspect current git status and branch.
2. Create a dedicated feature branch from the current approved branch.
3. Implement only the requested feature.

After the feature is complete:
1. Run relevant tests/builds.
2. Verify the feature end-to-end.
3. Commit the completed feature with a clear conventional commit message.
4. Show the branch name, commit hash, changed files, and verification results.
5. STOP and ask for permission to merge.

Never merge a feature branch without explicit user approval.

Do not create or switch branches unnecessarily.
Do not commit unrelated changes.
Do not push to a remote unless explicitly requested.

## Current Status

Phase 0, Phase 1, Phase 2 and Phase 3 are complete.

Phase 0 includes the project foundation, Go API, React frontend, PostgreSQL, Docker Compose, migrations, API client, routing, health endpoint, logging, and basic middleware.

Phase 1 includes authentication: the `users` and `refresh_tokens` tables, PLAYER/OWNER/ADMIN roles, registration, login, bcrypt password hashing, JWT access and refresh tokens with rotation and revocation, authentication and role middleware, `GET /api/v1/auth/me`, and the login, registration and account screens with protected routing.

Phase 2 includes player profiles and sports: the `sports`, `player_profiles` and `player_sports` tables with the six seeded sports, per-sport positions, `GET /api/v1/sports`, `GET|PUT|PATCH /api/v1/players/me`, `POST|DELETE /api/v1/players/me/sports`, the public `GET /api/v1/players/{userId}`, PLAYER-only authorisation on profile management, and the profile, profile edit and public player screens.

Phase 3 includes owner profiles and turf management: the `owner_profiles`, `turfs`, `amenities`, `turf_sports`, `turf_amenities` and `turf_images` tables with the six seeded amenities, turf statuses (`DRAFT`, `PENDING_APPROVAL`, `APPROVED`, `REJECTED`, `SUSPENDED`), `GET|PUT|PATCH /api/v1/owners/me`, owner turf CRUD under `/api/v1/owners/me/turfs`, the `/submit` status transition, turf sports/amenities/images sub-resources, the public `GET /api/v1/turfs` and `GET /api/v1/turfs/{turfId}` (APPROVED only), OWNER-only authorisation with per-owner isolation enforced at the database, and the owner profile, turf list, turf create/edit and public turf browsing screens.

Business features are NOT implemented yet: slots, availability, bookings, payments, teams, tournaments, reviews, ratings, statistics, and admin approval of turfs (Phase 4).

Derived statistics are deliberately not stored. Matches played, wins and ratings are computed from source data once that data exists.

Follow the project roadmap/task prompt for the current phase. Do not redo a completed phase unless required.