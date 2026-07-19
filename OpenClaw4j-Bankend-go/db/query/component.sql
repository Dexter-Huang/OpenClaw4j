-- name: CreateApplicationComponent :exec
INSERT INTO application_component (gmt_create,gmt_modified,code,name,workspace_id,type,app_id,config,description,status,creator,modifier,need_update,tenant_id)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,0,$13);

-- name: FindActiveApplicationComponent :one
SELECT id,gmt_create,gmt_modified,code,name,workspace_id,type,app_id,config,description,status,creator,modifier,need_update,tenant_id
FROM application_component WHERE workspace_id=$1 AND code=$2 AND status<>0 LIMIT 1;

-- name: FindActiveApplicationComponentByAppID :one
SELECT id,gmt_create,gmt_modified,code,name,workspace_id,type,app_id,config,description,status,creator,modifier,need_update,tenant_id
FROM application_component WHERE workspace_id=$1 AND app_id=$2 AND status<>0 LIMIT 1;

-- name: ListActiveApplicationComponents :many
SELECT id,gmt_create,gmt_modified,code,name,workspace_id,type,app_id,config,description,status,creator,modifier,need_update,tenant_id
FROM application_component
WHERE workspace_id=$1
  AND ($2::text='' OR name ILIKE '%' || $2::text || '%')
  AND ($3::text='' OR type=$3)
  AND ($4::text='' OR app_id=$4)
  AND ($5::smallint < 0 OR status=$5)
  AND status<>0
ORDER BY gmt_modified DESC;

-- name: UpdateApplicationComponent :exec
UPDATE application_component SET name=$3,type=$4,app_id=$5,config=$6,description=$7,status=$8,gmt_modified=$9,modifier=$10
WHERE workspace_id=$1 AND code=$2 AND status<>0;

-- name: DeleteApplicationComponent :exec
UPDATE application_component SET status=0,gmt_modified=$3,modifier=$4
WHERE workspace_id=$1 AND code=$2 AND status<>0;
