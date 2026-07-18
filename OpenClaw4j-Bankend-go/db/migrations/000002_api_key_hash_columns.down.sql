DROP INDEX IF EXISTS uk_api_key_hash;
ALTER TABLE api_key DROP COLUMN IF EXISTS api_key_prefix;
ALTER TABLE api_key DROP COLUMN IF EXISTS api_key_hash;

