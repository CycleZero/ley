-- =============================================================================
-- Ley Blog — 数据库初始化脚本
-- PostgreSQL 15+ 适用
--
-- 创建 4 个 Schema（user/post/comment/audit），每个 Schema 对应一个微服务。
-- 使用 GORM AutoMigrate 作为补充，此脚本确保索引和生成列正确创建。
--
-- 运行: psql -U ley -d ley -f migrations/001_init.sql
-- =============================================================================

BEGIN;

-- =============================================================================
-- 1. user schema — 用户服务
-- =============================================================================
CREATE SCHEMA IF NOT EXISTS "user";

-- 用户表
CREATE TABLE IF NOT EXISTS "user".users (
    id          BIGSERIAL    PRIMARY KEY,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ,
    uuid        VARCHAR(36)  NOT NULL UNIQUE,
    username    VARCHAR(32)  NOT NULL,
    email       VARCHAR(255) NOT NULL,
    password    VARCHAR(255) NOT NULL,          -- bcrypt hash
    nickname    VARCHAR(64)  NOT NULL DEFAULT '',
    avatar      VARCHAR(512) NOT NULL DEFAULT '',
    bio         TEXT         NOT NULL DEFAULT '',
    status      SMALLINT     NOT NULL DEFAULT 0, -- 0=正常 1=禁用
    role        VARCHAR(16)  NOT NULL DEFAULT 'reader'
);

-- 唯一索引（软删除感知）
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username
    ON "user".users(username) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email
    ON "user".users(email) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_uuid ON "user".users(uuid);
CREATE INDEX IF NOT EXISTS idx_users_status ON "user".users(status);
CREATE INDEX IF NOT EXISTS idx_users_deleted ON "user".users(deleted_at);

-- =============================================================================
-- 2. post schema — 文章服务
-- =============================================================================
CREATE SCHEMA IF NOT EXISTS "post";

-- 分类表
CREATE TABLE IF NOT EXISTS "post".categories (
    id          BIGSERIAL    PRIMARY KEY,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ,
    name        VARCHAR(64)  NOT NULL,
    slug        VARCHAR(64)  NOT NULL,
    description TEXT         NOT NULL DEFAULT '',
    parent_id   BIGINT,
    sort_order  INT          NOT NULL DEFAULT 0,
    post_count  BIGINT       NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_categories_slug
    ON "post".categories(slug);
CREATE INDEX IF NOT EXISTS idx_categories_parent
    ON "post".categories(parent_id);
CREATE INDEX IF NOT EXISTS idx_categories_deleted
    ON "post".categories(deleted_at);

-- 标签表
CREATE TABLE IF NOT EXISTS "post".tags (
    id          BIGSERIAL    PRIMARY KEY,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ,
    name        VARCHAR(64)  NOT NULL,
    slug        VARCHAR(64)  NOT NULL,
    post_count  BIGINT       NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_tags_name
    ON "post".tags(name) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_tags_slug
    ON "post".tags(slug) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_tags_deleted ON "post".tags(deleted_at);

-- 文章表
CREATE TABLE IF NOT EXISTS "post".posts (
    id              BIGSERIAL    PRIMARY KEY,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,
    uuid            VARCHAR(36)  NOT NULL UNIQUE,
    title           VARCHAR(200) NOT NULL,
    slug            VARCHAR(200) NOT NULL,
    content         TEXT         NOT NULL DEFAULT '',
    excerpt         TEXT         NOT NULL DEFAULT '',
    cover_image     VARCHAR(512) NOT NULL DEFAULT '',
    status          SMALLINT     NOT NULL DEFAULT 0,    -- 0=draft 1=published 2=archived
    author_id       BIGINT       NOT NULL,
    category_id     BIGINT,
    view_count      BIGINT       NOT NULL DEFAULT 0,
    like_count      BIGINT       NOT NULL DEFAULT 0,
    comment_count   BIGINT       NOT NULL DEFAULT 0,
    is_top          BOOLEAN      NOT NULL DEFAULT FALSE
);

-- 全文搜索向量列（由数据库自动生成和维护）
ALTER TABLE "post".posts
    ADD COLUMN IF NOT EXISTS search_vector tsvector;
ALTER TABLE "post".posts
    ALTER COLUMN search_vector SET DEFAULT ''::tsvector;

-- 手动维护搜索向量（因为 GENERATED ALWAYS AS 与 GORM 不兼容）
CREATE OR REPLACE FUNCTION "post".update_search_vector()
RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('simple', COALESCE(NEW.title, '')), 'A') ||
        setweight(to_tsvector('simple', COALESCE(NEW.content, '')), 'B');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_posts_search_vector ON "post".posts;
CREATE TRIGGER trg_posts_search_vector
    BEFORE INSERT OR UPDATE OF title, content ON "post".posts
    FOR EACH ROW EXECUTE FUNCTION "post".update_search_vector();

-- 索引
CREATE UNIQUE INDEX IF NOT EXISTS idx_posts_slug
    ON "post".posts(slug) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_posts_uuid ON "post".posts(uuid);
CREATE INDEX IF NOT EXISTS idx_posts_status
    ON "post".posts(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_posts_author
    ON "post".posts(author_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_posts_category
    ON "post".posts(category_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_posts_search
    ON "post".posts USING GIN(search_vector);
CREATE INDEX IF NOT EXISTS idx_posts_created
    ON "post".posts(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_posts_deleted
    ON "post".posts(deleted_at);

-- 文章-标签关联表（M:N）
CREATE TABLE IF NOT EXISTS "post".posts_tags (
    id          BIGSERIAL    PRIMARY KEY,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ,
    post_id     BIGINT       NOT NULL,
    tag_id      BIGINT       NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_posts_tags_pair
    ON "post".posts_tags(post_id, tag_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_posts_tags_post ON "post".posts_tags(post_id);
CREATE INDEX IF NOT EXISTS idx_posts_tags_tag ON "post".posts_tags(tag_id);
CREATE INDEX IF NOT EXISTS idx_posts_tags_deleted ON "post".posts_tags(deleted_at);

-- =============================================================================
-- 3. comment schema — 评论服务
-- =============================================================================
CREATE SCHEMA IF NOT EXISTS "comment";

CREATE TABLE IF NOT EXISTS "comment".comments (
    id          BIGSERIAL    PRIMARY KEY,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ,
    uuid        VARCHAR(36)  NOT NULL UNIQUE,
    post_id     BIGINT       NOT NULL,
    author_id   BIGINT       NOT NULL,
    parent_id   BIGINT,                          -- NULL=顶级评论
    content     TEXT         NOT NULL,
    status      SMALLINT     NOT NULL DEFAULT 0  -- 0=pending 1=approved 2=spam 3=deleted
);

CREATE INDEX IF NOT EXISTS idx_comments_post
    ON "comment".comments(post_id, created_at ASC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_comments_parent
    ON "comment".comments(parent_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_comments_uuid ON "comment".comments(uuid);
CREATE INDEX IF NOT EXISTS idx_comments_deleted ON "comment".comments(deleted_at);

-- =============================================================================
-- 4. audit schema — 审计日志（可选）
-- =============================================================================
CREATE SCHEMA IF NOT EXISTS "audit";

CREATE TABLE IF NOT EXISTS "audit".audit_logs (
    id          BIGSERIAL    PRIMARY KEY,
    user_id     BIGINT       NOT NULL DEFAULT 0,     -- 0=匿名
    action      VARCHAR(64)  NOT NULL,               -- post.viewed, comment.created
    target_id   VARCHAR(36),                         -- 文章/评论 UUID
    metadata    JSONB        DEFAULT '{}',
    ip          VARCHAR(45),
    user_agent  VARCHAR(512),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_user
    ON "audit".audit_logs(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_action
    ON "audit".audit_logs(action, created_at DESC);

COMMIT;
