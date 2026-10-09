CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE IF NOT EXISTS menu_options (
    id BIGSERIAL PRIMARY KEY,
    parent_option_id BIGINT REFERENCES menu_options(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role VARCHAR(30) NOT NULL CHECK (role IN ('STAFF', 'ADMIN')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT users_password_hash_not_blank CHECK (NULLIF(BTRIM(password_hash), '') IS NOT NULL),
    CONSTRAINT users_username_not_blank CHECK (NULLIF(BTRIM(username), '') IS NOT NULL)
);

CREATE TABLE IF NOT EXISTS menu_option_prompts (
    id BIGSERIAL PRIMARY KEY,
    menu_option_id BIGINT NOT NULL REFERENCES menu_options(id) ON DELETE CASCADE,
    message TEXT NOT NULL,
    weight INTEGER NOT NULL DEFAULT 1 CHECK (weight > 0),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS response_groups (
    id BIGSERIAL PRIMARY KEY,
    menu_option_id BIGINT NOT NULL REFERENCES menu_options(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    weight INTEGER NOT NULL DEFAULT 1 CHECK (weight > 0),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS response_items (
    id BIGSERIAL PRIMARY KEY,
    response_group_id BIGINT NOT NULL REFERENCES response_groups(id) ON DELETE CASCADE,
    type VARCHAR(30) NOT NULL CHECK (type IN ('TEXT', 'LINK', 'IMAGE', 'YOUTUBE', 'VIDEO', 'DOCUMENT')),
    text TEXT,
    url TEXT,
    caption TEXT,
    metadata JSONB,
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT response_items_content_check CHECK (
        (type = 'TEXT' AND NULLIF(BTRIM(text), '') IS NOT NULL)
        OR (type IN ('LINK', 'IMAGE', 'YOUTUBE', 'VIDEO', 'DOCUMENT') AND NULLIF(BTRIM(url), '') IS NOT NULL)
    )
);

CREATE TABLE IF NOT EXISTS content_feedback (
    id BIGSERIAL PRIMARY KEY,
    full_name VARCHAR(255) NOT NULL,
    ip INET NOT NULL,
    message TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT content_feedback_full_name_not_blank CHECK (NULLIF(BTRIM(full_name), '') IS NOT NULL),
    CONSTRAINT content_feedback_message_not_blank CHECK (NULLIF(BTRIM(message), '') IS NOT NULL)
);

-- Enforce the menu-flow invariant even when data is written outside the API:
-- an option either continues through prompts and children, or ends in response groups.
CREATE OR REPLACE FUNCTION enforce_prompt_without_response_groups()
RETURNS TRIGGER AS $$
BEGIN
    PERFORM pg_advisory_xact_lock(NEW.menu_option_id);
    IF EXISTS (SELECT 1 FROM response_groups WHERE menu_option_id = NEW.menu_option_id) THEN
        RAISE EXCEPTION 'menu option % already has response groups; prompts are not allowed', NEW.menu_option_id
            USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION enforce_response_group_without_prompts_or_children()
RETURNS TRIGGER AS $$
BEGIN
    PERFORM pg_advisory_xact_lock(NEW.menu_option_id);
    IF EXISTS (SELECT 1 FROM menu_option_prompts WHERE menu_option_id = NEW.menu_option_id) THEN
        RAISE EXCEPTION 'menu option % already has prompts; response groups are not allowed', NEW.menu_option_id
            USING ERRCODE = '23514';
    END IF;
    IF EXISTS (SELECT 1 FROM menu_options WHERE parent_option_id = NEW.menu_option_id) THEN
        RAISE EXCEPTION 'menu option % already has child options; response groups are not allowed', NEW.menu_option_id
            USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION enforce_child_without_parent_response_groups()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.parent_option_id IS NULL THEN
        RETURN NEW;
    END IF;
    PERFORM pg_advisory_xact_lock(NEW.parent_option_id);
    IF EXISTS (SELECT 1 FROM response_groups WHERE menu_option_id = NEW.parent_option_id) THEN
        RAISE EXCEPTION 'parent menu option % already has response groups; child options are not allowed', NEW.parent_option_id
            USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS menu_option_prompts_flow_guard ON menu_option_prompts;
CREATE TRIGGER menu_option_prompts_flow_guard
BEFORE INSERT OR UPDATE OF menu_option_id ON menu_option_prompts
FOR EACH ROW EXECUTE FUNCTION enforce_prompt_without_response_groups();

DROP TRIGGER IF EXISTS response_groups_flow_guard ON response_groups;
CREATE TRIGGER response_groups_flow_guard
BEFORE INSERT OR UPDATE OF menu_option_id ON response_groups
FOR EACH ROW EXECUTE FUNCTION enforce_response_group_without_prompts_or_children();

DROP TRIGGER IF EXISTS menu_options_flow_guard ON menu_options;
CREATE TRIGGER menu_options_flow_guard
BEFORE INSERT OR UPDATE OF parent_option_id ON menu_options
FOR EACH ROW EXECUTE FUNCTION enforce_child_without_parent_response_groups();

CREATE INDEX IF NOT EXISTS idx_menu_options_parent_sort_order ON menu_options (parent_option_id, sort_order);
CREATE INDEX IF NOT EXISTS idx_menu_options_active ON menu_options (is_active);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_only_one_admin ON users (role) WHERE role = 'ADMIN';
CREATE INDEX IF NOT EXISTS idx_menu_option_prompts_menu_option ON menu_option_prompts (menu_option_id);
CREATE INDEX IF NOT EXISTS idx_menu_option_prompts_active ON menu_option_prompts (is_active);
CREATE INDEX IF NOT EXISTS idx_response_groups_menu_option ON response_groups (menu_option_id);
CREATE INDEX IF NOT EXISTS idx_response_groups_active ON response_groups (is_active);
CREATE INDEX IF NOT EXISTS idx_response_items_response_group_sort_order ON response_items (response_group_id, sort_order);
CREATE INDEX IF NOT EXISTS idx_response_items_active ON response_items (is_active);
CREATE INDEX IF NOT EXISTS idx_content_feedback_created_at ON content_feedback (created_at);

DROP TRIGGER IF EXISTS set_menu_options_updated_at ON menu_options;
CREATE TRIGGER set_menu_options_updated_at
BEFORE UPDATE ON menu_options
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS set_menu_option_prompts_updated_at ON menu_option_prompts;
CREATE TRIGGER set_menu_option_prompts_updated_at
BEFORE UPDATE ON menu_option_prompts
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS set_response_groups_updated_at ON response_groups;
CREATE TRIGGER set_response_groups_updated_at
BEFORE UPDATE ON response_groups
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS set_response_items_updated_at ON response_items;
CREATE TRIGGER set_response_items_updated_at
BEFORE UPDATE ON response_items
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
