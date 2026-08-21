# PlayHub

Production-quality sports community and turf booking platform.

## Stack

- Mobile App: React Native + Expo + TypeScript
- Backend: Go + REST API
- Database: PostgreSQL
- Infrastructure: Docker + Docker Compose
- API prefix: /api/v1

The primary PlayHub client is a MOBILE APP.
React Native + Expo is mandatory for player and owner experiences.

Do NOT build player/owner features with React + Vite.
Do NOT treat responsive web as the mobile app.
Admin may remain a web application where appropriate.

## Architecture

- Keep handlers/controllers thin.
- Business logic belongs in services/use-cases.
- Database access belongs in repositories.
- Use dependency injection where useful.
- PostgreSQL schema changes require migrations.
- Use UUIDs, timestamps, foreign keys, constraints and indexes.
- Derive statistics from source data.
- Configuration/secrets come from environment variables.
- Never commit secrets.
- Reuse existing architecture.
- Inspect existing code before modifying it.
- No unnecessary dependencies or technologies.
- No microservices, Redis, GraphQL, Kubernetes or ORM without approval.
- Do not implement features outside the requested phase.

## Mobile

- Mobile is the primary product.
- Build screens for React Native + Expo.
- Test important flows on a real mobile device when practical.
- Use mobile-native navigation, touch targets, forms and loading/error states.
- Do not use browser viewport testing as a substitute for real mobile testing.

## Development

For every phase:
1. Inspect existing code and git status.
2. Create a feature branch from the current approved branch.
3. Implement only the requested phase.
4. Build database + Go API + React Native UI when the feature requires all three.
5. Run tests, builds and integration checks.
6. Test important mobile flows.
7. Review git diff for unrelated changes.
8. Commit with a clear conventional commit.
9. Report branch, commit, changed files and verification.
10. STOP and ask permission to merge.

Never merge without explicit user approval.
Never push unless explicitly requested.
Never commit unrelated changes.

## Current Architecture

Backend/domain work already exists for:
- Authentication
- Player profiles
- Sports
- Owner profiles
- Turfs
- Turf moderation
- Slot/availability management

Existing web frontend code may exist from earlier development, but it is NOT the primary mobile client.

When implementing or replacing user-facing mobile functionality, use React Native + Expo.

## Important

Do not redo completed backend/domain work unnecessarily.

Preserve working Go APIs and PostgreSQL schema where possible.

The current approved product direction is:

React Native/Expo
        ↓
      Go API
        ↓
   PostgreSQL

Admin web UI may use React web separately.