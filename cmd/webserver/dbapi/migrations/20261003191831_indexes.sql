-- +goose Up

-- Article revisions: latest published revision for an article
CREATE INDEX idx_articles_revisions_article_published
    ON articles__revisions (article_id, published_at DESC)
    WHERE published_at IS NOT NULL;

-- Article feed: latest published revisions across articles
CREATE INDEX idx_articles_revisions_published
    ON articles__revisions (published_at DESC, article_id)
    WHERE published_at IS NOT NULL;

-- Reverse lookup of articles by tag
CREATE INDEX idx_articles_tags_tag
    ON articles__tags (tag_id, article_id);

-- Find assets belonging to a revision
CREATE INDEX idx_revision_assets_revision
    ON articles__revisions__assets (revision_id);

-- Find all references to an asset
CREATE INDEX idx_revision_assets_hash
    ON articles__revisions__assets (sha512_hash);

-- Find sessions belonging to a user
CREATE INDEX idx_user_sessions_user
    ON users__sessions (user_id);

-- Find OIDC identities belonging to a user
CREATE INDEX idx_oidc_identities_user
    ON users__oidc_identities (user_id);

-- Find SSH keys belonging to a user
CREATE INDEX idx_ssh_keys_user
    ON users__ssh_keys (user_id);
