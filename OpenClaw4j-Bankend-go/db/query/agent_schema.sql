-- name: CreateAgentSchema :one
INSERT INTO agent_schema (
    agent_id, workspace_id, name, description, type, instruction, input_keys, output_key,
    handle, sub_agents, yaml_schema, status, enabled, gmt_create, gmt_modified, creator, modifier, tenant_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
RETURNING id;

-- name: FindAgentSchemaByIDAndWorkspace :one
SELECT id, agent_id, workspace_id, name, description, type, instruction, input_keys, output_key,
       handle, sub_agents, yaml_schema, status, enabled, gmt_create, gmt_modified, creator, modifier, tenant_id
FROM agent_schema
WHERE id = $1 AND workspace_id = $2
LIMIT 1;

-- name: FindAgentSchemaByNameAndWorkspace :one
SELECT id, agent_id, workspace_id, name, description, type, instruction, input_keys, output_key,
       handle, sub_agents, yaml_schema, status, enabled, gmt_create, gmt_modified, creator, modifier, tenant_id
FROM agent_schema
WHERE name = $1 AND workspace_id = $2
LIMIT 1;

-- name: ListAgentSchemasByWorkspace :many
SELECT id, agent_id, workspace_id, name, description, type, instruction, input_keys, output_key,
       handle, sub_agents, yaml_schema, status, enabled, gmt_create, gmt_modified, creator, modifier, tenant_id
FROM agent_schema
WHERE workspace_id = $1 AND ($2::text = '' OR name ILIKE '%' || $2::text || '%')
ORDER BY gmt_modified DESC;

-- name: UpdateAgentSchema :exec
UPDATE agent_schema
SET name = $3, description = $4, type = $5, instruction = $6, input_keys = $7, output_key = $8,
    handle = $9, sub_agents = $10, yaml_schema = $11, status = $12, enabled = $13,
    gmt_modified = $14, modifier = $15
WHERE id = $1 AND workspace_id = $2;

-- name: DeleteAgentSchema :exec
DELETE FROM agent_schema WHERE id = $1 AND workspace_id = $2;
