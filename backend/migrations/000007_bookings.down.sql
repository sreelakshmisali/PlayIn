-- 000007_bookings (down)

DROP TABLE IF EXISTS bookings;

ALTER TABLE turf_slots DROP CONSTRAINT IF EXISTS turf_slots_turf_id_id_key;

ALTER TABLE turf_slots DROP CONSTRAINT IF EXISTS turf_slots_status_chk;
ALTER TABLE turf_slots ADD CONSTRAINT turf_slots_status_chk CHECK (status IN ('OPEN', 'BLOCKED'));
