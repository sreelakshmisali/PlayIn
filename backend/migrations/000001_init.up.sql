-- 000001_init
--
-- Baseline migration. It exists so the migration tooling has a starting point
-- and so the schema_migrations bookkeeping table is created before any real
-- schema lands. No business tables are defined in Phase 0.
--
-- PostgreSQL 13 and later provide gen_random_uuid() in core, so no extension
-- is needed for UUID primary keys in later migrations.

SELECT 1;
