-- name: FindDefaultWorkspaceByAccountID :one
SELECT id, workspace_id, account_id, status, name, description, config,
       gmt_create, gmt_modified, creator, modifier, tenant_id
FROM workspace
WHERE account_id = $1 AND status <> 0
ORDER BY id ASC
LIMIT 1;

