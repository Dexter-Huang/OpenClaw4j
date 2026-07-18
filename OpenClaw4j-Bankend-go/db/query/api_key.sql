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

