-- 000007_bookings
--
-- Phase 6: a player's reservation of one turf slot.
--
-- Concurrency safety does not come from anything in this migration alone —
-- it comes from the combination of this schema and how the service writes to
-- it. The service reserves a slot with the same guarded UPDATE
-- turf_slots.status transitions already use elsewhere (see slot_model.go):
--
--     UPDATE turf_slots SET status = 'BOOKED'
--     WHERE id = $1 AND status = 'OPEN' AND <not blocked>
--
-- PostgreSQL takes a row lock for the duration of that UPDATE, so two
-- concurrent bookings for the same slot serialise on it: the first commits
-- with status now BOOKED, the second re-evaluates the WHERE clause against
-- that committed row and matches zero rows. No SELECT ... FOR UPDATE or
-- advisory lock is needed; this is the same "guarded write" shape the turf
-- and slot status columns already rely on.
--
-- bookings_slot_confirmed_key below is the second, independent guarantee:
-- even if the guarded UPDATE were ever bypassed or miswritten, the database
-- itself refuses a second CONFIRMED booking on the same slot.

-- BOOKED joins the slot's own status. Existing rows are unaffected: OPEN and
-- BLOCKED are unchanged, and nothing currently in turf_slots can be BOOKED
-- before this migration runs.
ALTER TABLE turf_slots DROP CONSTRAINT turf_slots_status_chk;
ALTER TABLE turf_slots ADD CONSTRAINT turf_slots_status_chk CHECK (status IN ('OPEN', 'BLOCKED', 'BOOKED'));

-- Lets bookings carry (turf_id, turf_slot_id) as a composite foreign key
-- back to the exact slot, so the two columns can never disagree about which
-- turf the slot belongs to. id alone is already unique via the primary key;
-- this is the same tuple, just also declared unique so a composite FK can
-- reference it.
ALTER TABLE turf_slots ADD CONSTRAINT turf_slots_turf_id_id_key UNIQUE (turf_id, id);

-- --- bookings ------------------------------------------------------------
--
-- player_id references users directly, not player_profiles: booking is a
-- capability of the PLAYER role (enforced by RequireRole at the handler),
-- not something that requires a player to have filled in a profile first.
--
-- turf_id is denormalised from turf_slot_id's own turf. It is not a second
-- source of truth: the composite foreign key below ties the pair to one row
-- in turf_slots, so turf_id can never point anywhere but the slot's actual
-- turf. Carrying it here anyway is what lets "my bookings" list a turf name
-- without a second join through turf_slots for every row.
--
-- price is copied from the slot at booking time, the same reasoning
-- turf_slots.price already uses for turfs.slot_price: once a booking exists,
-- a later change to the slot's price must not change what was already paid
-- for.
--
-- The state machine is deliberately two states. CONFIRMED is the only
-- status a booking is created with; CANCELLED is the only place it can go.
-- cancelled_at is required exactly when, and only when, status is
-- CANCELLED, so the column and the status can never drift apart.
CREATE TABLE bookings (
    id           UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id    UUID          NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    turf_id      UUID          NOT NULL,
    turf_slot_id UUID          NOT NULL,
    status       TEXT          NOT NULL DEFAULT 'CONFIRMED',
    price        NUMERIC(10,2) NOT NULL,
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ   NOT NULL DEFAULT now(),
    cancelled_at TIMESTAMPTZ,

    CONSTRAINT bookings_status_chk CHECK (status IN ('CONFIRMED', 'CANCELLED')),
    CONSTRAINT bookings_price_chk  CHECK (price >= 0),
    CONSTRAINT bookings_cancelled_at_chk CHECK (
        (status = 'CANCELLED' AND cancelled_at IS NOT NULL) OR
        (status <> 'CANCELLED' AND cancelled_at IS NULL)
    ),

    -- Ties turf_id and turf_slot_id to one real, matching row in turf_slots.
    -- ON DELETE RESTRICT: a slot (or, transitively, a turf) that has ever
    -- been booked cannot simply disappear out from under its booking
    -- history.
    CONSTRAINT bookings_slot_fk FOREIGN KEY (turf_id, turf_slot_id)
        REFERENCES turf_slots (turf_id, id) ON DELETE RESTRICT
);

-- At most one CONFIRMED booking per slot, ever. This is the structural
-- backstop for "two simultaneous requests must never successfully book the
-- same slot" — a partial unique index, not an application check, so it
-- holds even under concurrent transactions.
CREATE UNIQUE INDEX bookings_slot_confirmed_key ON bookings (turf_slot_id) WHERE status = 'CONFIRMED';

-- A player's own booking list, most recent first.
CREATE INDEX bookings_player_id_created_at_idx ON bookings (player_id, created_at DESC);

CREATE TRIGGER bookings_set_updated_at
    BEFORE UPDATE ON bookings
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
