-- 000005_admin_moderation (rollback)

DROP INDEX IF EXISTS turfs_pending_created_idx;

ALTER TABLE turfs DROP CONSTRAINT IF EXISTS turfs_moderation_reason_chk;

ALTER TABLE turfs
    DROP COLUMN IF EXISTS moderation_reason,
    DROP COLUMN IF EXISTS moderated_by,
    DROP COLUMN IF EXISTS moderated_at;
