-- name: CreateModel :exec
INSERT INTO model (
    workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source,
    gmt_create, gmt_modified, creator, modifier, tenant_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15);

-- name: FindModelByProviderAndIDAndWorkspace :one
SELECT id, workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source,
       gmt_create, gmt_modified, creator, modifier, tenant_id
FROM model
WHERE provider = $1 AND model_id = $2 AND workspace_id = $3
LIMIT 1;

-- name: ListModelsByProviderAndWorkspace :many
SELECT id, workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source,
       gmt_create, gmt_modified, creator, modifier, tenant_id
FROM model
WHERE provider = $1 AND workspace_id = $2
ORDER BY id ASC;

-- name: ListModelsByWorkspace :many
SELECT id, workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source,
       gmt_create, gmt_modified, creator, modifier, tenant_id
FROM model
WHERE workspace_id = $1
ORDER BY id ASC;

-- name: UpdateModel :exec
UPDATE model
SET icon = $4,
    name = $5,
    enable = $6,
    tags = $7,
    gmt_modified = $8,
    modifier = $9
WHERE provider = $1 AND model_id = $2 AND workspace_id = $3;

-- name: DeleteModel :exec
DELETE FROM model
WHERE provider = $1 AND model_id = $2 AND workspace_id = $3;
