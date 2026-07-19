-- name: CreateDocument :exec
INSERT INTO document (workspace_id,kb_id,doc_id,type,status,enabled,name,format,size,metadata,index_status,path,parsed_path,process_config,source,error,gmt_create,gmt_modified,creator,modifier,tenant_id)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21);

-- name: FindActiveDocument :one
SELECT id,workspace_id,kb_id,doc_id,type,status,enabled,name,format,size,metadata,index_status,path,parsed_path,process_config,source,error,gmt_create,gmt_modified,creator,modifier,tenant_id
FROM document WHERE workspace_id=$1 AND kb_id=$2 AND doc_id=$3 AND status<>0 LIMIT 1;

-- name: FindActiveDocumentByDocID :one
SELECT id,workspace_id,kb_id,doc_id,type,status,enabled,name,format,size,metadata,index_status,path,parsed_path,process_config,source,error,gmt_create,gmt_modified,creator,modifier,tenant_id
FROM document WHERE workspace_id=$1 AND doc_id=$2 AND status<>0 LIMIT 1;

-- name: ListActiveDocuments :many
SELECT id,workspace_id,kb_id,doc_id,type,status,enabled,name,format,size,metadata,index_status,path,parsed_path,process_config,source,error,gmt_create,gmt_modified,creator,modifier,tenant_id
FROM document WHERE workspace_id=$1 AND kb_id=$2 AND ($3::text='' OR name ILIKE '%' || $3::text || '%') AND ($4::smallint < 0 OR index_status=$4) AND status<>0 ORDER BY gmt_modified DESC;

-- name: UpdateDocument :exec
UPDATE document SET enabled=$4,name=$5,format=$6,size=$7,metadata=$8,index_status=$9,path=$10,parsed_path=$11,process_config=$12,source=$13,error=$14,gmt_modified=$15,modifier=$16
WHERE workspace_id=$1 AND kb_id=$2 AND doc_id=$3 AND status<>0;

-- name: DeleteDocument :exec
UPDATE document SET status=0,gmt_modified=$4,modifier=$5 WHERE workspace_id=$1 AND kb_id=$2 AND doc_id=$3 AND status<>0;

-- name: CreateDocumentChunk :exec
INSERT INTO document_chunk (workspace_id,kb_id,doc_id,chunk_id,doc_name,title,text,score,page_number,enabled,status,gmt_create,gmt_modified,creator,modifier,tenant_id)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,1,$11,$12,$13,$14,$15);

-- name: FindActiveDocumentChunk :one
SELECT id,workspace_id,kb_id,doc_id,chunk_id,doc_name,title,text,score,page_number,enabled,status,gmt_create,gmt_modified,creator,modifier,tenant_id
FROM document_chunk WHERE workspace_id=$1 AND doc_id=$2 AND chunk_id=$3 AND status<>0 LIMIT 1;

-- name: ListActiveDocumentChunks :many
SELECT id,workspace_id,kb_id,doc_id,chunk_id,doc_name,title,text,score,page_number,enabled,status,gmt_create,gmt_modified,creator,modifier,tenant_id
FROM document_chunk WHERE workspace_id=$1 AND doc_id=$2 AND status<>0 ORDER BY gmt_modified DESC;

-- name: UpdateDocumentChunk :exec
UPDATE document_chunk SET title=$4,text=$5,score=$6,page_number=$7,enabled=$8,gmt_modified=$9,modifier=$10
WHERE workspace_id=$1 AND doc_id=$2 AND chunk_id=$3 AND status<>0;

-- name: UpdateDocumentChunksEnabled :exec
UPDATE document_chunk SET enabled=$4,gmt_modified=$5,modifier=$6
WHERE workspace_id=$1 AND doc_id=$2 AND chunk_id=ANY($3::text[]) AND status<>0;

-- name: DeleteDocumentChunks :exec
UPDATE document_chunk SET status=0,gmt_modified=$4,modifier=$5
WHERE workspace_id=$1 AND doc_id=$2 AND chunk_id=ANY($3::text[]) AND status<>0;
