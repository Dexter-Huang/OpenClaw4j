CREATE TABLE IF NOT EXISTS auth_token_session (
  id BIGSERIAL NOT NULL,
  token_id varchar(64) NOT NULL,
  account_id varchar(64) NOT NULL,
  token_type varchar(32) NOT NULL,
  token_hash varchar(128) NOT NULL,
  expires_at timestamp NOT NULL,
  revoked smallint NOT NULL DEFAULT 0,
  source varchar(64) DEFAULT NULL,
  caller_ip varchar(64) DEFAULT NULL,
  user_agent varchar(512) DEFAULT NULL,
  gmt_create timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  gmt_modified timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  tenant_id bigint DEFAULT 0,
  PRIMARY KEY (id),
  CONSTRAINT uk_auth_token_session_token_hash UNIQUE (token_hash)
);

ALTER TABLE api_key ADD COLUMN IF NOT EXISTS api_key_hash varchar(128);
ALTER TABLE api_key ADD COLUMN IF NOT EXISTS api_key_prefix varchar(32);

