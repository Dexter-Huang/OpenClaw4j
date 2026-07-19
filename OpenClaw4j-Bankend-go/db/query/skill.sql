-- name: CreateSkill :exec
INSERT INTO skill (skill_code,workspace_id,account_id,name,description,source,status,tags,gmt_create,gmt_modified,creator,modifier) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12);
-- name: CreateSkillVersion :exec
INSERT INTO skill_version (skill_code,workspace_id,version,description,main_file_path,manifest,content_hash,storage_type,storage_bucket,storage_prefix,package_object_key,file_count,total_size_bytes,status,gmt_create,gmt_modified,creator,modifier) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18);
-- name: FindActiveSkill :one
SELECT id,skill_code,workspace_id,account_id,name,description,source,status,tags,gmt_create,gmt_modified,creator,modifier,tenant_id FROM skill WHERE skill_code=$1 AND workspace_id=$2 AND status<>0 LIMIT 1;
-- name: FindActiveSkillVersion :one
SELECT id,skill_code,workspace_id,version,description,main_file_path,manifest,content_hash,storage_type,storage_bucket,storage_prefix,package_object_key,file_count,total_size_bytes,status,gmt_create,gmt_modified,creator,modifier,tenant_id FROM skill_version WHERE skill_code=$1 AND workspace_id=$2 AND version=$3 AND status<>0 LIMIT 1;
-- name: FindLatestActiveSkillVersion :one
SELECT id,skill_code,workspace_id,version,description,main_file_path,manifest,content_hash,storage_type,storage_bucket,storage_prefix,package_object_key,file_count,total_size_bytes,status,gmt_create,gmt_modified,creator,modifier,tenant_id FROM skill_version WHERE skill_code=$1 AND workspace_id=$2 AND status<>0 ORDER BY gmt_modified DESC LIMIT 1;
-- name: ListActiveSkills :many
SELECT id,skill_code,workspace_id,account_id,name,description,source,status,tags,gmt_create,gmt_modified,creator,modifier,tenant_id FROM skill WHERE workspace_id=$1 AND ($2::text='' OR name ILIKE '%' || $2::text || '%') AND ($3::smallint < 0 OR status=$3) AND status<>0 ORDER BY gmt_modified DESC;
-- name: UpdateSkill :exec
UPDATE skill SET name=$3,description=$4,source=$5,status=$6,tags=$7,gmt_modified=$8,modifier=$9 WHERE skill_code=$1 AND workspace_id=$2 AND status<>0;
-- name: UpdateSkillVersion :exec
UPDATE skill_version SET description=$4,main_file_path=$5,manifest=$6,content_hash=$7,storage_type=$8,storage_bucket=$9,storage_prefix=$10,package_object_key=$11,file_count=$12,total_size_bytes=$13,status=$14,gmt_modified=$15,modifier=$16 WHERE skill_code=$1 AND workspace_id=$2 AND version=$3 AND status<>0;
-- name: MarkSkillDeleted :exec
UPDATE skill SET status=0,gmt_modified=$3,modifier=$4 WHERE skill_code=$1 AND workspace_id=$2 AND status<>0;
-- name: MarkSkillVersionsDeleted :exec
UPDATE skill_version SET status=0,gmt_modified=$3,modifier=$4 WHERE skill_code=$1 AND workspace_id=$2 AND status<>0;
