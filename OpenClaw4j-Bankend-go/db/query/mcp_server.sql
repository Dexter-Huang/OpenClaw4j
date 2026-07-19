-- name: CreateMcpServer :exec
INSERT INTO mcp_server (gmt_create,gmt_modified,server_code,name,description,source,type,deploy_config,workspace_id,account_id,status,detail_config,install_type,tenant_id) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14);
-- name: FindActiveMcpServer :one
SELECT id,gmt_create,gmt_modified,server_code,name,description,source,deploy_env,type,deploy_config,workspace_id,account_id,status,biz_type,detail_config,host,install_type,tenant_id FROM mcp_server WHERE server_code=$1 AND workspace_id=$2 AND status<>3 LIMIT 1;
-- name: ListActiveMcpServers :many
SELECT id,gmt_create,gmt_modified,server_code,name,description,source,deploy_env,type,deploy_config,workspace_id,account_id,status,biz_type,detail_config,host,install_type,tenant_id FROM mcp_server WHERE workspace_id=$1 AND ($2::text='' OR name ILIKE '%' || $2::text || '%') AND status<>3 ORDER BY gmt_modified DESC;
-- name: UpdateMcpServer :exec
UPDATE mcp_server SET gmt_modified=$3,name=$4,description=$5,source=$6,type=$7,deploy_config=$8,status=$9,detail_config=$10,install_type=$11 WHERE server_code=$1 AND workspace_id=$2 AND status<>3;
-- name: DeleteMcpServer :exec
UPDATE mcp_server SET status=3,gmt_modified=$3 WHERE server_code=$1 AND workspace_id=$2 AND status<>3;
