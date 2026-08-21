-- 000006_turf_availability (down)

DROP TABLE IF EXISTS turf_blocked_time_ranges;
DROP TABLE IF EXISTS turf_blocked_dates;
DROP TABLE IF EXISTS turf_slots;

ALTER TABLE turfs
    DROP CONSTRAINT IF EXISTS turfs_slot_duration_chk,
    DROP CONSTRAINT IF EXISTS turfs_slot_price_chk,
    DROP COLUMN IF EXISTS slot_duration_minutes,
    DROP COLUMN IF EXISTS slot_price;

-- btree_gist is left in place: dropping an extension other migrations or
-- objects might come to depend on is a heavier, less reversible action than
-- this down migration should take on its own.
