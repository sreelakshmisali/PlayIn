-- 000003_player_profiles (rollback)
--
-- Dropped in reverse dependency order. player_sports references both other
-- tables, so it goes first. The seeded sports rows go with their table.

DROP TABLE IF EXISTS player_sports;
DROP TABLE IF EXISTS player_profiles;
DROP TABLE IF EXISTS sports;
