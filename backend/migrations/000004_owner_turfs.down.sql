-- 000004_owner_turfs (rollback)
--
-- Dropped in reverse dependency order: the join tables first, then turfs,
-- then the two tables they reference. Seeded amenities rows go with their
-- table.

DROP TABLE IF EXISTS turf_images;
DROP TABLE IF EXISTS turf_amenities;
DROP TABLE IF EXISTS turf_sports;
DROP TABLE IF EXISTS turfs;
DROP TABLE IF EXISTS amenities;
DROP TABLE IF EXISTS owner_profiles;
