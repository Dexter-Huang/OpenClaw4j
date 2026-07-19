CREATE TABLE IF NOT EXISTS document_chunk (
  id BIGSERIAL PRIMARY KEY,
  workspace_id varchar(64) NOT NULL,
  kb_id varchar(64) NOT NULL,
  doc_id varchar(64) NOT NULL,
  chunk_id varchar(64) NOT NULL,
  doc_name varchar(255) NOT NULL,
  title varchar(512) NOT NULL DEFAULT '',
  text text NOT NULL,
  score double precision DEFAULT NULL,
  page_number integer DEFAULT NULL,
  enabled smallint NOT NULL DEFAULT 1,
  status smallint NOT NULL DEFAULT 1,
  gmt_create timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  gmt_modified timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  creator varchar(64) NOT NULL DEFAULT '',
  modifier varchar(64) NOT NULL DEFAULT '',
  tenant_id bigint DEFAULT 0,
  CONSTRAINT uk_document_chunk_workspace_doc_chunk UNIQUE (workspace_id, doc_id, chunk_id)
);

CREATE INDEX IF NOT EXISTS idx_document_chunk_workspace_doc_active
  ON document_chunk (workspace_id, doc_id, status, gmt_modified DESC);
