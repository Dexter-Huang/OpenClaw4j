-- name: CreateTokenSession :exec
INSERT INTO auth_token_session (
  token_id, account_id, token_type, token_hash, expires_at,
  revoked, source, caller_ip, user_agent, tenant_id
) VALUES (
  $1, $2, $3, $4, $5,
  0, $6, $7, $8, $9
);

-- name: FindActiveTokenSessionByHash :one
SELECT id, token_id, account_id, token_type, token_hash, expires_at, revoked,
       source, caller_ip, user_agent, gmt_create, gmt_modified, tenant_id
FROM auth_token_session
WHERE token_hash = $1
  AND token_type = $2
  AND revoked = 0
  AND expires_at > $3
LIMIT 1;

-- name: RevokeTokenSessionByHash :exec
UPDATE auth_token_session
SET revoked = 1,
    gmt_modified = CURRENT_TIMESTAMP
WHERE token_hash = $1;

-- name: RevokeAccountTokens :exec
UPDATE auth_token_session
SET revoked = 1,
    gmt_modified = CURRENT_TIMESTAMP
WHERE account_id = $1
  AND token_type = $2
  AND revoked = 0;

-- name: DeleteExpiredTokenSessions :exec
DELETE FROM auth_token_session
WHERE expires_at <= $1;

