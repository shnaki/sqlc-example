CREATE TYPE post_status AS ENUM ('draft', 'published', 'archived');

CREATE TABLE posts (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    author_id    UUID        NOT NULL REFERENCES authors(id) ON DELETE CASCADE,
    title        TEXT        NOT NULL,
    body         TEXT        NOT NULL DEFAULT '',
    tags         TEXT[]      NOT NULL DEFAULT '{}',
    status       post_status NOT NULL DEFAULT 'draft',
    published_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_posts_author ON posts(author_id);
CREATE INDEX idx_posts_status ON posts(status);
CREATE INDEX idx_posts_keyset ON posts(created_at DESC, id DESC);
