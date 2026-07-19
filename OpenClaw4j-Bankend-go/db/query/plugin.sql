-- name: CreatePlugin :exec
INSERT INTO plugin (plugin_id,workspace_id,type,status,name,description,config,source,gmt_create,gmt_modified,creator,modifier,tenant_id)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13);

-- name: FindActivePlugin :one
SELECT id,plugin_id,workspace_id,type,status,name,description,config,source,gmt_create,gmt_modified,creator,modifier,tenant_id
FROM plugin WHERE plugin_id=$1 AND workspace_id=$2 AND status<>0 LIMIT 1;

-- name: ListActivePlugins :many
SELECT id,plugin_id,workspace_id,type,status,name,description,config,source,gmt_create,gmt_modified,creator,modifier,tenant_id
FROM plugin WHERE workspace_id=$1 AND ($2::text='' OR name ILIKE '%' || $2::text || '%') AND ($3::smallint < 0 OR status=$3) AND status<>0 ORDER BY gmt_modified DESC;

-- name: UpdatePlugin :exec
UPDATE plugin SET type=$3,status=$4,name=$5,description=$6,config=$7,source=$8,gmt_modified=$9,modifier=$10
WHERE plugin_id=$1 AND workspace_id=$2 AND status<>0;

-- name: DeletePlugin :exec
DELETE FROM plugin WHERE plugin_id=$1 AND workspace_id=$2;

-- name: CreatePluginTool :exec
INSERT INTO tool (plugin_id,tool_id,workspace_id,status,enabled,test_status,name,description,config,api_schema,gmt_create,gmt_modified,creator,modifier,tenant_id)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15);

-- name: FindPluginTool :one
SELECT id,plugin_id,tool_id,workspace_id,status,enabled,test_status,name,description,config,api_schema,gmt_create,gmt_modified,creator,modifier,tenant_id
FROM tool WHERE plugin_id=$1 AND tool_id=$2 AND workspace_id=$3 LIMIT 1;

-- name: FindPluginToolByID :one
SELECT id,plugin_id,tool_id,workspace_id,status,enabled,test_status,name,description,config,api_schema,gmt_create,gmt_modified,creator,modifier,tenant_id
FROM tool WHERE tool_id=$1 AND workspace_id=$2 LIMIT 1;

-- name: ListPluginTools :many
SELECT id,plugin_id,tool_id,workspace_id,status,enabled,test_status,name,description,config,api_schema,gmt_create,gmt_modified,creator,modifier,tenant_id
FROM tool WHERE plugin_id=$1 AND workspace_id=$2 AND ($3::text='' OR name ILIKE '%' || $3::text || '%') ORDER BY gmt_modified DESC;

-- name: ListPluginToolsByIDs :many
SELECT id,plugin_id,tool_id,workspace_id,status,enabled,test_status,name,description,config,api_schema,gmt_create,gmt_modified,creator,modifier,tenant_id
FROM tool WHERE workspace_id=$1 AND tool_id = ANY($2::text[]) ORDER BY gmt_modified DESC;

-- name: UpdatePluginTool :exec
UPDATE tool SET status=$4,enabled=$5,test_status=$6,name=$7,description=$8,config=$9,api_schema=$10,gmt_modified=$11,modifier=$12
WHERE plugin_id=$1 AND tool_id=$2 AND workspace_id=$3;

-- name: DeletePluginTool :exec
DELETE FROM tool WHERE plugin_id=$1 AND tool_id=$2 AND workspace_id=$3;

-- name: DeletePluginTools :exec
DELETE FROM tool WHERE plugin_id=$1 AND workspace_id=$2;
