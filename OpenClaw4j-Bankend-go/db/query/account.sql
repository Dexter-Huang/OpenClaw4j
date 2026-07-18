-- name: FindActiveAccountByID :one
SELECT id, account_id, username, email, mobile, password, nickname, icon,
       type, status, gmt_create, gmt_modified, gmt_last_login, creator, modifier, tenant_id
FROM account
WHERE account_id = $1 AND status <> 0
LIMIT 1;

-- name: FindActiveAccountByUsername :one
SELECT id, account_id, username, email, mobile, password, nickname, icon,
       type, status, gmt_create, gmt_modified, gmt_last_login, creator, modifier, tenant_id
FROM account
WHERE username = $1 AND status <> 0
LIMIT 1;

-- name: UpdateAccountLastLogin :exec
UPDATE account
SET gmt_last_login = $2,
    gmt_modified = CURRENT_TIMESTAMP
WHERE account_id = $1 AND status <> 0;

