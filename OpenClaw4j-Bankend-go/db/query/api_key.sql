-- name: FindActiveAPIKeyByEncryptedKey :one
SELECT id, account_id, api_key, status, description,
       gmt_create, gmt_modified, creator, modifier, tenant_id
FROM api_key
WHERE api_key = $1 AND status <> 0
LIMIT 1;

-- name: FindActiveAPIKeyByHash :one
SELECT id, account_id, api_key, status, description,
       gmt_create, gmt_modified, creator, modifier, tenant_id
FROM api_key
WHERE api_key_hash = $1 AND status <> 0
LIMIT 1;

-- name: CreateAPIKey :one
INSERT INTO api_key (
    account_id, api_key, status, description,
    gmt_create, gmt_modified, creator, modifier, tenant_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id;

-- name: FindActiveAPIKeyByIDAndAccountID :one
SELECT id, account_id, api_key, status, description,
       gmt_create, gmt_modified, creator, modifier, tenant_id
FROM api_key
WHERE id = $1 AND account_id = $2 AND status <> 0
LIMIT 1;

-- name: UpdateAPIKeyDescription :exec
UPDATE api_key
SET description = $3,
    modifier = $4,
    gmt_modified = $5
WHERE id = $1 AND account_id = $2 AND status <> 0;

-- name: SoftDeleteAPIKey :exec
UPDATE api_key
SET status = 0,
    modifier = $3,
    gmt_modified = $4
WHERE id = $1 AND account_id = $2 AND status <> 0;

-- name: CountActiveAPIKeysByAccountID :one
SELECT COUNT(*)::bigint
FROM api_key
WHERE account_id = $1 AND status <> 0;

-- name: ListActiveAPIKeysByAccountID :many
SELECT id, account_id, api_key, status, description,
       gmt_create, gmt_modified, creator, modifier, tenant_id
FROM api_key
WHERE account_id = $1 AND status <> 0
ORDER BY id DESC
LIMIT $2 OFFSET $3;
