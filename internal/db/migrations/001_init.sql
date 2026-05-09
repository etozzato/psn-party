CREATE TABLE IF NOT EXISTS groups (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    admin_token_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS groups_name_active_unique
    ON groups (LOWER(name))
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS entries (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    online_id TEXT NOT NULL,
    online_id_norm TEXT NOT NULL,
    entry_token_hash TEXT NOT NULL,
    profile_url TEXT NOT NULL,
    is_public BOOLEAN,
    profile_checked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    removed_at TIMESTAMPTZ,
    banned_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS entries_group_online_id_active_unique
    ON entries (group_id, online_id_norm)
    WHERE removed_at IS NULL;

CREATE INDEX IF NOT EXISTS entries_group_recent_idx
    ON entries (group_id, created_at DESC)
    WHERE removed_at IS NULL AND banned_at IS NULL;

CREATE TABLE IF NOT EXISTS blocked_entries (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    online_id TEXT NOT NULL,
    online_id_norm TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS blocked_entries_group_online_id_unique
    ON blocked_entries (group_id, online_id_norm);

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS groups_set_updated_at ON groups;
CREATE TRIGGER groups_set_updated_at
    BEFORE UPDATE ON groups
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS entries_set_updated_at ON entries;
CREATE TRIGGER entries_set_updated_at
    BEFORE UPDATE ON entries
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
