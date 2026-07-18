ALTER TABLE api_key ADD COLUMN IF NOT EXISTS api_key_hash varchar(128);
ALTER TABLE api_key ADD COLUMN IF NOT EXISTS api_key_prefix varchar(32);

CREATE UNIQUE INDEX IF NOT EXISTS uk_api_key_hash
  ON api_key (api_key_hash)
  WHERE api_key_hash IS NOT NULL;

