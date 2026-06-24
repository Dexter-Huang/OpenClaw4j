/******************************************/
/*   SQLite Database Initialization       */
/*   Converted from H2/MySQL Version      */
/******************************************/

/******************************************/
/*   table = account                      */
/******************************************/
CREATE TABLE IF NOT EXISTS account
(
    id             INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    account_id     TEXT    NOT NULL,
    username       TEXT    NOT NULL,
    email          TEXT    DEFAULT NULL,
    mobile         TEXT    DEFAULT NULL,
    password       TEXT    NOT NULL,
    nickname       TEXT    DEFAULT NULL,
    icon           TEXT    DEFAULT NULL,
    type           TEXT    NOT NULL,
    status         INTEGER NOT NULL DEFAULT 1,
    gmt_create     TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    gmt_modified   TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    gmt_last_login TEXT    DEFAULT NULL,
    creator        TEXT    NOT NULL,
    modifier       TEXT    NOT NULL,
    tenant_id      INTEGER DEFAULT 0
);
CREATE UNIQUE INDEX IF NOT EXISTS uk_account_id       ON account (account_id);
CREATE        INDEX IF NOT EXISTS idx_account_username ON account (username);

/******************************************/
/*   table = agent_schema                 */
/******************************************/
CREATE TABLE IF NOT EXISTS agent_schema
(
    id           INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    agent_id     TEXT    DEFAULT NULL,
    workspace_id TEXT    NOT NULL,
    name         TEXT    NOT NULL,
    description  TEXT    DEFAULT NULL,
    type         TEXT    NOT NULL,
    instruction  TEXT    DEFAULT NULL,
    input_keys   TEXT    DEFAULT NULL,
    output_key   TEXT    DEFAULT NULL,
    handle       TEXT    DEFAULT NULL,
    sub_agents   TEXT    DEFAULT NULL,
    yaml_schema  TEXT    DEFAULT NULL,
    status       TEXT    NOT NULL DEFAULT 'DRAFT',
    enabled      INTEGER NOT NULL DEFAULT 1,
    gmt_create   TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    gmt_modified TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creator      TEXT    NOT NULL,
    modifier     TEXT    NOT NULL,
    tenant_id    INTEGER DEFAULT 0
);
CREATE UNIQUE INDEX IF NOT EXISTS uk_agent_id                ON agent_schema (agent_id);
CREATE        INDEX IF NOT EXISTS idx_agent_workspace_type   ON agent_schema (workspace_id, type);
CREATE        INDEX IF NOT EXISTS idx_agent_workspace_status ON agent_schema (workspace_id, status);

/******************************************/
/*   table = api_key                      */
/******************************************/
CREATE TABLE IF NOT EXISTS api_key
(
    id           INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    account_id   TEXT    NOT NULL,
    api_key      TEXT    NOT NULL,
    status       INTEGER NOT NULL DEFAULT 1,
    description  TEXT    DEFAULT NULL,
    gmt_create   TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    gmt_modified TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creator      TEXT    NOT NULL,
    modifier     TEXT    NOT NULL,
    tenant_id    INTEGER DEFAULT 0
);
CREATE UNIQUE INDEX IF NOT EXISTS uk_api_key             ON api_key (api_key);
CREATE        INDEX IF NOT EXISTS idx_api_key_account_id ON api_key (account_id);

/******************************************/
/*   table = application                  */
/******************************************/
CREATE TABLE IF NOT EXISTS application
(
    id           INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    workspace_id TEXT    NOT NULL,
    app_id       TEXT    NOT NULL,
    name         TEXT    NOT NULL,
    description  TEXT    DEFAULT NULL,
    icon         TEXT    DEFAULT NULL,
    source       TEXT    NOT NULL,
    type         TEXT    NOT NULL,
    status       INTEGER NOT NULL DEFAULT 1,
    gmt_create   TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    gmt_modified TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creator      TEXT    NOT NULL,
    modifier     TEXT    NOT NULL,
    tenant_id    INTEGER DEFAULT 0
);
CREATE UNIQUE INDEX IF NOT EXISTS uk_app_id              ON application (app_id);
CREATE        INDEX IF NOT EXISTS idx_app_workspace_type ON application (workspace_id, type);

/******************************************/
/*   table = application_component        */
/******************************************/
CREATE TABLE IF NOT EXISTS application_component
(
    id           INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    gmt_create   TEXT    NOT NULL,
    gmt_modified TEXT    NOT NULL,
    code         TEXT    NOT NULL,
    name         TEXT    NOT NULL,
    workspace_id TEXT    NOT NULL,
    type         TEXT    NOT NULL,
    app_id       TEXT    DEFAULT NULL,
    config       TEXT    DEFAULT NULL,
    description  TEXT    DEFAULT NULL,
    status       INTEGER DEFAULT NULL,
    creator      TEXT    DEFAULT NULL,
    modifier     TEXT    DEFAULT NULL,
    need_update  INTEGER DEFAULT NULL,
    tenant_id    INTEGER DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_workspace_type_status_appcode ON application_component (workspace_id, type, app_id);

/******************************************/
/*   table = application_version          */
/******************************************/
CREATE TABLE IF NOT EXISTS application_version
(
    id           INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    app_id       TEXT    NOT NULL,
    workspace_id TEXT    NOT NULL,
    config       TEXT    DEFAULT NULL,
    status       INTEGER NOT NULL,
    version      TEXT    NOT NULL DEFAULT '0.0.1',
    description  TEXT    DEFAULT NULL,
    gmt_create   TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    gmt_modified TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creator      TEXT    NOT NULL,
    modifier     TEXT    NOT NULL,
    tenant_id    INTEGER DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_workspace_app_version ON application_version (workspace_id, app_id, version);

/******************************************/
/*   table = document                     */
/******************************************/
CREATE TABLE IF NOT EXISTS document
(
    id             INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    workspace_id   TEXT    NOT NULL,
    kb_id          TEXT    NOT NULL,
    doc_id         TEXT    NOT NULL,
    type           TEXT    NOT NULL,
    status         INTEGER NOT NULL DEFAULT 1,
    enabled        INTEGER NOT NULL DEFAULT 1,
    name           TEXT    NOT NULL,
    format         TEXT    NOT NULL,
    size           INTEGER NOT NULL DEFAULT 0,
    metadata       TEXT    DEFAULT NULL,
    index_status   INTEGER NOT NULL DEFAULT 1,
    path           TEXT    NOT NULL,
    parsed_path    TEXT    DEFAULT NULL,
    process_config TEXT    DEFAULT NULL,
    source         TEXT    DEFAULT NULL,
    error          TEXT    DEFAULT NULL,
    gmt_create     TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    gmt_modified   TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creator        TEXT    NOT NULL,
    modifier       TEXT    NOT NULL,
    tenant_id      INTEGER DEFAULT 0
);
CREATE UNIQUE INDEX IF NOT EXISTS uk_document_id           ON document (doc_id);
CREATE        INDEX IF NOT EXISTS idx_document_workspace_kb ON document (workspace_id, kb_id);

/******************************************/
/*   table = knowledge_base               */
/******************************************/
CREATE TABLE IF NOT EXISTS knowledge_base
(
    id             INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    workspace_id   TEXT    NOT NULL,
    kb_id          TEXT    NOT NULL,
    type           TEXT    NOT NULL,
    status         INTEGER NOT NULL DEFAULT 1,
    name           TEXT    NOT NULL,
    description    TEXT    DEFAULT NULL,
    process_config TEXT    DEFAULT NULL,
    index_config   TEXT    DEFAULT NULL,
    search_config  TEXT    DEFAULT NULL,
    total_docs     INTEGER NOT NULL DEFAULT 0,
    gmt_create     TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    gmt_modified   TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creator        TEXT    NOT NULL,
    modifier       TEXT    NOT NULL,
    tenant_id      INTEGER DEFAULT 0
);
CREATE UNIQUE INDEX IF NOT EXISTS uk_kb_id                    ON knowledge_base (kb_id);
CREATE        INDEX IF NOT EXISTS idx_kb_workspace_status_name ON knowledge_base (workspace_id);

/******************************************/
/*   table = mcp_server                   */
/******************************************/
CREATE TABLE IF NOT EXISTS mcp_server
(
    id            INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    gmt_create    TEXT    NOT NULL,
    gmt_modified  TEXT    NOT NULL,
    server_code   TEXT    NOT NULL,
    name          TEXT    NOT NULL,
    description   TEXT    DEFAULT NULL,
    source        TEXT    DEFAULT NULL,
    deploy_env    TEXT    DEFAULT NULL,
    type          TEXT    NOT NULL,
    deploy_config TEXT    NOT NULL,
    workspace_id  TEXT    DEFAULT NULL,
    account_id    TEXT    DEFAULT NULL,
    status        INTEGER NOT NULL,
    biz_type      TEXT    DEFAULT NULL,
    detail_config TEXT    DEFAULT NULL,
    host          TEXT    DEFAULT NULL,
    install_type  TEXT    DEFAULT NULL,
    tenant_id     INTEGER DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_mcp_code        ON mcp_server (server_code);
CREATE INDEX IF NOT EXISTS idx_mcp_server_name ON mcp_server (name);
CREATE INDEX IF NOT EXISTS idx_mcp_name_status ON mcp_server (status, name);
CREATE INDEX IF NOT EXISTS idx_mcp_type        ON mcp_server (type);

/******************************************/
/*   table = model                        */
/******************************************/
CREATE TABLE IF NOT EXISTS model
(
    id           INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    workspace_id TEXT    DEFAULT NULL,
    icon         TEXT    DEFAULT NULL,
    name         TEXT    DEFAULT NULL,
    type         TEXT    DEFAULT 'LLM',
    mode         TEXT    DEFAULT 'chat',
    model_id     TEXT    NOT NULL,
    provider     TEXT    NOT NULL,
    enable       INTEGER DEFAULT 1,
    tags         TEXT    DEFAULT NULL,
    source       TEXT    NOT NULL DEFAULT 'preset',
    gmt_create   TEXT    DEFAULT CURRENT_TIMESTAMP,
    gmt_modified TEXT    DEFAULT CURRENT_TIMESTAMP,
    creator      TEXT    DEFAULT NULL,
    modifier     TEXT    DEFAULT NULL,
    tenant_id    INTEGER DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_account_workspace_provider_model ON model (workspace_id, provider, model_id);

/******************************************/
/*   table = plugin                       */
/******************************************/
CREATE TABLE IF NOT EXISTS plugin
(
    id           INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    plugin_id    TEXT    NOT NULL,
    workspace_id TEXT    NOT NULL,
    type         TEXT    NOT NULL,
    status       INTEGER NOT NULL DEFAULT 1,
    name         TEXT    NOT NULL,
    description  TEXT    DEFAULT NULL,
    config       TEXT    DEFAULT NULL,
    source       TEXT    NOT NULL,
    gmt_create   TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    gmt_modified TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creator      TEXT    NOT NULL,
    modifier     TEXT    NOT NULL,
    tenant_id    INTEGER DEFAULT 0
);
CREATE UNIQUE INDEX IF NOT EXISTS uk_plugin_id              ON plugin (plugin_id);
CREATE        INDEX IF NOT EXISTS idx_plugin_workspace_type ON plugin (workspace_id, type);

/******************************************/
/*   table = provider                     */
/******************************************/
CREATE TABLE IF NOT EXISTS provider
(
    id                    INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    workspace_id          TEXT    DEFAULT NULL,
    icon                  TEXT    DEFAULT NULL,
    name                  TEXT    DEFAULT NULL,
    description           TEXT    DEFAULT NULL,
    provider              TEXT    NOT NULL,
    enable                INTEGER DEFAULT 1,
    source                TEXT    NOT NULL DEFAULT 'preset',
    credential            TEXT    DEFAULT NULL,
    supported_model_types TEXT    DEFAULT NULL,
    protocol              TEXT    DEFAULT NULL,
    gmt_create            TEXT    DEFAULT CURRENT_TIMESTAMP,
    gmt_modified          TEXT    DEFAULT CURRENT_TIMESTAMP,
    creator               TEXT    DEFAULT NULL,
    modifier              TEXT    DEFAULT NULL,
    tenant_id             INTEGER DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_provider_workspace ON provider (workspace_id, provider);

/******************************************/
/*   table = reference                    */
/******************************************/
CREATE TABLE IF NOT EXISTS reference
(
    id           INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    gmt_create   TEXT    NOT NULL,
    gmt_modified TEXT    NOT NULL,
    main_code    TEXT    NOT NULL,
    main_type    INTEGER NOT NULL,
    refer_code   TEXT    NOT NULL,
    refer_type   INTEGER NOT NULL,
    workspace_id TEXT    NOT NULL DEFAULT '1',
    tenant_id    INTEGER DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_refer_code               ON reference (refer_code);
CREATE INDEX IF NOT EXISTS idx_main_code_workspace_type ON reference (workspace_id, main_code, main_type);

/******************************************/
/*   table = tool                         */
/******************************************/
CREATE TABLE IF NOT EXISTS tool
(
    id           INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    plugin_id    TEXT    NOT NULL,
    tool_id      TEXT    NOT NULL,
    workspace_id TEXT    NOT NULL,
    status       INTEGER NOT NULL DEFAULT 1,
    enabled      INTEGER NOT NULL DEFAULT 1,
    test_status  INTEGER NOT NULL DEFAULT 1,
    name         TEXT    NOT NULL,
    description  TEXT    DEFAULT NULL,
    config       TEXT    NOT NULL,
    api_schema   TEXT    NOT NULL,
    gmt_create   TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    gmt_modified TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creator      TEXT    NOT NULL,
    modifier     TEXT    NOT NULL,
    tenant_id    INTEGER DEFAULT 0
);
CREATE UNIQUE INDEX IF NOT EXISTS uk_tool_id               ON tool (tool_id);
CREATE        INDEX IF NOT EXISTS idx_tool_workspace_plugin ON tool (workspace_id, plugin_id);

/******************************************/
/*   table = workspace                    */
/******************************************/
CREATE TABLE IF NOT EXISTS workspace
(
    id           INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    workspace_id TEXT    NOT NULL,
    account_id   TEXT    NOT NULL,
    status       INTEGER NOT NULL DEFAULT 1,
    name         TEXT    NOT NULL,
    description  TEXT    DEFAULT NULL,
    config       TEXT    DEFAULT NULL,
    gmt_create   TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    gmt_modified TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creator      TEXT    NOT NULL,
    modifier     TEXT    NOT NULL,
    tenant_id    INTEGER DEFAULT 0
);
CREATE UNIQUE INDEX IF NOT EXISTS uk_workspace_id         ON workspace (workspace_id);
CREATE        INDEX IF NOT EXISTS idx_workspace_account_id ON workspace (account_id);

/******************************************/
/*   admin compat: prompt and evaluation  */
/******************************************/
CREATE TABLE IF NOT EXISTS prompt
(
    id                    INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    prompt_key            TEXT    NOT NULL,
    prompt_desc           TEXT    DEFAULT NULL,
    prompt_description    TEXT    DEFAULT NULL,
    latest_version        TEXT    DEFAULT '1.0.0',
    tags                  TEXT    DEFAULT NULL,
    latest_version_status TEXT    DEFAULT 'pre',
    create_time           TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time           TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS uk_prompt_key ON prompt (prompt_key);

CREATE TABLE IF NOT EXISTS prompt_version
(
    id                  INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    prompt_key          TEXT    NOT NULL,
    version             TEXT    NOT NULL,
    version_description TEXT    DEFAULT NULL,
    template            TEXT    DEFAULT NULL,
    variables           TEXT    DEFAULT NULL,
    model_config        TEXT    DEFAULT NULL,
    previous_version    TEXT    DEFAULT NULL,
    status              TEXT    DEFAULT 'pre',
    create_time         TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time         TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS uk_prompt_key_version ON prompt_version (prompt_key, version);

CREATE TABLE IF NOT EXISTS prompt_build_template
(
    id                  INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    prompt_template_key TEXT    NOT NULL,
    template_desc       TEXT    DEFAULT NULL,
    template            TEXT    DEFAULT NULL,
    variables           TEXT    DEFAULT NULL,
    model_config        TEXT    DEFAULT NULL,
    tags                TEXT    DEFAULT NULL,
    create_time         TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time         TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS uk_prompt_template_key ON prompt_build_template (prompt_template_key);

CREATE TABLE IF NOT EXISTS dataset
(
    id             INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    name           TEXT    NOT NULL,
    description    TEXT    DEFAULT NULL,
    columns_config TEXT    DEFAULT '[]',
    data_count     INTEGER DEFAULT 0,
    create_time    TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time    TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS dataset_version
(
    id             INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    dataset_id     INTEGER NOT NULL,
    version        TEXT    NOT NULL DEFAULT '1.0.0',
    description    TEXT    DEFAULT NULL,
    data_count     INTEGER DEFAULT 0,
    status         TEXT    DEFAULT 'draft',
    experiments    TEXT    DEFAULT '[]',
    dataset_items  TEXT    DEFAULT '[]',
    columns_config TEXT    DEFAULT '[]',
    create_time    TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time    TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS uk_dataset_version ON dataset_version (dataset_id, version);

CREATE TABLE IF NOT EXISTS dataset_item
(
    id           INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    dataset_id   INTEGER NOT NULL,
    data_content TEXT    DEFAULT NULL,
    remark       TEXT    DEFAULT NULL,
    create_time  TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time  TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS evaluator
(
    id             INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    name           TEXT    NOT NULL,
    description    TEXT    DEFAULT NULL,
    latest_version TEXT    DEFAULT NULL,
    model_name     TEXT    DEFAULT NULL,
    prompt         TEXT    DEFAULT NULL,
    model_config   TEXT    DEFAULT NULL,
    variables      TEXT    DEFAULT NULL,
    create_time    TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time    TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS evaluator_version
(
    id            INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    evaluator_id  INTEGER NOT NULL,
    description   TEXT    DEFAULT NULL,
    version       TEXT    NOT NULL DEFAULT '1.0.0',
    model_config  TEXT    DEFAULT NULL,
    prompt        TEXT    DEFAULT NULL,
    variables     TEXT    DEFAULT NULL,
    status        TEXT    DEFAULT 'draft',
    experiments   TEXT    DEFAULT '[]',
    create_time   TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time   TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS uk_evaluator_version ON evaluator_version (evaluator_id, version);

CREATE TABLE IF NOT EXISTS evaluator_template
(
    id                     INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    evaluator_template_key TEXT    NOT NULL,
    template_desc          TEXT    DEFAULT NULL,
    template               TEXT    DEFAULT NULL,
    variables              TEXT    DEFAULT NULL,
    model_config           TEXT    DEFAULT NULL,
    create_time            TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time            TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS uk_evaluator_template_key ON evaluator_template (evaluator_template_key);

CREATE TABLE IF NOT EXISTS experiment
(
    id                       INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    name                     TEXT    NOT NULL,
    description              TEXT    DEFAULT NULL,
    dataset_id               INTEGER DEFAULT NULL,
    dataset_version_id       INTEGER DEFAULT NULL,
    dataset_version          TEXT    DEFAULT NULL,
    evaluator_id             TEXT    DEFAULT NULL,
    evaluation_object_config TEXT    DEFAULT NULL,
    evaluator_config         TEXT    DEFAULT NULL,
    status                   TEXT    DEFAULT 'created',
    progress                 INTEGER DEFAULT 0,
    complete_time            TEXT    DEFAULT NULL,
    create_time              TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time              TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS experiment_result
(
    id                   INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    experiment_id        INTEGER NOT NULL,
    evaluator_version_id INTEGER DEFAULT NULL,
    input                TEXT    DEFAULT NULL,
    actual_output        TEXT    DEFAULT NULL,
    reference_output     TEXT    DEFAULT NULL,
    score                REAL    DEFAULT NULL,
    reason               TEXT    DEFAULT NULL,
    create_time          TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time          TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

/******************************************/
/*   Initial Data                         */
/******************************************/

-- init account
INSERT OR IGNORE INTO account (account_id, username, email, mobile, password, type, status, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('10000', 'saa', 'ken.lj.hz@gmail.com', NULL,
        '$argon2id$v=19$m=66536,t=2,p=1$KSDQowfZxDjKLqBtxFNRng$znU0oQFQs2shR9la4S11n7d0LpGApmSBXvDOXuhbR40',
        'admin', 1, datetime('now'), datetime('now'), '10000', '10000', 0);

-- init api_key
INSERT OR IGNORE INTO api_key (account_id, api_key, status, description, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('10000', '+Ae6iTvRFwv7auIV/RN5vxanWB07uxn3CH9Za7EPTMA9Mq4eNRK8K0sprMrUEaYM', 1, '11',
        datetime('now'), datetime('now'), '10000', '10000', 0);

-- init workspace
INSERT OR IGNORE INTO workspace (workspace_id, account_id, status, name, description, config, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', '10000', 1, 'Default Workspace', 'Default Workspace', NULL, datetime('now'), datetime('now'), '10000', '10000', 0);

-- init provider
INSERT OR IGNORE INTO provider (workspace_id, icon, name, description, provider, enable, source, credential, supported_model_types, protocol, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'Tongyi', 'Tongyi', 'Tongyi', 1, 'preset',
        '{"endpoint":"https://dashscope.aliyuncs.com/compatible-mode","api_key":"gA64aJ7fOMLT/J2DxCxZoHbVgUHeUCyHwpWT/pzH8pyvPDacWMkTiy/hf1lxdSIkkDfeLUbDO9Jeo+Uw0bjEBhSBy6tXWfAEHwD2MXZNd+3FjCSW2w+6WHN7hxG5ObVGyabUZiDZoiTrDN83XUGCaLvc5+qOUtj0mR6pY4KuY9QaDBV/bzBNr8AgHPOZJWxmvNpQcXvZ3yieZorc4g4942ivbNks+bDYobOiZEVSig9fQTd+jWNnqtnI7S5ak29V4tNp9SOsY0v8vmlIJi+9+HpG6z+plM+7KMU0l/WfYiTi0RjZ4DHy8AUM3iJkO/VL7HmKrVUkjzfzqYc9SR552g=="}',
        NULL, 'OpenAI', datetime('now'), datetime('now'), NULL, NULL, 0);

-- init model
INSERT OR IGNORE INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'qwen-max', 'llm', 'chat', 'qwen-max', 'Tongyi', 1, 'web_search,function_call', 'preset', datetime('now'), datetime('now'), NULL, NULL, 0);

INSERT OR IGNORE INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'qwen-max-latest', 'llm', 'chat', 'qwen-max-latest', 'Tongyi', 1, 'web_search,function_call,reasoning', 'preset', datetime('now'), datetime('now'), NULL, NULL, 0);

INSERT OR IGNORE INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'qwen-plus', 'llm', 'chat', 'qwen-plus', 'Tongyi', 1, 'web_search,function_call', 'preset', datetime('now'), datetime('now'), NULL, NULL, 0);

INSERT OR IGNORE INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'qwen-plus-latest', 'llm', 'chat', 'qwen-plus-latest', 'Tongyi', 1, 'web_search,function_call,reasoning', 'preset', datetime('now'), datetime('now'), NULL, NULL, 0);

INSERT OR IGNORE INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'qwen-turbo', 'llm', 'chat', 'qwen-turbo', 'Tongyi', 1, 'web_search,function_call', 'preset', datetime('now'), datetime('now'), NULL, NULL, 0);

INSERT OR IGNORE INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'qwen-turbo-latest', 'llm', 'chat', 'qwen-turbo-latest', 'Tongyi', 1, 'web_search,function_call,reasoning', 'preset', datetime('now'), datetime('now'), NULL, NULL, 0);

INSERT OR IGNORE INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'qwen3-235b-a22b', 'llm', 'chat', 'qwen3-235b-a22b', 'Tongyi', 1, 'function_call,reasoning', 'preset', datetime('now'), datetime('now'), NULL, NULL, 0);

INSERT OR IGNORE INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'qwen3-30b-a3b', 'llm', 'chat', 'qwen3-30b-a3b', 'Tongyi', 1, 'function_call,reasoning', 'preset', datetime('now'), datetime('now'), NULL, NULL, 0);

INSERT OR IGNORE INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'qwen3-32b', 'llm', 'chat', 'qwen3-32b', 'Tongyi', 1, 'function_call,reasoning', 'preset', datetime('now'), datetime('now'), NULL, NULL, 0);

INSERT OR IGNORE INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'qwen3-14b', 'llm', 'chat', 'qwen3-14b', 'Tongyi', 1, 'function_call,reasoning', 'preset', datetime('now'), datetime('now'), NULL, NULL, 0);

INSERT OR IGNORE INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'qwen3-8b', 'llm', 'chat', 'qwen3-8b', 'Tongyi', 1, 'function_call,reasoning', 'preset', datetime('now'), datetime('now'), NULL, NULL, 0);

INSERT OR IGNORE INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'qwen3-4b', 'llm', 'chat', 'qwen3-4b', 'Tongyi', 1, 'function_call,reasoning', 'preset', datetime('now'), datetime('now'), NULL, NULL, 0);

INSERT OR IGNORE INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'qwen3-1.7b', 'llm', 'chat', 'qwen3-1.7b', 'Tongyi', 1, 'function_call,reasoning', 'preset', datetime('now'), datetime('now'), NULL, NULL, 0);

INSERT OR IGNORE INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'qwen3-0.6b', 'llm', 'chat', 'qwen3-0.6b', 'Tongyi', 1, 'function_call,reasoning', 'preset', datetime('now'), datetime('now'), NULL, NULL, 0);

INSERT OR IGNORE INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'qwen-vl-max', 'llm', 'chat', 'qwen-vl-max', 'Tongyi', 1, 'vision,function_call', 'preset', datetime('now'), datetime('now'), NULL, NULL, 0);

INSERT OR IGNORE INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'qwen-vl-plus', 'llm', 'chat', 'qwen-vl-plus', 'Tongyi', 1, 'vision,function_call', 'preset', datetime('now'), datetime('now'), NULL, NULL, 0);

INSERT OR IGNORE INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'qvq-max', 'llm', 'chat', 'qvq-max', 'Tongyi', 1, 'vision,reasoning', 'preset', datetime('now'), datetime('now'), NULL, NULL, 0);

INSERT OR IGNORE INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'qwq-plus', 'llm', 'chat', 'qwq-plus', 'Tongyi', 1, 'reasoning,function_call', 'preset', datetime('now'), datetime('now'), NULL, NULL, 0);

INSERT OR IGNORE INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'text-embedding-v1', 'text_embedding', 'chat', 'text-embedding-v1', 'Tongyi', 1, 'embedding', 'preset', datetime('now'), datetime('now'), NULL, NULL, 0);

INSERT OR IGNORE INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'text-embedding-v2', 'text_embedding', 'chat', 'text-embedding-v2', 'Tongyi', 1, 'embedding', 'preset', datetime('now'), datetime('now'), NULL, NULL, 0);

INSERT OR IGNORE INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'text-embedding-v3', 'text_embedding', 'chat', 'text-embedding-v3', 'Tongyi', 1, 'embedding', 'preset', datetime('now'), datetime('now'), NULL, NULL, 0);

INSERT OR IGNORE INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'gte-rerank-v2', 'rerank', 'chat', 'gte-rerank-v2', 'Tongyi', 1, NULL, 'preset', datetime('now'), datetime('now'), NULL, NULL, 0);

INSERT OR IGNORE INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'deepseek-r1', 'llm', 'chat', 'deepseek-r1', 'Tongyi', 1, 'reasoning', 'preset', datetime('now'), datetime('now'), NULL, NULL, 0);

-- init admin compat templates
INSERT OR IGNORE INTO prompt_build_template
    (prompt_template_key, template_desc, template, variables, model_config, tags, create_time, update_time)
VALUES
    ('通用助手', '通用问答 Prompt 模板', '你是一个可靠的助手，请回答用户问题：{{query}}',
     '[{"name":"query","type":"String","required":true}]',
     '{"model":"qwen-max","temperature":0.7}', 'assistant,general', datetime('now'), datetime('now'));

INSERT OR IGNORE INTO evaluator_template
    (evaluator_template_key, template_desc, template, variables, model_config, create_time, update_time)
VALUES
    ('准确性评估', '根据参考答案评估模型回答准确性',
     '请根据参考答案评价实际回答，输出 0 到 1 的分数和原因。\n参考答案：{{reference}}\n实际回答：{{actual}}',
     '[{"name":"reference","type":"String","required":true},{"name":"actual","type":"String","required":true}]',
     '{"model":"qwen-max","temperature":0}', datetime('now'), datetime('now'));
