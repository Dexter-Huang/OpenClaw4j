-- name: CreateProvider :exec
INSERT INTO provider (
    workspace_id, icon, name, description, provider, enable, source, credential,
    supported_model_types, protocol, gmt_create, gmt_modified, creator, modifier, tenant_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15);

-- name: FindProviderByCodeAndWorkspace :one
SELECT id, workspace_id, icon, name, description, provider, enable, source, credential,
       supported_model_types, protocol, gmt_create, gmt_modified, creator, modifier, tenant_id
FROM provider
WHERE provider = $1 AND workspace_id = $2
LIMIT 1;

-- name: ListProvidersByWorkspace :many
SELECT id, workspace_id, icon, name, description, provider, enable, source, credential,
       supported_model_types, protocol, gmt_create, gmt_modified, creator, modifier, tenant_id
FROM provider
WHERE workspace_id = $1
  AND ($2::text = '' OR name ILIKE '%' || $2::text || '%')
ORDER BY id DESC;

-- name: UpdateProvider :exec
UPDATE provider
SET icon = $3,
    name = $4,
    description = $5,
    enable = $6,
    credential = $7,
    supported_model_types = $8,
    protocol = $9,
    gmt_modified = $10,
    modifier = $11
WHERE provider = $1 AND workspace_id = $2;

-- name: DeleteProvider :exec
DELETE FROM provider
WHERE provider = $1 AND workspace_id = $2;

-- name: CountModelsByProviderAndWorkspace :one
SELECT COUNT(*)::bigint
FROM model
WHERE provider = $1 AND workspace_id = $2;
