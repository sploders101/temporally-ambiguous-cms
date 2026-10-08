-- name: CreateUser :one
INSERT INTO users (
    username,
    email
) VALUES ($1, $2)
RETURNING *;

-- name: SeedUser :one
INSERT INTO users (
    username,
    email,
    password_hash
) VALUES ($1, $2, $3)
ON CONFLICT (username) DO NOTHING
RETURNING 1;

-- name: UpdateUserInfo :exec
UPDATE users
SET
    username = $2,
    email = $3
WHERE id = $1;

-- name: SetUserPassword :exec
UPDATE users
SET password_hash = $1
WHERE id = $2;

-- name: CreateUserSession :exec
INSERT INTO users__sessions (
    token_hash,
    user_id,
    expires
) VALUES ($1, $2, $3);

-- name: DeleteUserSession :exec
DELETE FROM users__sessions
WHERE token_hash = $1;

-- name: CreateOidcIdentity :one
INSERT INTO users__oidc_identities (
    user_id,
    issuer,
    subject
) VALUES ($1, $2, $3)
RETURNING *;

-- name: GetUser :one
SELECT
    users.*
FROM users
WHERE
    id = $1;

-- name: ListUsers :many
SELECT
    users.*
FROM users
LIMIT $1
OFFSET $2;

-- name: GetUserByOidc :one
SELECT
    sqlc.embed(users),
    sqlc.embed(oidc)
FROM users__oidc_identities oidc
INNER JOIN users ON oidc.user_id = users.id
WHERE
    oidc.issuer = $1
    AND oidc.subject = $2;

-- name: GetSessionDataByTokenHash :one
SELECT
    sqlc.embed(users),
    sqlc.embed(users__sessions)
FROM users__sessions
INNER JOIN users ON users.id = users__sessions.user_id
WHERE users__sessions.token_hash = $1;

-- name: GetUserByUsername :one
SELECT users.*
FROM users
WHERE users.username = $1;

-- name: AddSSHKey :one
INSERT INTO users__ssh_keys (
    user_id,
    name,
    public_key,
    fingerprint,
    expires_at
) VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetUserBySshKey :one
SELECT
    sqlc.embed(users),
    sqlc.embed(users__ssh_keys)
FROM users__ssh_keys
INNER JOIN users ON users__ssh_keys.user_id = users.id
WHERE users__ssh_keys.fingerprint = $1;

-- name: ListSSHKeysForUser :many
SELECT *
FROM users__ssh_keys
WHERE user_id = $1;

-- name: DeleteUserScopedSSHKey :exec
DELETE FROM users__ssh_keys
WHERE
    id = $1
    AND user_id = $2;

-- name: LogSSHAuthentication :exec
UPDATE users__ssh_keys
SET last_used_at = now()
WHERE fingerprint = $1;
