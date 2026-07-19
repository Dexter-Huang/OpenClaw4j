-- name: CreateApplication :exec
INSERT INTO application (
    workspace_id, app_id, name, description, icon, source, type, status,
    gmt_create, gmt_modified, creator, modifier, tenant_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13);

-- name: CreateApplicationVersion :exec
INSERT INTO application_version (
    app_id, workspace_id, config, status, version, description,
    gmt_create, gmt_modified, creator, modifier, tenant_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: FindActiveApplicationByIDAndWorkspace :one
SELECT id, workspace_id, app_id, name, description, icon, source, type, status,
       gmt_create, gmt_modified, creator, modifier, tenant_id
FROM application
WHERE app_id = $1 AND workspace_id = $2 AND status <> 0
LIMIT 1;

-- name: FindActiveApplicationByNameAndWorkspace :one
SELECT id, workspace_id, app_id, name, description, icon, source, type, status,
       gmt_create, gmt_modified, creator, modifier, tenant_id
FROM application
WHERE name = $1 AND workspace_id = $2 AND status <> 0
LIMIT 1;

-- name: FindLatestActiveApplicationVersion :one
SELECT id, app_id, workspace_id, config, status, version, description,
       gmt_create, gmt_modified, creator, modifier, tenant_id
FROM application_version
WHERE app_id = $1 AND workspace_id = $2 AND status <> 0
ORDER BY id DESC
LIMIT 1;

-- name: FindLastPublishedApplicationVersion :one
SELECT id, app_id, workspace_id, config, status, version, description,
       gmt_create, gmt_modified, creator, modifier, tenant_id
FROM application_version
WHERE app_id = $1 AND workspace_id = $2 AND status = 2
ORDER BY id DESC
LIMIT 1;

-- name: FindApplicationVersionByVersion :one
SELECT id, app_id, workspace_id, config, status, version, description,
       gmt_create, gmt_modified, creator, modifier, tenant_id
FROM application_version
WHERE app_id = $1 AND workspace_id = $2 AND version = $3 AND status <> 0
LIMIT 1;

-- name: ListActiveApplicationsByWorkspace :many
SELECT id, workspace_id, app_id, name, description, icon, source, type, status,
       gmt_create, gmt_modified, creator, modifier, tenant_id
FROM application
WHERE workspace_id = $1
  AND ($2::text = '' OR name ILIKE '%' || $2::text || '%')
  AND ($3::text = '' OR type = $3::text)
  AND ($4::smallint = -1 OR status = $4::smallint)
  AND status <> 0
ORDER BY id DESC
LIMIT $5 OFFSET $6;

-- name: CountActiveApplicationsByWorkspace :one
SELECT COUNT(*)::bigint
FROM application
WHERE workspace_id = $1
  AND ($2::text = '' OR name ILIKE '%' || $2::text || '%')
  AND ($3::text = '' OR type = $3::text)
  AND ($4::smallint = -1 OR status = $4::smallint)
  AND status <> 0;

-- name: ListActiveApplicationVersions :many
SELECT id, app_id, workspace_id, config, status, version, description,
       gmt_create, gmt_modified, creator, modifier, tenant_id
FROM application_version
WHERE app_id = $1 AND workspace_id = $2
  AND ($3::smallint = -1 OR status = $3::smallint)
  AND status <> 0
ORDER BY id DESC
LIMIT $4 OFFSET $5;

-- name: CountActiveApplicationVersions :one
SELECT COUNT(*)::bigint
FROM application_version
WHERE app_id = $1 AND workspace_id = $2
  AND ($3::smallint = -1 OR status = $3::smallint)
  AND status <> 0;

-- name: UpdateApplication :exec
UPDATE application
SET name = $3, description = $4, icon = $5, type = $6, status = $7,
    gmt_modified = $8, modifier = $9
WHERE app_id = $1 AND workspace_id = $2 AND status <> 0;

-- name: UpdateApplicationVersion :exec
UPDATE application_version
SET config = $4, status = $5, description = $6, gmt_modified = $7, modifier = $8
WHERE app_id = $1 AND workspace_id = $2 AND version = $3 AND status <> 0;

-- name: MarkApplicationDeleted :exec
UPDATE application
SET status = 0, gmt_modified = $3, modifier = $4
WHERE app_id = $1 AND workspace_id = $2 AND status <> 0;

-- name: MarkApplicationVersionsDeleted :exec
UPDATE application_version
SET status = 0, gmt_modified = $3, modifier = $4
WHERE app_id = $1 AND workspace_id = $2 AND status <> 0;
