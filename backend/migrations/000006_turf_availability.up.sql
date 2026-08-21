-- 000006_turf_availability
--
-- Turf slot and availability management (Phase 5).
--
-- Operating hours are not duplicated here: turfs.opening_time and
-- turfs.closing_time from 000004 already are the operating window, edited
-- through the existing turf update endpoint. This migration adds only what
-- Phase 3 had no concept of: a configurable slot duration and price per turf,
-- the generated slot rows themselves, and two kinds of owner-authored block.
--
-- btree_gist backs the EXCLUDE constraints below: PostgreSQL's GiST index
-- needs it to combine the plain-equality columns (turf_id, the date) with a
-- range-overlap column in a single index. It is a standard contrib
-- extension, not an application dependency.
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- --- slot configuration on turfs -------------------------------------------
--
-- Nullable: a turf has no slots until its owner configures both. Generation
-- requires both to be set (enforced in the service, not here, so the
-- constraint story stays "each column is independently sane" rather than
-- "these two columns must agree").
ALTER TABLE turfs
    ADD COLUMN slot_duration_minutes INTEGER,
    ADD COLUMN slot_price            NUMERIC(10, 2);

ALTER TABLE turfs
    ADD CONSTRAINT turfs_slot_duration_chk
        CHECK (slot_duration_minutes IS NULL
            OR (slot_duration_minutes BETWEEN 15 AND 240 AND slot_duration_minutes % 15 = 0)),
    ADD CONSTRAINT turfs_slot_price_chk
        CHECK (slot_price IS NULL OR slot_price >= 0);

-- --- turf slots --------------------------------------------------------------
--
-- The materialized, addressable unit of availability. start_time/end_time are
-- TEXT HH:MM, matching turfs' own opening_time/closing_time convention and for
-- the same reason (pgx has no light-weight scan for SQL TIME). minute_range is
-- a STORED generated column computed from those two columns purely so the
-- EXCLUDE constraint below has a range type to compare; the repository never
-- selects it and no Go type represents it. It is not a second copy of
-- availability, only a derived normal form of the same start/end the row
-- already carries, kept in lockstep by PostgreSQL itself rather than by
-- application code.
--
-- status is OPEN or BLOCKED in this phase. Phase 6 adds a BOOKED value and
-- reserves a slot with `UPDATE turf_slots SET status = 'BOOKED' WHERE id = $1
-- AND status = 'OPEN'`, the same guarded-write shape turfs.status transitions
-- already use, so no new concurrency mechanism is needed for booking.
--
-- price is copied from turfs.slot_price at generation time rather than read
-- live from the turf on every request: once a slot exists, changing the
-- turf's configured price must not change what an already-generated slot
-- costs.
CREATE TABLE turf_slots (
    id           UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    turf_id      UUID          NOT NULL REFERENCES turfs (id) ON DELETE CASCADE,
    slot_date    DATE          NOT NULL,
    start_time   TEXT          NOT NULL,
    end_time     TEXT          NOT NULL,
    price        NUMERIC(10,2) NOT NULL,
    status       TEXT          NOT NULL DEFAULT 'OPEN',
    minute_range int4range GENERATED ALWAYS AS (
        int4range(
            split_part(start_time, ':', 1)::int * 60 + split_part(start_time, ':', 2)::int,
            split_part(end_time, ':', 1)::int * 60 + split_part(end_time, ':', 2)::int
        )
    ) STORED,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT turf_slots_start_time_chk CHECK (start_time ~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'),
    CONSTRAINT turf_slots_end_time_chk   CHECK (end_time ~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'),
    CONSTRAINT turf_slots_time_order_chk CHECK (end_time > start_time),
    CONSTRAINT turf_slots_price_chk      CHECK (price >= 0),
    CONSTRAINT turf_slots_status_chk     CHECK (status IN ('OPEN', 'BLOCKED')),

    -- The structural overlap guard: no two slots on the same turf and date
    -- may occupy overlapping minutes, regardless of how they were inserted.
    CONSTRAINT turf_slots_no_overlap
        EXCLUDE USING gist (turf_id WITH =, slot_date WITH =, minute_range WITH &&)
);

-- Both the owner's management view and the public availability read filter by
-- turf and a date or date range; this is the one index that serves both.
CREATE INDEX turf_slots_turf_date_idx ON turf_slots (turf_id, slot_date);

CREATE TRIGGER turf_slots_set_updated_at
    BEFORE UPDATE ON turf_slots
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- --- blocked dates -----------------------------------------------------------
--
-- A whole day an owner has taken off availability, independent of whether
-- slots have been generated for it yet: generation skips a blocked date, and
-- the availability read excludes one even if slots already exist there.
CREATE TABLE turf_blocked_dates (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    turf_id      UUID        NOT NULL REFERENCES turfs (id) ON DELETE CASCADE,
    blocked_date DATE        NOT NULL,
    reason       TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT turf_blocked_dates_reason_chk CHECK (reason IS NULL OR char_length(reason) <= 250),
    CONSTRAINT turf_blocked_dates_turf_date_key UNIQUE (turf_id, blocked_date)
);

-- --- blocked time ranges -----------------------------------------------------
--
-- A part of one day, e.g. a maintenance window. Same overlap guard as
-- turf_slots, for the same reason: two ranges an owner enters for the same
-- turf and date must not be allowed to overlap either.
CREATE TABLE turf_blocked_time_ranges (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    turf_id      UUID        NOT NULL REFERENCES turfs (id) ON DELETE CASCADE,
    blocked_date DATE        NOT NULL,
    start_time   TEXT        NOT NULL,
    end_time     TEXT        NOT NULL,
    reason       TEXT,
    minute_range int4range GENERATED ALWAYS AS (
        int4range(
            split_part(start_time, ':', 1)::int * 60 + split_part(start_time, ':', 2)::int,
            split_part(end_time, ':', 1)::int * 60 + split_part(end_time, ':', 2)::int
        )
    ) STORED,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT turf_blocked_time_ranges_start_chk CHECK (start_time ~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'),
    CONSTRAINT turf_blocked_time_ranges_end_chk   CHECK (end_time ~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'),
    CONSTRAINT turf_blocked_time_ranges_order_chk CHECK (end_time > start_time),
    CONSTRAINT turf_blocked_time_ranges_reason_chk CHECK (reason IS NULL OR char_length(reason) <= 250),

    CONSTRAINT turf_blocked_time_ranges_no_overlap
        EXCLUDE USING gist (turf_id WITH =, blocked_date WITH =, minute_range WITH &&)
);

CREATE INDEX turf_blocked_time_ranges_turf_date_idx ON turf_blocked_time_ranges (turf_id, blocked_date);
