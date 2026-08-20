-- 000005_admin_moderation
--
-- Admin turf moderation. Adds the minimal audit trail migration 000004 left
-- for Phase 4: who last changed a turf's status, when, and why (for the two
-- statuses where a reason is meaningful). No separate audit/event table --
-- the task asks for simple audit information, and one row's own moderation
-- columns are enough to answer "why is this turf REJECTED/SUSPENDED" without
-- standing up an event log nothing else needs yet.

ALTER TABLE turfs
    ADD COLUMN moderation_reason TEXT,
    ADD COLUMN moderated_by      UUID REFERENCES users (id) ON DELETE SET NULL,
    ADD COLUMN moderated_at      TIMESTAMPTZ;

-- A reason only means something on a REJECTED or SUSPENDED turf. Tying the
-- column to status here means the moderation queries themselves (UPDATE ...
-- SET status = 'APPROVED', moderation_reason = NULL, ...) are the only way to
-- satisfy the constraint, so approving or restoring a turf without also
-- clearing the reason is a constraint violation, not a silent stale value.
ALTER TABLE turfs ADD CONSTRAINT turfs_moderation_reason_chk
    CHECK (
        moderation_reason IS NULL
        OR (
            status IN ('REJECTED', 'SUSPENDED')
            AND char_length(btrim(moderation_reason)) BETWEEN 3 AND 500
        )
    );

-- Mirrors turfs_approved_created_idx: the admin queue's one query filters on
-- PENDING_APPROVAL and orders by age, so the partial index matches that query
-- exactly rather than indexing every status value.
CREATE INDEX turfs_pending_created_idx ON turfs (created_at) WHERE status = 'PENDING_APPROVAL';
