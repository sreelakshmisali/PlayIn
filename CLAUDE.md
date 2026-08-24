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
8. Follow the Git Commit Workflow below — commit only if verification passed.
9. Report branch, commit hash, changed files and verification.
10. STOP and ask permission to merge.

Never merge without explicit user approval.
Never push unless explicitly requested.
Never commit unrelated changes.

## Git Commit Workflow

Committing completed work is a required, automatic step of finishing a
feature/task — not an optional or separately-requested one. This applies to
every future development task in this project.

**When to commit**
- The moment a feature/task is fully implemented AND verified, commit it.
  Do not wait to be asked.
- Treat the commit as the last step of "done," the same way you'd treat a
  passing test run — a task isn't complete until it's committed.

**Before committing — verify first**
- Confirm the feature actually works: run the relevant builds/tests/
  typechecks, and exercise the important flow (mobile flow, API call,
  migration) per the Development steps above.
- Confirm there are no obvious errors or regressions introduced by the
  change.
- Confirm there's no unfinished TODO, stub, or half-wired piece belonging
  to this task.
- If verification fails or something is unfinished: do NOT commit. Fix it
  first, or if it can't be fixed now, clearly report what's blocking
  completion instead of committing broken/partial work.

**What to stage**
- Review the actual diff (`git status`, `git diff`) before staging anything.
- Stage only the files relevant to the completed task.
- Never blindly run `git add .` (or `git add -A`) if the working tree could
  contain unrelated changes — inspect first, then stage deliberately (e.g.
  `git add <specific files>`).
- Never commit unrelated changes, leftover debug files, or edits from a
  different task, even if they're sitting in the working tree.

**Commit shape**
- One completed feature/task normally produces one focused commit, not a
  scattering of intermediate WIP commits.
- Do not create unnecessary commits for intermediate/in-progress states —
  commit the finished, verified result.
- Use a clear conventional commit message describing what was completed:
  - `feat: add turf booking flow`
  - `fix: resolve booking API timeout`
  - `refactor: simplify booking state management`

**After committing**
- Report the commit hash and a short summary of what was committed.
- This still does not authorize a merge or push — those remain separate,
  explicit-approval steps per the Development workflow above.

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