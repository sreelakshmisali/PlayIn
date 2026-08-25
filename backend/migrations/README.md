# Migrations

Schema changes are applied with [golang-migrate](https://github.com/golang-migrate/migrate).
The tool runs from the `migrate/migrate` Docker image, so nothing extra has to
be installed locally.

## Rules

- Every change is a new migration. Never edit a migration that has been applied.
- Every `.up.sql` has a matching `.down.sql` that reverses it.
- Use UUID primary keys with `gen_random_uuid()` as the default.
- Every table gets `created_at` and `updated_at` as `TIMESTAMPTZ NOT NULL DEFAULT now()`.
- Declare foreign keys, unique constraints and indexes in the same migration as
  the table they belong to.
- Do not store values that can be derived from other rows. Counts, totals and
  averages are computed at read time.

## File naming

```
<version>_<description>.up.sql
<version>_<description>.down.sql
```

Version is a zero-padded six digit sequence: `000002`, `000003`, and so on.

## Commands

Run from the repository root.

```bash
docker compose --profile tools run --rm migrate up
```

```bash
docker compose --profile tools run --rm migrate down 1
```

```bash
docker compose --profile tools run --rm migrate version
```

Create a new migration pair:

```bash
docker compose --profile tools run --rm migrate create -ext sql -dir /migrations -seq add_venues
```

If a migration fails halfway, golang-migrate marks the schema dirty and refuses
to continue. Fix the SQL, then force the version back to the last good one:

```bash
docker compose --profile tools run --rm migrate force 1
```

## State

| Version | Contents |
|---|---|
| `000001_init` | Baseline placeholder. Creates nothing. |
| `000002_auth` | `users`, `refresh_tokens`, the `set_updated_at()` trigger function. |
| `000003_player_profiles` | `sports` (seeded), `player_profiles`, `player_sports`. |
| `000004_owner_turfs` | `owner_profiles`, `turfs`, `amenities` (seeded), `turf_sports`, `turf_amenities`, `turf_images`. |
| `000005_admin_moderation` | Turf moderation columns and admin actions. |
| `000006_turf_availability` | `turf_slots`, `turf_blocked_dates`, `turf_blocked_time_ranges`, slot settings on `turfs`. |
| `000007_bookings` | `bookings`; adds `BOOKED` to `turf_slots.status`. |

`000003` and `000004` seed the sports and amenities catalogues. They are
reference data the API cannot function without, not sample data, so they
belong in a migration rather than in an application bootstrap that would have
to be made idempotent.
