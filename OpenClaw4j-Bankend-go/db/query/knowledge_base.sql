-- name: CreateKnowledgeBase :exec
INSERT INTO knowledge_base (workspace_id, kb_id, type, status, name, description, process_config, index_config, search_config, total_docs, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15);

-- name: FindActiveKnowledgeBase :one
SELECT id, workspace_id, kb_id, type, status, name, description, process_config, index_config, search_config, total_docs, gmt_create, gmt_modified, creator, modifier, tenant_id FROM knowledge_base WHERE kb_id=$1 AND workspace_id=$2 AND status<>0 LIMIT 1;

-- name: ListActiveKnowledgeBases :many
SELECT id, workspace_id, kb_id, type, status, name, description, process_config, index_config, search_config, total_docs, gmt_create, gmt_modified, creator, modifier, tenant_id FROM knowledge_base WHERE workspace_id=$1 AND ($2::text='' OR name ILIKE '%' || $2::text || '%') AND status<>0 ORDER BY gmt_modified DESC;

-- name: UpdateKnowledgeBase :exec
UPDATE knowledge_base SET type=$3,name=$4,description=$5,process_config=$6,index_config=$7,search_config=$8,gmt_modified=$9,modifier=$10 WHERE kb_id=$1 AND workspace_id=$2 AND status<>0;

-- name: DeleteKnowledgeBase :exec
UPDATE knowledge_base SET status=0,gmt_modified=$3,modifier=$4 WHERE kb_id=$1 AND workspace_id=$2 AND status<>0;
