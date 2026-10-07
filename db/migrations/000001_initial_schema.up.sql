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
    sort_order INTEGER NOT NULL DEFAULT 0,
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
    sort_order INTEGER NOT NULL DEFAULT 0,
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
    menu_option_id BIGINT REFERENCES menu_options(id) ON DELETE SET NULL,
    response_group_id BIGINT REFERENCES response_groups(id) ON DELETE SET NULL,
    feedback_type VARCHAR(30) NOT NULL CHECK (feedback_type IN ('CONTENT_REPORT', 'CONTENT_REQUEST')),
    category VARCHAR(30) NOT NULL CHECK (category IN ('INCORRECT', 'INCOMPLETE', 'MISSING', 'UNCLEAR', 'OUTDATED', 'OTHER')),
    subject VARCHAR(255),
    message TEXT NOT NULL,
    suggested_content TEXT,
    source_url TEXT,
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'IN_REVIEW', 'APPROVED', 'REJECTED', 'RESOLVED')),
    admin_notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at TIMESTAMPTZ,
    resolved_at TIMESTAMPTZ
);

ALTER TABLE content_feedback
    ADD COLUMN IF NOT EXISTS response_group_id BIGINT REFERENCES response_groups(id) ON DELETE SET NULL;

ALTER TABLE content_feedback
    DROP COLUMN IF EXISTS response_item_id;

CREATE INDEX IF NOT EXISTS idx_menu_options_parent_sort_order ON menu_options (parent_option_id, sort_order);
CREATE INDEX IF NOT EXISTS idx_menu_options_active ON menu_options (is_active);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_only_one_admin ON users (role) WHERE role = 'ADMIN';
CREATE INDEX IF NOT EXISTS idx_menu_option_prompts_menu_option_sort_order ON menu_option_prompts (menu_option_id, sort_order);
CREATE INDEX IF NOT EXISTS idx_menu_option_prompts_active ON menu_option_prompts (is_active);
CREATE INDEX IF NOT EXISTS idx_response_groups_menu_option_sort_order ON response_groups (menu_option_id, sort_order);
CREATE INDEX IF NOT EXISTS idx_response_groups_active ON response_groups (is_active);
CREATE INDEX IF NOT EXISTS idx_response_items_response_group_sort_order ON response_items (response_group_id, sort_order);
CREATE INDEX IF NOT EXISTS idx_response_items_active ON response_items (is_active);
CREATE INDEX IF NOT EXISTS idx_content_feedback_menu_option ON content_feedback (menu_option_id);
CREATE INDEX IF NOT EXISTS idx_content_feedback_response_group ON content_feedback (response_group_id);
CREATE INDEX IF NOT EXISTS idx_content_feedback_status_created_at ON content_feedback (status, created_at);
CREATE INDEX IF NOT EXISTS idx_content_feedback_type_category ON content_feedback (feedback_type, category);

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
