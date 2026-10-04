-- name: CreateArticle :one
INSERT INTO articles (
    author,
    slug
) VALUES ($1, $2)
ON CONFLICT (slug) DO NOTHING
RETURNING *;

-- name: StageArticleRevision :one
INSERT INTO articles__revisions(
    article_id,
    title,
    description,
    body
) VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetArticleRevision :one
SELECT
    sqlc.embed(a),
    sqlc.embed(ar)
FROM articles__revisions ar
INNER JOIN articles a ON a.id = ar.article_id
WHERE ar.public_id = $1;

-- name: PublishArticleRevision :exec
UPDATE articles__revisions
SET
    published_at = COALESCE($2, now())
WHERE
    public_id = $1;

-- name: GetArticleBySlug :one
SELECT *
FROM articles
WHERE slug = $1;

-- name: GetPublishedRevisionBySlug :one
SELECT
    sqlc.embed(a),
    sqlc.embed(ar)
FROM articles a
INNER JOIN articles__revisions ar ON ar.article_id = a.id
WHERE
    a.slug = $1
    AND ar.published_at IS NOT NULL
    AND ar.published_at <= now()
ORDER BY ar.published_at DESC
LIMIT 1;

-- name: GetPublishedRevisionByUuid :one
SELECT
    sqlc.embed(a),
    sqlc.embed(ar)
FROM articles a
INNER JOIN articles__revisions ar ON ar.article_id = a.id
WHERE
    (
        a.public_id = $1
        OR ar.public_id = $1
    )
    AND ar.published_at IS NOT NULL
    AND ar.published_at <= now()
ORDER BY ar.published_at DESC
LIMIT 1;

-- name: RedactArticle :exec
UPDATE articles__revisions
SET published_at = NULL
WHERE article_id = $1;

-- name: CreateAsset :exec
INSERT INTO assets(
    sha512_hash,
    content_type,
    content_length
) VALUES ($1, $2, $3);

-- name: LinkArticleAsset :exec
INSERT INTO articles__revisions__assets(
    revision_id,
    sha512_hash,
    file_name
) VALUES ($1, $2, $3);

-- name: GetMissingArticleAssets :many
SELECT *
FROM articles__revisions__assets ara
WHERE
    revision_id = $1
    AND NOT EXISTS (
        SELECT 1
        FROM assets
        WHERE assets.sha512_hash = ara.sha512_hash
    );

-- name: GetMissingArticleAssetsByUuid :many
SELECT ara.*
FROM articles__revisions__assets ara
INNER JOIN articles__revisions ar ON ar.id = ara.revision_id
WHERE
    ar.public_id = $1
    AND NOT EXISTS (
        SELECT 1
        FROM assets
        WHERE assets.sha512_hash = ara.sha512_hash
    );

-- name: GetAsset :one
SELECT assets.*
FROM assets
INNER JOIN articles__revisions__assets ara ON ara.sha512_hash = assets.sha512_hash
INNER JOIN articles__revisions ar ON ara.revision_id = ar.id
INNER JOIN articles a ON ar.article_id = a.id
WHERE
    (
        a.slug = $1
        OR ar.public_id = sqlc.arg(article_id)
        OR a.public_id = sqlc.arg(article_id)
    )
    AND ara.file_name = $2
    AND ar.published_at IS NOT NULL
ORDER BY ar.published_at DESC
LIMIT 1;

-- name: GetArticleFeed :many
WITH latest_revisions AS (
    SELECT
        a.id AS article_id,
        MAX(ar.published_at) AS latest_publish,
        MIN(ar.published_at) AS oldest_publish
    FROM articles a
    INNER JOIN articles__revisions ar ON a.id = ar.article_id
    WHERE
        ar.published_at IS NOT NULL
        AND ar.published_at < now()
    GROUP BY a.id
)
SELECT
    sqlc.embed(a),
    sqlc.embed(ar),
    CAST(lr.oldest_publish AS TIMESTAMPTZ) AS original_publish
FROM articles a
INNER JOIN latest_revisions lr ON a.id = lr.article_id
INNER JOIN articles__revisions ar ON
    a.id = ar.article_id
    AND lr.latest_publish = ar.published_at
    AND ar.published_at IS NOT NULL
ORDER BY ar.published_at DESC
LIMIT $1
OFFSET $2;
