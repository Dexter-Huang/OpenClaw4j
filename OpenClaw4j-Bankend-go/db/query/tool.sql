-- name: CreateTool :one
INSERT INTO tool (plugin_id, tool_id, workspace_id, status, enabled, test_status, name, description, config, api_schema, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
RETURNING id;

-- name: FindToolByIDAndWorkspace :one
SELECT id, plugin_id, tool_id, workspace_id, status, enabled, test_status, name, description, config, api_schema, gmt_create, gmt_modified, creator, modifier, tenant_id
FROM tool WHERE id = $1 AND workspace_id = $2 LIMIT 1;

-- name: ListToolsByWorkspace :many
SELECT id, plugin_id, tool_id, workspace_id, status, enabled, test_status, name, description, config, api_schema, gmt_create, gmt_modified, creator, modifier, tenant_id
FROM tool
WHERE workspace_id = $1
  AND ($2::text = '' OR name ILIKE '%' || $2::text || '%')
  AND ($3::text = '' OR plugin_id = $3::text)
ORDER BY gmt_modified DESC;

-- name: UpdateTool :exec
UPDATE tool
SET plugin_id = $3, status = $4, enabled = $5, test_status = $6, name = $7, description = $8,
    config = $9, api_schema = $10, gmt_modified = $11, modifier = $12
WHERE id = $1 AND workspace_id = $2;

-- name: DeleteTool :exec
DELETE FROM tool WHERE id = $1 AND workspace_id = $2;
