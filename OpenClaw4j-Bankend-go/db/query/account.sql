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

-- name: CreateAccount :exec
INSERT INTO account (
    account_id, username, email, mobile, password, nickname, icon,
    type, status, gmt_create, gmt_modified, creator, modifier, tenant_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14);

-- name: UpdateAccount :exec
UPDATE account
SET email = $2, mobile = $3, password = $4, nickname = $5, icon = $6,
    gmt_modified = $7, modifier = $8
WHERE account_id = $1 AND status <> 0;

-- name: SoftDeleteAccount :exec
UPDATE account
SET status = 0, gmt_modified = $2, modifier = $3
WHERE account_id = $1 AND status <> 0;

-- name: ListActiveUserAccounts :many
SELECT id, account_id, username, email, mobile, password, nickname, icon,
       type, status, gmt_create, gmt_modified, gmt_last_login, creator, modifier, tenant_id
FROM account
WHERE type = 'user' AND status <> 0 AND ($1::text = '' OR username ILIKE '%' || $1::text || '%')
ORDER BY id DESC
LIMIT $2 OFFSET $3;

-- name: CountActiveUserAccounts :one
SELECT COUNT(*)::bigint
FROM account
WHERE type = 'user' AND status <> 0 AND ($1::text = '' OR username ILIKE '%' || $1::text || '%');
