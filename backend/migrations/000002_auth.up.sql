-- 000002_auth
--
-- Authentication schema: the user account table and the refresh token table
-- that makes logout and token rotation possible.
--
-- gen_random_uuid() is in core PostgreSQL from 13 onwards, so no extension is
-- required for the UUID primary keys.

-- Keeps updated_at honest without every writer having to remember it.
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- --- users -------------------------------------------------------------------
--
-- role is TEXT with a CHECK rather than a native enum. golang-migrate wraps a
-- migration in a transaction and ALTER TYPE ... ADD VALUE cannot run inside
-- one, so an enum would make every future role change awkward for no extra
-- safety over the constraint below.
--
-- Email is stored already lowercased. The application normalises before it
-- writes and the CHECK stops anything else getting in, which lets a plain
-- unique index do case-insensitive uniqueness without citext.
CREATE TABLE users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT        NOT NULL,
    password_hash TEXT        NOT NULL,
    full_name     TEXT        NOT NULL,
    role          TEXT        NOT NULL DEFAULT 'PLAYER',
    is_active     BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT users_email_lowercase_chk CHECK (email = lower(email)),
    CONSTRAINT users_email_format_chk    CHECK (email ~ '^[^@[:space:]]+@[^@[:space:]]+\.[^@[:space:]]+$'),
    CONSTRAINT users_email_length_chk    CHECK (char_length(email) BETWEEN 3 AND 254),
    CONSTRAINT users_full_name_chk       CHECK (char_length(btrim(full_name)) BETWEEN 2 AND 120),
    CONSTRAINT users_password_hash_chk   CHECK (char_length(password_hash) >= 20),
    CONSTRAINT users_role_chk            CHECK (role IN ('PLAYER', 'OWNER', 'ADMIN'))
);

CREATE UNIQUE INDEX users_email_key ON users (email);

-- Admin screens list accounts by role; without this they sequential scan.
CREATE INDEX users_role_idx ON users (role);

CREATE TRIGGER users_set_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- --- refresh tokens ----------------------------------------------------------
--
-- One row per issued refresh token. The row id is the JWT's jti claim, so a
-- presented refresh token can be checked against server state: unknown, expired
-- or revoked jti is refused. This is what makes logout and rotation real rather
-- than a client-side gesture.
CREATE TABLE refresh_tokens (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT refresh_tokens_expiry_chk CHECK (expires_at > created_at)
);

-- Revoking every token for a user (logout everywhere, deactivation) filters on
-- user_id, and the partial predicate keeps the index to live tokens only.
CREATE INDEX refresh_tokens_active_user_idx
    ON refresh_tokens (user_id)
    WHERE revoked_at IS NULL;

-- Supports periodic deletion of tokens that are past their expiry.
CREATE INDEX refresh_tokens_expires_at_idx ON refresh_tokens (expires_at);

CREATE TRIGGER refresh_tokens_set_updated_at
    BEFORE UPDATE ON refresh_tokens
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
