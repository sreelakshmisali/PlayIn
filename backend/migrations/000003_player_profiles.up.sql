-- 000003_player_profiles
--
-- Player profiles and the sports catalogue they reference.
--
-- The sports rows are seeded here rather than by the application. They are
-- reference data the API cannot function without, not sample data: a player
-- picking a preferred sport has nothing to pick from until they exist.

-- --- sports ------------------------------------------------------------------
--
-- positions is an array on the sport rather than its own table, because a
-- position has no attributes of its own and exists only as a choice offered by
-- one sport. Membership is still enforced at the database: player_sports rows
-- are inserted through a SELECT against this column, so a position that is not
-- in the array inserts nothing.
--
-- Sports with no positions (badminton, tennis) carry an empty array, which is
-- what makes "position where applicable" a fact about the data rather than a
-- rule in application code.
CREATE TABLE sports (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    slug       TEXT        NOT NULL,
    name       TEXT        NOT NULL,
    positions  TEXT[]      NOT NULL DEFAULT '{}',
    -- Sports are retired by clearing this flag, never deleted: player_sports
    -- references them with ON DELETE RESTRICT.
    is_active  BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT sports_slug_format_chk CHECK (slug ~ '^[a-z][a-z0-9-]*$'),
    CONSTRAINT sports_name_chk        CHECK (char_length(btrim(name)) BETWEEN 2 AND 60),
    CONSTRAINT sports_positions_chk   CHECK (array_position(positions, '') IS NULL)
);

CREATE UNIQUE INDEX sports_slug_key ON sports (slug);
CREATE UNIQUE INDEX sports_name_key ON sports (name);

CREATE TRIGGER sports_set_updated_at
    BEFORE UPDATE ON sports
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

INSERT INTO sports (slug, name, positions) VALUES
    ('football',   'Football',   ARRAY['Goalkeeper', 'Defender', 'Midfielder', 'Forward']),
    ('cricket',    'Cricket',    ARRAY['Batter', 'Bowler', 'All-rounder', 'Wicketkeeper']),
    ('badminton',  'Badminton',  ARRAY[]::TEXT[]),
    ('basketball', 'Basketball', ARRAY['Point Guard', 'Shooting Guard', 'Small Forward', 'Power Forward', 'Center']),
    ('volleyball', 'Volleyball', ARRAY['Setter', 'Outside Hitter', 'Middle Blocker', 'Opposite', 'Libero']),
    ('tennis',     'Tennis',     ARRAY[]::TEXT[]);

-- --- player profiles ---------------------------------------------------------
--
-- One profile per account, enforced by the unique index on user_id rather than
-- by application code, so two concurrent first-time saves cannot both create
-- one.
--
-- Nothing here is derived. Match counts, win rates and ratings are computed
-- from source data when that data exists; storing them now would guarantee they
-- drift from it later.
CREATE TABLE player_profiles (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    display_name TEXT        NOT NULL,
    image_url    TEXT,
    bio          TEXT,
    location     TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT player_profiles_display_name_chk
        CHECK (char_length(btrim(display_name)) BETWEEN 2 AND 80),
    CONSTRAINT player_profiles_bio_chk
        CHECK (bio IS NULL OR char_length(bio) <= 500),
    CONSTRAINT player_profiles_location_chk
        CHECK (location IS NULL OR char_length(btrim(location)) BETWEEN 2 AND 120),
    -- A profile image is fetched by a browser, so the scheme is restricted to
    -- http and https. Without it the column would accept javascript: and data:
    -- URLs, which is a stored cross-site scripting vector the moment one is
    -- rendered into an href.
    CONSTRAINT player_profiles_image_url_chk
        CHECK (image_url IS NULL OR (image_url ~ '^https?://[^[:space:]]+$' AND char_length(image_url) <= 2048))
);

CREATE UNIQUE INDEX player_profiles_user_id_key ON player_profiles (user_id);

CREATE TRIGGER player_profiles_set_updated_at
    BEFORE UPDATE ON player_profiles
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- --- player sports -----------------------------------------------------------
--
-- The preferred sports of a profile, with an optional position for the sports
-- that have them.
CREATE TABLE player_sports (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID        NOT NULL REFERENCES player_profiles (id) ON DELETE CASCADE,
    -- RESTRICT, not CASCADE: deleting a sport that players have chosen would
    -- silently edit their profiles. Retire it with is_active instead.
    sport_id   UUID        NOT NULL REFERENCES sports (id) ON DELETE RESTRICT,
    position   TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT player_sports_position_chk
        CHECK (position IS NULL OR char_length(btrim(position)) BETWEEN 1 AND 60)
);

-- A sport appears at most once per profile. This is also the conflict target
-- the add-or-update insert relies on.
CREATE UNIQUE INDEX player_sports_profile_sport_key ON player_sports (profile_id, sport_id);

-- An unindexed foreign key column makes every delete on the referenced table
-- scan this one to check the constraint.
CREATE INDEX player_sports_sport_id_idx ON player_sports (sport_id);

CREATE TRIGGER player_sports_set_updated_at
    BEFORE UPDATE ON player_sports
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
