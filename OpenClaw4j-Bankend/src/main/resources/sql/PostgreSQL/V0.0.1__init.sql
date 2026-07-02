-- PostgreSQL initialization for OpenClaw4j.
-- Generated from the legacy schema shape for a new PostgreSQL/pgvector dev database.
CREATE EXTENSION IF NOT EXISTS vector;
CREATE SCHEMA IF NOT EXISTS openclaw_rag;

CREATE TABLE IF NOT EXISTS account (
  id BIGSERIAL NOT NULL,
  account_id varchar(64) NOT NULL,
  username varchar(255) NOT NULL,
  email varchar(255) DEFAULT NULL,
  mobile varchar(255) DEFAULT NULL,
  password varchar(255) NOT NULL,
  nickname varchar(255) DEFAULT NULL,
  icon varchar(255) DEFAULT NULL,
  type varchar(64) NOT NULL,
  status smallint NOT NULL DEFAULT 1,
  gmt_create timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  gmt_modified timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  gmt_last_login timestamp DEFAULT NULL,
  creator varchar(64) NOT NULL,
  modifier varchar(64) NOT NULL,
  tenant_id bigint DEFAULT 0,
  PRIMARY KEY (id),
  CONSTRAINT uk_account_id UNIQUE (account_id)
);
CREATE INDEX IF NOT EXISTS account_idx_email_password ON account (username);

INSERT INTO account VALUES (10000,'10000','saa','ken.lj.hz@gmail.com',NULL,'$argon2id$v=19$m=66536,t=2,p=1$KSDQowfZxDjKLqBtxFNRng$znU0oQFQs2shR9la4S11n7d0LpGApmSBXvDOXuhbR40',NULL,NULL,'admin',1,'2025-08-22 18:20:21','2025-08-22 18:20:21','2025-10-13 14:09:18','10000','10000',0) ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS agent_schema (
  id BIGSERIAL NOT NULL,
  agent_id varchar(64) DEFAULT NULL,
  workspace_id varchar(64) NOT NULL,
  name varchar(255) NOT NULL,
  description varchar(4096) DEFAULT NULL,
  type varchar(64) NOT NULL,
  instruction text,
  input_keys text,
  output_key varchar(255) DEFAULT NULL,
  handle text,
  sub_agents text,
  yaml_schema text,
  status varchar(64) NOT NULL DEFAULT 'DRAFT',
  enabled smallint NOT NULL DEFAULT 1,
  gmt_create timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  gmt_modified timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  creator varchar(64) NOT NULL,
  modifier varchar(64) NOT NULL,
  tenant_id bigint DEFAULT 0,
  PRIMARY KEY (id),
  CONSTRAINT uk_agent_id UNIQUE (agent_id)
);
CREATE INDEX IF NOT EXISTS agent_schema_idx_workspace_type ON agent_schema (workspace_id,type);
CREATE INDEX IF NOT EXISTS agent_schema_idx_workspace_status ON agent_schema (workspace_id,status);

CREATE TABLE IF NOT EXISTS api_key (
  id BIGSERIAL NOT NULL,
  account_id varchar(64) NOT NULL,
  api_key varchar(512) NOT NULL,
  status smallint NOT NULL DEFAULT 1,
  description varchar(4096) DEFAULT NULL,
  gmt_create timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  gmt_modified timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  creator varchar(64) NOT NULL,
  modifier varchar(64) NOT NULL,
  tenant_id bigint DEFAULT 0,
  PRIMARY KEY (id),
  CONSTRAINT uk_api_key UNIQUE (api_key)
);
CREATE INDEX IF NOT EXISTS api_key_idx_account_id ON api_key (account_id);

INSERT INTO api_key VALUES (10000,'10000','+Ae6iTvRFwv7auIV/RN5vxanWB07uxn3CH9Za7EPTMA9Mq4eNRK8K0sprMrUEaYM',1,'11','2025-08-23 18:48:26','2025-08-23 18:48:26','10000','10000',0) ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS application (
  id BIGSERIAL NOT NULL,
  workspace_id varchar(64) NOT NULL,
  app_id varchar(64) NOT NULL,
  name varchar(255) NOT NULL,
  description varchar(4096) DEFAULT NULL,
  icon varchar(255) DEFAULT NULL,
  source varchar(64) NOT NULL,
  type varchar(64) NOT NULL,
  status smallint NOT NULL DEFAULT 1,
  gmt_create timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  gmt_modified timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  creator varchar(64) NOT NULL,
  modifier varchar(64) NOT NULL,
  tenant_id bigint DEFAULT 0,
  PRIMARY KEY (id),
  CONSTRAINT uk_app_id UNIQUE (app_id)
);
CREATE INDEX IF NOT EXISTS application_idx_workspace_type ON application (workspace_id,type);

CREATE TABLE IF NOT EXISTS application_component (
  id BIGSERIAL NOT NULL,
  gmt_create timestamp NOT NULL,
  gmt_modified timestamp NOT NULL,
  code varchar(64) NOT NULL,
  name varchar(128) NOT NULL,
  workspace_id varchar(64) NOT NULL,
  type varchar(64) NOT NULL,
  app_id varchar(64) DEFAULT NULL,
  config text,
  description varchar(4096) DEFAULT NULL,
  status smallint DEFAULT NULL,
  creator varchar(64) DEFAULT NULL,
  modifier varchar(64) DEFAULT NULL,
  need_update smallint DEFAULT NULL,
  tenant_id bigint DEFAULT 0,
  PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS application_component_idx_workspace_type_status_appcode ON application_component (workspace_id,type,app_id);

CREATE TABLE IF NOT EXISTS application_version (
  id BIGSERIAL NOT NULL,
  app_id varchar(64) NOT NULL,
  workspace_id varchar(64) NOT NULL,
  config text,
  status smallint NOT NULL,
  version varchar(32) NOT NULL DEFAULT '0.0.1',
  description varchar(4096) DEFAULT NULL,
  gmt_create timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  gmt_modified timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  creator varchar(64) NOT NULL,
  modifier varchar(64) NOT NULL,
  tenant_id bigint DEFAULT 0,
  PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS application_version_idx_workspace_app_version ON application_version (workspace_id,app_id,version);

CREATE TABLE IF NOT EXISTS document (
  id BIGSERIAL NOT NULL,
  workspace_id varchar(64) NOT NULL,
  kb_id varchar(64) NOT NULL,
  doc_id varchar(64) NOT NULL,
  type varchar(64) NOT NULL,
  status smallint NOT NULL DEFAULT 1,
  enabled smallint NOT NULL DEFAULT 1,
  name varchar(255) NOT NULL,
  format varchar(64) NOT NULL,
  size bigint NOT NULL DEFAULT 0,
  metadata text,
  index_status smallint NOT NULL DEFAULT 1,
  path varchar(512) NOT NULL,
  parsed_path varchar(512) DEFAULT NULL,
  process_config text,
  source varchar(255) DEFAULT NULL,
  error text,
  gmt_create timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  gmt_modified timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  creator varchar(64) NOT NULL,
  modifier varchar(64) NOT NULL,
  tenant_id bigint DEFAULT 0,
  PRIMARY KEY (id),
  CONSTRAINT uk_document_id UNIQUE (doc_id)
);
CREATE INDEX IF NOT EXISTS document_idx_workspace_kb ON document (workspace_id,kb_id);

CREATE TABLE IF NOT EXISTS knowledge_base (
  id BIGSERIAL NOT NULL,
  workspace_id varchar(64) NOT NULL,
  kb_id varchar(64) NOT NULL,
  type varchar(64) NOT NULL,
  status smallint NOT NULL DEFAULT 1,
  name varchar(255) NOT NULL,
  description varchar(4096) DEFAULT NULL,
  process_config text,
  index_config text,
  search_config text,
  total_docs bigint NOT NULL DEFAULT 0,
  gmt_create timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  gmt_modified timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  creator varchar(64) NOT NULL,
  modifier varchar(64) NOT NULL,
  tenant_id bigint DEFAULT 0,
  PRIMARY KEY (id),
  CONSTRAINT uk_kb_id UNIQUE (kb_id)
);
CREATE INDEX IF NOT EXISTS knowledge_base_idx_workspace_status_name ON knowledge_base (workspace_id);

CREATE TABLE IF NOT EXISTS mcp_server (
  id BIGSERIAL NOT NULL,
  gmt_create timestamp NOT NULL,
  gmt_modified timestamp NOT NULL,
  server_code varchar(64) NOT NULL,
  name varchar(64) NOT NULL,
  description varchar(1024) DEFAULT NULL,
  source varchar(128) DEFAULT NULL,
  deploy_env varchar(16) DEFAULT NULL,
  type varchar(32) NOT NULL,
  deploy_config text NOT NULL,
  workspace_id varchar(64) DEFAULT NULL,
  account_id varchar(64) DEFAULT NULL,
  status smallint NOT NULL,
  biz_type varchar(512) DEFAULT NULL,
  detail_config text,
  host varchar(1024) DEFAULT NULL,
  install_type varchar(32) DEFAULT NULL,
  tenant_id bigint DEFAULT 0,
  PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS mcp_server_idx_code ON mcp_server (server_code);
CREATE INDEX IF NOT EXISTS mcp_server_idx_server_name ON mcp_server (name);
CREATE INDEX IF NOT EXISTS mcp_server_idx_name_status ON mcp_server (status,name);
CREATE INDEX IF NOT EXISTS mcp_server_idx_type ON mcp_server (type);

CREATE TABLE IF NOT EXISTS model (
  id BIGSERIAL NOT NULL,
  workspace_id varchar(64) DEFAULT NULL,
  icon varchar(255) DEFAULT NULL,
  name varchar(100) DEFAULT NULL,
  type varchar(100) DEFAULT 'LLM',
  mode varchar(100) DEFAULT 'chat',
  model_id varchar(100) NOT NULL,
  provider varchar(100) NOT NULL,
  enable smallint DEFAULT 1,
  tags varchar(255) DEFAULT NULL,
  source varchar(100) NOT NULL DEFAULT 'preset',
  gmt_create timestamp DEFAULT CURRENT_TIMESTAMP,
  gmt_modified timestamp DEFAULT CURRENT_TIMESTAMP,
  creator varchar(64) DEFAULT NULL,
  modifier varchar(64) DEFAULT NULL,
  tenant_id bigint DEFAULT 0,
  PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS model_idx_account_workspace_provider_model ON model (workspace_id,provider,model_id);

INSERT INTO model VALUES (10000,'1',NULL,'qwen-max','llm','chat','qwen-max','Tongyi',1,'web_search,function_call','preset','2025-08-22 18:20:21','2025-08-22 18:20:21',NULL,NULL,0),(10001,'1',NULL,'qwen-max-latest','llm','chat','qwen-max-latest','Tongyi',1,'web_search,function_call,reasoning','preset','2025-08-22 18:20:21','2025-08-22 18:20:21',NULL,NULL,0),(10002,'1',NULL,'qwen-plus','llm','chat','qwen-plus','Tongyi',1,'web_search,function_call','preset','2025-08-22 18:20:21','2025-08-22 18:20:21',NULL,NULL,0),(10003,'1',NULL,'qwen-plus-latest','llm','chat','qwen-plus-latest','Tongyi',1,'web_search,function_call,reasoning','preset','2025-08-22 18:20:21','2025-08-22 18:20:21',NULL,NULL,0),(10004,'1',NULL,'qwen-turbo','llm','chat','qwen-turbo','Tongyi',1,'web_search,function_call','preset','2025-08-22 18:20:21','2025-08-22 18:20:21',NULL,NULL,0),(10005,'1',NULL,'qwen-turbo-latest','llm','chat','qwen-turbo-latest','Tongyi',1,'web_search,function_call,reasoning','preset','2025-08-22 18:20:21','2025-08-22 18:20:21',NULL,NULL,0),(10006,'1',NULL,'qwen3-235b-a22b','llm','chat','qwen3-235b-a22b','Tongyi',1,'function_call,reasoning','preset','2025-08-22 18:20:21','2025-08-22 18:20:21',NULL,NULL,0),(10007,'1',NULL,'qwen3-30b-a3b','llm','chat','qwen3-30b-a3b','Tongyi',1,'function_call,reasoning','preset','2025-08-22 18:20:21','2025-08-22 18:20:21',NULL,NULL,0),(10008,'1',NULL,'qwen3-32b','llm','chat','qwen3-32b','Tongyi',1,'function_call,reasoning','preset','2025-08-22 18:20:21','2025-08-22 18:20:21',NULL,NULL,0),(10009,'1',NULL,'qwen3-14b','llm','chat','qwen3-14b','Tongyi',1,'function_call,reasoning','preset','2025-08-22 18:20:21','2025-08-22 18:20:21',NULL,NULL,0),(10010,'1',NULL,'qwen3-8b','llm','chat','qwen3-8b','Tongyi',1,'function_call,reasoning','preset','2025-08-22 18:20:21','2025-08-22 18:20:21',NULL,NULL,0),(10011,'1',NULL,'qwen3-4b','llm','chat','qwen3-4b','Tongyi',1,'function_call,reasoning','preset','2025-08-22 18:20:21','2025-08-22 18:20:21',NULL,NULL,0),(10012,'1',NULL,'qwen3-1.7b','llm','chat','qwen3-1.7b','Tongyi',1,'function_call,reasoning','preset','2025-08-22 18:20:21','2025-08-22 18:20:21',NULL,NULL,0),(10013,'1',NULL,'qwen3-0.6b','llm','chat','qwen3-0.6b','Tongyi',1,'function_call,reasoning','preset','2025-08-22 18:20:21','2025-08-22 18:20:21',NULL,NULL,0),(10014,'1',NULL,'qwen-vl-max','llm','chat','qwen-vl-max','Tongyi',1,'vision,function_call','preset','2025-08-22 18:20:21','2025-08-22 18:20:21',NULL,NULL,0),(10015,'1',NULL,'qwen-vl-plus','llm','chat','qwen-vl-plus','Tongyi',1,'vision,function_call','preset','2025-08-22 18:20:21','2025-08-22 18:20:21',NULL,NULL,0),(10016,'1',NULL,'qvq-max','llm','chat','qvq-max','Tongyi',1,'vision,reasoning','preset','2025-08-22 18:20:21','2025-08-22 18:20:21',NULL,NULL,0),(10017,'1',NULL,'qwq-plus','llm','chat','qwq-plus','Tongyi',1,'reasoning,function_call','preset','2025-08-22 18:20:21','2025-08-22 18:20:21',NULL,NULL,0),(10018,'1',NULL,'text-embedding-v1','text_embedding','chat','text-embedding-v1','Tongyi',1,'embedding','preset','2025-08-22 18:20:21','2025-08-22 18:20:21',NULL,NULL,0),(10019,'1',NULL,'text-embedding-v2','text_embedding','chat','text-embedding-v2','Tongyi',1,'embedding','preset','2025-08-22 18:20:22','2025-08-22 18:20:22',NULL,NULL,0),(10020,'1',NULL,'text-embedding-v3','text_embedding','chat','text-embedding-v3','Tongyi',1,'embedding','preset','2025-08-22 18:20:22','2025-08-22 18:20:22',NULL,NULL,0),(10021,'1',NULL,'gte-rerank-v2','rerank','chat','gte-rerank-v2','Tongyi',1,NULL,'preset','2025-08-22 18:20:22','2025-08-22 18:20:22',NULL,NULL,0),(10022,'1',NULL,'deepseek-r1','llm','chat','deepseek-r1','Tongyi',1,'reasoning','preset','2025-08-22 18:20:22','2025-08-22 18:20:22',NULL,NULL,0) ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS plugin (
  id BIGSERIAL NOT NULL,
  plugin_id varchar(64) NOT NULL,
  workspace_id varchar(64) NOT NULL,
  type varchar(64) NOT NULL,
  status smallint NOT NULL DEFAULT 1,
  name varchar(255) NOT NULL,
  description varchar(4096) DEFAULT NULL,
  config text,
  source varchar(64) NOT NULL,
  gmt_create timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  gmt_modified timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  creator varchar(64) NOT NULL,
  modifier varchar(64) NOT NULL,
  tenant_id bigint DEFAULT 0,
  PRIMARY KEY (id),
  CONSTRAINT uk_plugin_id UNIQUE (plugin_id)
);
CREATE INDEX IF NOT EXISTS plugin_idx_workspace_type ON plugin (workspace_id,type);

CREATE TABLE IF NOT EXISTS provider (
  id BIGSERIAL NOT NULL,
  workspace_id varchar(64) DEFAULT NULL,
  icon varchar(255) DEFAULT NULL,
  name varchar(255) DEFAULT NULL,
  description varchar(1024) DEFAULT NULL,
  provider varchar(255) NOT NULL,
  enable smallint DEFAULT 1,
  source varchar(64) NOT NULL DEFAULT 'preset',
  credential varchar(1024) DEFAULT NULL,
  supported_model_types varchar(255) DEFAULT NULL,
  protocol varchar(64) DEFAULT NULL,
  gmt_create timestamp DEFAULT CURRENT_TIMESTAMP,
  gmt_modified timestamp DEFAULT CURRENT_TIMESTAMP,
  creator varchar(64) DEFAULT NULL,
  modifier varchar(64) DEFAULT NULL,
  tenant_id bigint DEFAULT 0,
  PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS provider_idx_account_workspace_provider ON provider (workspace_id,provider);

INSERT INTO provider VALUES (10000,'1',NULL,'Tongyi','Tongyi','Tongyi',1,'preset','{"endpoint":"https://dashscope.aliyuncs.com/compatible-mode/v1","api_key":"gA64aJ7fOMLT/J2DxCxZoHbVgUHeUCyHwpWT/pzH8pyvPDacWMkTiy/hf1lxdSIkkDfeLUbDO9Jeo+Uw0bjEBhSBy6tXWfAEHwD2MXZNd+3FjCSW2w+6WHN7hxG5ObVGyabUZiDZoiTrDN83XUGCaLvc5+qOUtj0mR6pY4KuY9QaDBV/bzBNr8AgHPOZJWxmvNpQcXvZ3yieZorc4g4942ivbNks+bDYobOiZEVSig9fQTd+jWNnqtnI7S5ak29V4tNp9SOsY0v8vmlIJi+9+HpG6z+plM+7KMU0l/WfYiTi0RjZ4DHy8AUM3iJkO/VL7HmKrVUkjzfzqYc9SR552g=="}','llm,text_embedding,rerank','OpenAI','2025-08-22 18:20:21','2025-08-24 17:37:34',NULL,'10000',0) ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS reference (
  id BIGSERIAL NOT NULL,
  gmt_create timestamp NOT NULL,
  gmt_modified timestamp NOT NULL,
  main_code varchar(64) NOT NULL,
  main_type smallint NOT NULL,
  refer_code varchar(64) NOT NULL,
  refer_type smallint NOT NULL,
  workspace_id varchar(64) NOT NULL DEFAULT 1,
  tenant_id bigint DEFAULT 0,
  PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS reference_idx_refer_code ON reference (refer_code);
CREATE INDEX IF NOT EXISTS reference_idx_main_code_workspace_type ON reference (workspace_id,main_code,main_type);

CREATE TABLE IF NOT EXISTS tool (
  id BIGSERIAL NOT NULL,
  plugin_id varchar(64) NOT NULL,
  tool_id varchar(64) NOT NULL,
  workspace_id varchar(64) NOT NULL,
  status smallint NOT NULL DEFAULT 1,
  enabled smallint NOT NULL DEFAULT 1,
  test_status smallint NOT NULL DEFAULT 1,
  name varchar(255) NOT NULL,
  description varchar(4096) DEFAULT NULL,
  config text NOT NULL,
  api_schema text NOT NULL,
  gmt_create timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  gmt_modified timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  creator varchar(64) NOT NULL,
  modifier varchar(64) NOT NULL,
  tenant_id bigint DEFAULT 0,
  PRIMARY KEY (id),
  CONSTRAINT uk_tool_id UNIQUE (tool_id)
);
CREATE INDEX IF NOT EXISTS tool_idx_workspace_plugin ON tool (workspace_id,plugin_id);

CREATE TABLE IF NOT EXISTS workspace (
  id BIGSERIAL NOT NULL,
  workspace_id varchar(64) NOT NULL,
  account_id varchar(64) NOT NULL,
  status smallint NOT NULL DEFAULT 1,
  name varchar(255) NOT NULL,
  description varchar(4096) DEFAULT NULL,
  config text,
  gmt_create timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  gmt_modified timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  creator varchar(64) NOT NULL,
  modifier varchar(64) NOT NULL,
  tenant_id bigint DEFAULT 0,
  PRIMARY KEY (id),
  CONSTRAINT uk_workspace_id UNIQUE (workspace_id)
);
CREATE INDEX IF NOT EXISTS workspace_idx_account_id ON workspace (account_id);

INSERT INTO workspace VALUES (10000,'1','10000',1,'Default Workspace','Default Workspace',NULL,'2025-08-22 18:20:21','2025-08-22 18:20:21','10000','10000',0) ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS dataset (
  id BIGSERIAL NOT NULL,
  name varchar(255) NOT NULL,
  description text DEFAULT NULL,
  columns_config text DEFAULT NULL,
  data_count int NOT NULL DEFAULT 0,
  create_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  update_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  deleted smallint NOT NULL DEFAULT 0,
  PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS dataset_idx_dataset_deleted ON dataset (deleted);
CREATE INDEX IF NOT EXISTS dataset_idx_dataset_update_time ON dataset (update_time);

CREATE TABLE IF NOT EXISTS dataset_version (
  id BIGSERIAL NOT NULL,
  dataset_id bigint NOT NULL,
  version varchar(32) NOT NULL,
  description text DEFAULT NULL,
  data_count int NOT NULL DEFAULT 0,
  status varchar(32) NOT NULL DEFAULT 'draft',
  experiments text DEFAULT NULL,
  dataset_items text DEFAULT NULL,
  columns_config text DEFAULT NULL,
  create_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  update_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  CONSTRAINT uk_dataset_version UNIQUE (dataset_id,version)
);
CREATE INDEX IF NOT EXISTS dataset_version_idx_dataset_version_dataset_id ON dataset_version (dataset_id);
CREATE INDEX IF NOT EXISTS dataset_version_idx_dataset_version_update_time ON dataset_version (update_time);

CREATE TABLE IF NOT EXISTS dataset_item (
  id BIGSERIAL NOT NULL,
  dataset_id bigint NOT NULL,
  columns_config text DEFAULT NULL,
  data_content text DEFAULT NULL,
  remark text DEFAULT NULL,
  create_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  update_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  deleted smallint NOT NULL DEFAULT 0,
  PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS dataset_item_idx_dataset_item_dataset_id ON dataset_item (dataset_id);
CREATE INDEX IF NOT EXISTS dataset_item_idx_dataset_item_deleted ON dataset_item (deleted);
CREATE INDEX IF NOT EXISTS dataset_item_idx_dataset_item_update_time ON dataset_item (update_time);

CREATE TABLE IF NOT EXISTS evaluator (
  id BIGSERIAL NOT NULL,
  name varchar(255) NOT NULL,
  description text DEFAULT NULL,
  latest_version varchar(32) DEFAULT NULL,
  model_name varchar(255) DEFAULT NULL,
  prompt text DEFAULT NULL,
  model_config text DEFAULT NULL,
  variables text DEFAULT NULL,
  create_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  update_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  deleted smallint NOT NULL DEFAULT 0,
  PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS evaluator_idx_evaluator_deleted ON evaluator (deleted);
CREATE INDEX IF NOT EXISTS evaluator_idx_evaluator_update_time ON evaluator (update_time);

CREATE TABLE IF NOT EXISTS evaluator_version (
  id BIGSERIAL NOT NULL,
  evaluator_id bigint NOT NULL,
  description text DEFAULT NULL,
  version varchar(32) NOT NULL,
  model_config text DEFAULT NULL,
  prompt text DEFAULT NULL,
  variables text DEFAULT NULL,
  status varchar(32) DEFAULT NULL,
  experiments text DEFAULT NULL,
  create_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  update_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  CONSTRAINT uk_evaluator_version UNIQUE (evaluator_id,version)
);
CREATE INDEX IF NOT EXISTS evaluator_version_idx_evaluator_version_evaluator_id ON evaluator_version (evaluator_id);
CREATE INDEX IF NOT EXISTS evaluator_version_idx_evaluator_version_update_time ON evaluator_version (update_time);

CREATE TABLE IF NOT EXISTS evaluator_template (
  id BIGSERIAL NOT NULL,
  evaluator_template_key varchar(255) NOT NULL,
  template_desc varchar(255) DEFAULT NULL,
  template text,
  variables text DEFAULT NULL,
  model_config text DEFAULT NULL,
  create_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  update_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  CONSTRAINT uk_evaluator_template_key UNIQUE (evaluator_template_key)
);
CREATE INDEX IF NOT EXISTS evaluator_template_idx_evaluator_template_desc ON evaluator_template (template_desc);

CREATE TABLE IF NOT EXISTS experiment (
  id BIGSERIAL NOT NULL,
  name varchar(255) NOT NULL,
  description text DEFAULT NULL,
  dataset_id bigint DEFAULT NULL,
  dataset_version_id bigint DEFAULT NULL,
  dataset_version varchar(32) DEFAULT NULL,
  evaluation_object_config text DEFAULT NULL,
  evaluator_config text DEFAULT NULL,
  evaluator_id varchar(64) DEFAULT NULL,
  status varchar(32) NOT NULL DEFAULT 'created',
  progress int NOT NULL DEFAULT 0,
  complete_time timestamp DEFAULT NULL,
  create_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  update_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS experiment_idx_experiment_dataset_id ON experiment (dataset_id);
CREATE INDEX IF NOT EXISTS experiment_idx_experiment_dataset_version_id ON experiment (dataset_version_id);
CREATE INDEX IF NOT EXISTS experiment_idx_experiment_status ON experiment (status);
CREATE INDEX IF NOT EXISTS experiment_idx_experiment_update_time ON experiment (update_time);

CREATE TABLE IF NOT EXISTS experiment_result (
  id BIGSERIAL NOT NULL,
  experiment_id bigint NOT NULL,
  input text DEFAULT NULL,
  actual_output text DEFAULT NULL,
  reference_output text DEFAULT NULL,
  score numeric(6,4) DEFAULT NULL,
  reason text DEFAULT NULL,
  evaluation_time timestamp DEFAULT NULL,
  evaluator_version_id bigint DEFAULT NULL,
  create_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  update_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS experiment_result_idx_experiment_result_experiment_id ON experiment_result (experiment_id);
CREATE INDEX IF NOT EXISTS experiment_result_idx_experiment_result_evaluator_version_id ON experiment_result (evaluator_version_id);
CREATE INDEX IF NOT EXISTS experiment_result_idx_experiment_result_update_time ON experiment_result (update_time);

CREATE TABLE IF NOT EXISTS prompt (
  id BIGSERIAL NOT NULL,
  prompt_key varchar(255) NOT NULL,
  prompt_desc varchar(255) DEFAULT NULL,
  prompt_description varchar(255) DEFAULT NULL,
  latest_version varchar(32) DEFAULT NULL,
  latest_version_status varchar(32) DEFAULT NULL,
  tags varchar(255) DEFAULT NULL,
  create_time timestamp(3) DEFAULT CURRENT_TIMESTAMP(3),
  update_time timestamp(3) DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  CONSTRAINT uk_prompt_key UNIQUE (prompt_key)
);
CREATE INDEX IF NOT EXISTS prompt_idx_prompt_update_time ON prompt (update_time);

CREATE TABLE IF NOT EXISTS prompt_version (
  id BIGSERIAL NOT NULL,
  version varchar(32) NOT NULL,
  prompt_key varchar(255) NOT NULL,
  version_desc varchar(255) DEFAULT NULL,
  version_description varchar(255) DEFAULT NULL,
  template text,
  variables text DEFAULT NULL,
  model_config text DEFAULT NULL,
  status varchar(32) NOT NULL DEFAULT 'pre',
  create_time timestamp(3) DEFAULT CURRENT_TIMESTAMP(3),
  update_time timestamp(3) DEFAULT CURRENT_TIMESTAMP(3),
  previous_version varchar(32) DEFAULT NULL,
  PRIMARY KEY (id),
  CONSTRAINT uk_prompt_key_version UNIQUE (prompt_key,version)
);
CREATE INDEX IF NOT EXISTS prompt_version_idx_prompt_version_prompt_key ON prompt_version (prompt_key);
CREATE INDEX IF NOT EXISTS prompt_version_idx_prompt_version_status ON prompt_version (status);
CREATE INDEX IF NOT EXISTS prompt_version_idx_prompt_version_update_time ON prompt_version (update_time);

CREATE TABLE IF NOT EXISTS prompt_build_template (
  id BIGSERIAL NOT NULL,
  prompt_template_key varchar(255) NOT NULL,
  tags varchar(255) DEFAULT NULL,
  template_desc varchar(255) DEFAULT NULL,
  template text,
  variables text DEFAULT NULL,
  model_config text DEFAULT NULL,
  create_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  update_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  CONSTRAINT uk_prompt_template_key UNIQUE (prompt_template_key)
);
CREATE INDEX IF NOT EXISTS prompt_build_template_idx_prompt_build_template_tags ON prompt_build_template (tags);

CREATE TABLE IF NOT EXISTS model_config (
  id BIGSERIAL NOT NULL,
  name varchar(100) NOT NULL,
  provider varchar(50) NOT NULL,
  model_name varchar(100) NOT NULL,
  base_url varchar(500) NOT NULL,
  api_key varchar(500) NOT NULL,
  default_parameters jsonb DEFAULT NULL,
  supported_parameters jsonb DEFAULT NULL,
  status smallint NOT NULL DEFAULT 1,
  create_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  update_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  deleted smallint NOT NULL DEFAULT 0,
  PRIMARY KEY (id),
  CONSTRAINT uk_model_config_name UNIQUE (name)
);
CREATE INDEX IF NOT EXISTS model_config_idx_model_config_provider ON model_config (provider);
CREATE INDEX IF NOT EXISTS model_config_idx_model_config_status ON model_config (status);
CREATE INDEX IF NOT EXISTS model_config_idx_model_config_update_time ON model_config (update_time);

INSERT INTO prompt_build_template (prompt_template_key, tags, template_desc, template, variables, model_config)
VALUES
    ('conversational_ai', 'chat,dialogue', 'Conversational AI template',
     'You are a {{role}}. Traits: {{personality}}. User: {{user_input}}. Reply:',
     'role,personality,user_input',
     '{"model": "qwen-max", "temperature": 0.7, "max_tokens": 2000}'),
    ('task_executor', 'task,execution', 'Task execution template',
     'As a {{domain}} expert, complete this task: {{task_description}}. Input: {{input_data}}. Requirements: {{output_requirements}}.',
     'domain,task_description,input_data,output_requirements',
     '{"model": "qwen-max", "temperature": 0.3, "max_tokens": 3000}') ON CONFLICT DO NOTHING;

INSERT INTO evaluator_template (evaluator_template_key, template_desc, template, variables, model_config)
VALUES
    ('text_similarity', 'Text similarity evaluator',
     'Evaluate similarity between reference output {{reference_output}} and actual output {{actual_output}}. Return a score from 0 to 1.',
     'reference_output,actual_output',
     '{"modelId": "qwen-max", "temperature": 0.1, "max_tokens": 100}'),
    ('code_quality', 'Code quality evaluator',
     'Evaluate the following code for readability, efficiency, and best practices: {{code}}.',
     'code',
     '{"modelId": "qwen-max", "temperature": 0.2, "max_tokens": 1000}') ON CONFLICT DO NOTHING;


