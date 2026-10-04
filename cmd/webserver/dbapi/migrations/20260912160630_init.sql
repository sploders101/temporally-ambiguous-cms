-- +goose Up
CREATE TABLE users (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE users__sessions(
    token_hash BYTEA NOT NULL UNIQUE PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires TIMESTAMPTZ NOT NULL
);

CREATE TABLE users__oidc_identities (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    issuer TEXT NOT NULL,
    subject TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_login_at TIMESTAMPTZ,

    UNIQUE (issuer, subject)
);

CREATE TABLE users__ssh_keys(
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    public_key TEXT NOT NULL UNIQUE,
    fingerprint TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ
);

CREATE TABLE tags(
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    slug TEXT NOT NULL UNIQUE
);

CREATE TABLE articles(
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    public_id UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    author UUID REFERENCES users(id) ON DELETE SET NULL,
    slug TEXT NOT NULL UNIQUE
);

CREATE TABLE articles__revisions(
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    public_id UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    article_id BIGINT NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ
);

CREATE TABLE articles__tags(
    article_id BIGINT NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    tag_id BIGINT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (article_id, tag_id)
);

CREATE TABLE articles__revisions__assets(
    revision_id BIGINT NOT NULL REFERENCES articles__revisions(id) ON DELETE CASCADE,
    sha512_hash BYTEA NOT NULL, -- Not a foreign key because this row may be created first.
    file_name TEXT NOT NULL,

    UNIQUE (revision_id, file_name)
);

CREATE TABLE assets(
    sha512_hash BYTEA NOT NULL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    content_type TEXT NOT NULL,
    content_length BIGINT NOT NULL
);
