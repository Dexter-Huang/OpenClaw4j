-- name: FindDefaultWorkspaceByAccountID :one
SELECT id, workspace_id, account_id, status, name, description, config,
       gmt_create, gmt_modified, creator, modifier, tenant_id
FROM workspace
WHERE account_id = $1 AND status <> 0
ORDER BY id ASC
LIMIT 1;

-- name: CreateWorkspace :exec
INSERT INTO workspace (
    workspace_id, account_id, status, name, description, config,
    gmt_create, gmt_modified, creator, modifier, tenant_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: FindActiveWorkspaceByIDAndAccountID :one
SELECT id, workspace_id, account_id, status, name, description, config,
       gmt_create, gmt_modified, creator, modifier, tenant_id
FROM workspace
WHERE workspace_id = $1 AND account_id = $2 AND status <> 0
LIMIT 1;

-- name: FindActiveWorkspaceByNameAndAccountID :one
SELECT id, workspace_id, account_id, status, name, description, config,
       gmt_create, gmt_modified, creator, modifier, tenant_id
FROM workspace
WHERE name = $1 AND account_id = $2 AND status <> 0
LIMIT 1;

-- name: CountActiveWorkspacesByAccountID :one
SELECT COUNT(*)::bigint
FROM workspace
WHERE account_id = $1 AND status <> 0;

-- name: ListActiveWorkspacesByAccountID :many
SELECT id, workspace_id, account_id, status, name, description, config,
       gmt_create, gmt_modified, creator, modifier, tenant_id
FROM workspace
WHERE account_id = $1 AND status <> 0
ORDER BY id DESC
LIMIT $2 OFFSET $3;

-- name: UpdateWorkspace :exec
UPDATE workspace
SET name = $3, description = $4, config = $5, modifier = $6, gmt_modified = $7
WHERE workspace_id = $1 AND account_id = $2 AND status <> 0;

-- name: SoftDeleteWorkspace :exec
UPDATE workspace
SET status = 0, modifier = $3, gmt_modified = $4
WHERE workspace_id = $1 AND account_id = $2 AND status <> 0;
