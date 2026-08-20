-- 000004_owner_turfs
--
-- Owner profiles and the turfs they list, plus the amenities catalogue and the
-- join tables that attach sports and amenities to a turf.
--
-- Turf sports reference the sports table from 000003 rather than a duplicate
-- catalogue. Amenities get their own catalogue because they are not shared
-- with any other feature.

-- --- owner profiles ------------------------------------------------------
--
-- One profile per account, same shape and same reasoning as player_profiles:
-- the unique index on user_id is what makes two concurrent first-time saves
-- safe, not application code.
CREATE TABLE owner_profiles (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    display_name TEXT        NOT NULL,
    phone        TEXT,
    description  TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT owner_profiles_display_name_chk
        CHECK (char_length(btrim(display_name)) BETWEEN 2 AND 120),
    CONSTRAINT owner_profiles_phone_chk
        CHECK (phone IS NULL OR phone ~ '^[+0-9][0-9 ()-]{6,19}$'),
    CONSTRAINT owner_profiles_description_chk
        CHECK (description IS NULL OR char_length(description) <= 1000)
);

CREATE UNIQUE INDEX owner_profiles_user_id_key ON owner_profiles (user_id);

CREATE TRIGGER owner_profiles_set_updated_at
    BEFORE UPDATE ON owner_profiles
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- --- turfs -----------------------------------------------------------------
--
-- owner_id references owner_profiles, not users directly: listing a turf
-- requires an owner profile to exist first, the same anchor pattern
-- player_sports uses against player_profiles. It also means a turf's owner
-- display name is one join away without touching the users table.
--
-- opening_time and closing_time are TEXT with an HH:MM check rather than the
-- SQL TIME type. pgx does not scan TIME into a plain Go type without extra
-- machinery, and a daily operating window has no need for the precision or
-- arithmetic TIME would offer over a validated string.
--
-- status has no default that means "live". A turf starts as DRAFT so an owner
-- can fill it in before anyone can find it, and only reaches PENDING_APPROVAL
-- by an explicit submit.
CREATE TABLE turfs (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id      UUID        NOT NULL REFERENCES owner_profiles (id) ON DELETE CASCADE,
    name          TEXT        NOT NULL,
    description   TEXT,
    address       TEXT        NOT NULL,
    city          TEXT        NOT NULL,
    latitude      DOUBLE PRECISION,
    longitude     DOUBLE PRECISION,
    capacity      INTEGER,
    opening_time  TEXT        NOT NULL,
    closing_time  TEXT        NOT NULL,
    status        TEXT        NOT NULL DEFAULT 'DRAFT',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT turfs_name_chk        CHECK (char_length(btrim(name)) BETWEEN 2 AND 120),
    CONSTRAINT turfs_description_chk CHECK (description IS NULL OR char_length(description) <= 2000),
    CONSTRAINT turfs_address_chk     CHECK (char_length(btrim(address)) BETWEEN 5 AND 250),
    CONSTRAINT turfs_city_chk        CHECK (char_length(btrim(city)) BETWEEN 2 AND 100),
    CONSTRAINT turfs_capacity_chk    CHECK (capacity IS NULL OR capacity > 0),
    -- Latitude and longitude are supplied together or not at all; one without
    -- the other is not a usable coordinate.
    CONSTRAINT turfs_latlng_pair_chk CHECK ((latitude IS NULL) = (longitude IS NULL)),
    CONSTRAINT turfs_latitude_chk    CHECK (latitude IS NULL OR latitude BETWEEN -90 AND 90),
    CONSTRAINT turfs_longitude_chk   CHECK (longitude IS NULL OR longitude BETWEEN -180 AND 180),
    CONSTRAINT turfs_opening_time_chk
        CHECK (opening_time ~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'),
    CONSTRAINT turfs_closing_time_chk
        CHECK (closing_time ~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'),
    CONSTRAINT turfs_status_chk
        CHECK (status IN ('DRAFT', 'PENDING_APPROVAL', 'APPROVED', 'REJECTED', 'SUSPENDED'))
);

-- An owner's turf list is fetched by owner_id; unindexed it would scan the
-- whole table on every one of their requests.
CREATE INDEX turfs_owner_id_idx ON turfs (owner_id);

-- One owner cannot list two turfs under the same name. Case-insensitive so
-- "Riverside Turf" and "riverside turf" cannot coexist either.
CREATE UNIQUE INDEX turfs_owner_name_key ON turfs (owner_id, lower(name));

-- The public listing is the only query that filters on status, and it always
-- filters to APPROVED and orders by recency, so the index matches that query
-- exactly rather than indexing every status value.
CREATE INDEX turfs_approved_created_idx ON turfs (created_at DESC) WHERE status = 'APPROVED';

CREATE TRIGGER turfs_set_updated_at
    BEFORE UPDATE ON turfs
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- --- amenities ---------------------------------------------------------------
--
-- Same shape as sports: a small seeded catalogue, retired with is_active
-- rather than deleted so turf_amenities keeps a valid foreign key.
CREATE TABLE amenities (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    slug       TEXT        NOT NULL,
    name       TEXT        NOT NULL,
    is_active  BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT amenities_slug_format_chk CHECK (slug ~ '^[a-z][a-z0-9-]*$'),
    CONSTRAINT amenities_name_chk        CHECK (char_length(btrim(name)) BETWEEN 2 AND 60)
);

CREATE UNIQUE INDEX amenities_slug_key ON amenities (slug);
CREATE UNIQUE INDEX amenities_name_key ON amenities (name);

CREATE TRIGGER amenities_set_updated_at
    BEFORE UPDATE ON amenities
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

INSERT INTO amenities (slug, name) VALUES
    ('parking',         'Parking'),
    ('floodlights',     'Floodlights'),
    ('washroom',        'Washroom'),
    ('drinking-water',  'Drinking Water'),
    ('changing-room',   'Changing Room'),
    ('cafeteria',       'Cafeteria');

-- --- turf sports ---------------------------------------------------------
--
-- Which sports a turf supports. No per-row attributes: a turf either supports
-- a sport or it does not, unlike a player's preferred sport there is no
-- position to record.
CREATE TABLE turf_sports (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    turf_id    UUID        NOT NULL REFERENCES turfs (id) ON DELETE CASCADE,
    -- RESTRICT: a sport chosen by a live turf cannot be deleted out from
    -- under it. Retire it with is_active instead.
    sport_id   UUID        NOT NULL REFERENCES sports (id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT turf_sports_turf_sport_key UNIQUE (turf_id, sport_id)
);

CREATE INDEX turf_sports_sport_id_idx ON turf_sports (sport_id);

-- --- turf amenities --------------------------------------------------------
CREATE TABLE turf_amenities (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    turf_id     UUID        NOT NULL REFERENCES turfs (id) ON DELETE CASCADE,
    amenity_id  UUID        NOT NULL REFERENCES amenities (id) ON DELETE RESTRICT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT turf_amenities_turf_amenity_key UNIQUE (turf_id, amenity_id)
);

CREATE INDEX turf_amenities_amenity_id_idx ON turf_amenities (amenity_id);

-- --- turf images -------------------------------------------------------------
--
-- URLs only. No upload, no storage, no resizing: a row is exactly the string
-- the owner supplied, restricted to http and https for the same reason the
-- player profile image is: anything else becomes stored cross-site scripting
-- the moment it is rendered into an attribute.
CREATE TABLE turf_images (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    turf_id    UUID        NOT NULL REFERENCES turfs (id) ON DELETE CASCADE,
    image_url  TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT turf_images_url_chk
        CHECK (image_url ~ '^https?://[^[:space:]]+$' AND char_length(image_url) <= 2048),
    CONSTRAINT turf_images_turf_url_key UNIQUE (turf_id, image_url)
);

CREATE INDEX turf_images_turf_id_idx ON turf_images (turf_id);
