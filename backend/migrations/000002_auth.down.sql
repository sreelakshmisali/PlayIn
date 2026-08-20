-- 000002_auth (rollback)
--
-- Dropped in reverse dependency order. refresh_tokens references users, and
-- both triggers go with their tables, so only the shared function needs an
-- explicit drop.

DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS users;

DROP FUNCTION IF EXISTS set_updated_at();
