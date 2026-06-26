/******************************************/
/*   CBES LLM Database Initialization     */
/*   H2 MySQL Mode Compatible Version     */
/******************************************/

/******************************************/
/*   table = account                      */
/******************************************/
DROP TABLE IF EXISTS account;
CREATE TABLE account
(
    id             BIGINT AUTO_INCREMENT NOT NULL COMMENT 'pk',
    account_id     VARCHAR(64)           NOT NULL COMMENT 'account id',
    username       VARCHAR(255)          NOT NULL COMMENT 'account name',
    email          VARCHAR(255)          DEFAULT NULL COMMENT 'account email',
    mobile         VARCHAR(255)          DEFAULT NULL COMMENT 'account mobile',
    password       VARCHAR(255)          NOT NULL COMMENT 'password',
    nickname       VARCHAR(255)          DEFAULT NULL COMMENT 'nickname',
    icon           VARCHAR(255)          DEFAULT NULL COMMENT 'account icon',
    type           VARCHAR(64)           NOT NULL COMMENT 'type: basic, admin',
    status         TINYINT               NOT NULL DEFAULT 1 COMMENT 'status: 0- deleted, 1- normal',
    gmt_create     TIMESTAMP             NOT NULL DEFAULT CURRENT_TIMESTAMP,
    gmt_modified   TIMESTAMP             NOT NULL DEFAULT CURRENT_TIMESTAMP,
    gmt_last_login TIMESTAMP             DEFAULT NULL COMMENT 'account last login time',
    creator        VARCHAR(64)           NOT NULL COMMENT 'creator uid',
    modifier       VARCHAR(64)           NOT NULL COMMENT 'modifier uid',
    tenant_id      BIGINT                DEFAULT 0,
    PRIMARY KEY (id),
    UNIQUE KEY uk_account_id (account_id),
    KEY idx_email_password (username)
);

/******************************************/
/*   table = agent_schema                 */
/******************************************/
DROP TABLE IF EXISTS agent_schema;
CREATE TABLE agent_schema
(
    id           BIGINT AUTO_INCREMENT NOT NULL COMMENT 'pk',
    agent_id     VARCHAR(64)           DEFAULT NULL COMMENT 'agent id',
    workspace_id VARCHAR(64)           NOT NULL COMMENT 'workspace id',
    name         VARCHAR(255)          NOT NULL COMMENT 'agent name',
    description  VARCHAR(4096)         DEFAULT NULL COMMENT 'agent description',
    type         VARCHAR(64)           NOT NULL COMMENT 'agent type',
    instruction  TEXT                  DEFAULT NULL COMMENT 'system instruction',
    input_keys   TEXT                  DEFAULT NULL COMMENT 'input keys JSON',
    output_key   VARCHAR(255)          DEFAULT NULL COMMENT 'output key',
    handle       LONGTEXT              DEFAULT NULL COMMENT 'handle configuration JSON',
    sub_agents   LONGTEXT              DEFAULT NULL COMMENT 'sub agents configuration JSON',
    yaml_schema  LONGTEXT              DEFAULT NULL COMMENT 'generated YAML schema',
    status       VARCHAR(64)           NOT NULL DEFAULT 'DRAFT' COMMENT 'agent status',
    enabled      TINYINT               NOT NULL DEFAULT 1 COMMENT 'enabled: 0-disabled, 1-enabled',
    gmt_create   TIMESTAMP             NOT NULL DEFAULT CURRENT_TIMESTAMP,
    gmt_modified TIMESTAMP             NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creator      VARCHAR(64)           NOT NULL COMMENT 'creator uid',
    modifier     VARCHAR(64)           NOT NULL COMMENT 'modifier uid',
    tenant_id    BIGINT                DEFAULT 0,
    PRIMARY KEY (id),
    UNIQUE KEY uk_agent_id (agent_id),
    KEY idx_workspace_type_agent_schema (workspace_id, type),
    KEY idx_workspace_status (workspace_id, status)
);

/******************************************/
/*   table = api_key                      */
/******************************************/
DROP TABLE IF EXISTS api_key;
CREATE TABLE api_key
(
    id           BIGINT AUTO_INCREMENT NOT NULL COMMENT 'pk',
    account_id   VARCHAR(64)           NOT NULL COMMENT 'uid',
    api_key      VARCHAR(512)          NOT NULL COMMENT 'api key',
    status       TINYINT               NOT NULL DEFAULT 1 COMMENT 'status: 0-deleted, 1-normal',
    description  VARCHAR(4096)         DEFAULT NULL COMMENT 'api key description',
    gmt_create   TIMESTAMP             NOT NULL DEFAULT CURRENT_TIMESTAMP,
    gmt_modified TIMESTAMP             NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creator      VARCHAR(64)           NOT NULL COMMENT 'creator uid',
    modifier     VARCHAR(64)           NOT NULL COMMENT 'creator uid',
    tenant_id    BIGINT                DEFAULT 0,
    PRIMARY KEY (id),
    UNIQUE KEY uk_api_key (api_key),
    KEY idx_account_id_api_key (account_id)
);

/******************************************/
/*   table = application                  */
/******************************************/
DROP TABLE IF EXISTS application;
CREATE TABLE application
(
    id           BIGINT AUTO_INCREMENT NOT NULL COMMENT 'pk',
    workspace_id VARCHAR(64)           NOT NULL COMMENT 'workspace id',
    app_id       VARCHAR(64)           NOT NULL COMMENT 'app id',
    name         VARCHAR(255)          NOT NULL COMMENT 'app name',
    description  VARCHAR(4096)         DEFAULT NULL COMMENT 'app description',
    icon         VARCHAR(255)          DEFAULT NULL COMMENT 'app icon',
    source       VARCHAR(64)           NOT NULL COMMENT 'app source',
    type         VARCHAR(64)           NOT NULL COMMENT 'type, agent, workflow',
    status       TINYINT               NOT NULL DEFAULT 1 COMMENT 'status',
    gmt_create   TIMESTAMP             NOT NULL DEFAULT CURRENT_TIMESTAMP,
    gmt_modified TIMESTAMP             NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creator      VARCHAR(64)           NOT NULL COMMENT 'creator uid',
    modifier     VARCHAR(64)           NOT NULL COMMENT 'modifier uid',
    tenant_id    BIGINT                DEFAULT 0,
    PRIMARY KEY (id),
    UNIQUE KEY uk_app_id (app_id),
    KEY idx_workspace_type_application (workspace_id, type)
);

/******************************************/
/*   table = application_component        */
/******************************************/
DROP TABLE IF EXISTS application_component;
CREATE TABLE application_component
(
    id           BIGINT AUTO_INCREMENT NOT NULL COMMENT 'pk',
    gmt_create   TIMESTAMP             NOT NULL COMMENT 'create time',
    gmt_modified TIMESTAMP             NOT NULL COMMENT 'modified time',
    code         VARCHAR(64)           NOT NULL COMMENT 'component code',
    name         VARCHAR(128)          NOT NULL COMMENT 'name',
    workspace_id VARCHAR(64)           NOT NULL COMMENT 'workspace id',
    type         VARCHAR(64)           NOT NULL COMMENT 'type, agent, workflow',
    app_id       VARCHAR(64)           DEFAULT NULL,
    config       LONGTEXT              DEFAULT NULL COMMENT 'component config',
    description  VARCHAR(4096)         DEFAULT NULL COMMENT 'description',
    status       TINYINT               DEFAULT NULL COMMENT 'status',
    creator      VARCHAR(64)           DEFAULT NULL COMMENT 'creator uid',
    modifier     VARCHAR(64)           DEFAULT NULL COMMENT 'modifier uid',
    need_update  TINYINT               DEFAULT NULL COMMENT '0-no need update, 1-need update',
    tenant_id    BIGINT                DEFAULT 0,
    PRIMARY KEY (id),
    KEY idx_workspace_type_status_appcode (workspace_id, type, app_id)
);

/******************************************/
/*   table = application_version          */
/******************************************/
DROP TABLE IF EXISTS application_version;
CREATE TABLE application_version
(
    id           BIGINT AUTO_INCREMENT NOT NULL COMMENT 'pk',
    app_id       VARCHAR(64)           NOT NULL COMMENT 'app id',
    workspace_id VARCHAR(64)           NOT NULL COMMENT 'workspace id',
    config       LONGTEXT              DEFAULT NULL COMMENT 'app config',
    status       TINYINT               NOT NULL COMMENT 'status',
    version      VARCHAR(32)           NOT NULL DEFAULT '0.0.1' COMMENT 'version name',
    description  VARCHAR(4096)         DEFAULT NULL COMMENT 'version description',
    gmt_create   TIMESTAMP             NOT NULL DEFAULT CURRENT_TIMESTAMP,
    gmt_modified TIMESTAMP             NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creator      VARCHAR(64)           NOT NULL COMMENT 'creator uid',
    modifier     VARCHAR(64)           NOT NULL COMMENT 'modifier uid',
    tenant_id    BIGINT                DEFAULT 0,
    PRIMARY KEY (id),
    KEY idx_workspace_app_version (workspace_id, app_id, version)
);

/******************************************/
/*   table = document                     */
/******************************************/
DROP TABLE IF EXISTS document;
CREATE TABLE document
(
    id             BIGINT AUTO_INCREMENT NOT NULL COMMENT 'pk',
    workspace_id   VARCHAR(64)           NOT NULL COMMENT 'workspace id',
    kb_id          VARCHAR(64)           NOT NULL COMMENT 'knowledge base id',
    doc_id         VARCHAR(64)           NOT NULL COMMENT 'document id',
    type           VARCHAR(64)           NOT NULL COMMENT 'type: file, url',
    status         TINYINT               NOT NULL DEFAULT 1 COMMENT 'status: 0-deleted, 1-normal',
    enabled        TINYINT               NOT NULL DEFAULT 1 COMMENT 'enabled: 0-disabled, 1-enabled',
    name           VARCHAR(255)          NOT NULL COMMENT 'document name',
    format         VARCHAR(64)           NOT NULL COMMENT 'document format',
    size           BIGINT                NOT NULL DEFAULT 0 COMMENT 'document size',
    metadata       TEXT                  DEFAULT NULL COMMENT 'document metadata',
    index_status   TINYINT               NOT NULL DEFAULT 1 COMMENT 'Index status',
    path           VARCHAR(512)          NOT NULL COMMENT 'storage path',
    parsed_path    VARCHAR(512)          DEFAULT NULL COMMENT 'parsed path',
    process_config TEXT                  DEFAULT NULL COMMENT 'document chunk config',
    source         VARCHAR(255)          DEFAULT NULL COMMENT 'doc source',
    error          TEXT                  DEFAULT NULL,
    gmt_create     TIMESTAMP             NOT NULL DEFAULT CURRENT_TIMESTAMP,
    gmt_modified   TIMESTAMP             NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creator        VARCHAR(64)           NOT NULL COMMENT 'creator uid',
    modifier       VARCHAR(64)           NOT NULL COMMENT 'modifier uid',
    tenant_id      BIGINT                DEFAULT 0,
    PRIMARY KEY (id),
    UNIQUE KEY uk_document_id (doc_id),
    KEY idx_workspace_kb (workspace_id, kb_id)
);

/******************************************/
/*   table = knowledge_base               */
/******************************************/
DROP TABLE IF EXISTS knowledge_base;
CREATE TABLE knowledge_base
(
    id             BIGINT AUTO_INCREMENT NOT NULL COMMENT 'pk',
    workspace_id   VARCHAR(64)           NOT NULL COMMENT 'workspace id',
    kb_id          VARCHAR(64)           NOT NULL COMMENT 'knowledge base id',
    type           VARCHAR(64)           NOT NULL COMMENT 'unstructured',
    status         TINYINT               NOT NULL DEFAULT 1 COMMENT 'status: 0-deleted, 1-normal',
    name           VARCHAR(255)          NOT NULL COMMENT 'knowledge base name',
    description    VARCHAR(4096)         DEFAULT NULL COMMENT 'knowledge base description',
    process_config TEXT                  DEFAULT NULL COMMENT 'process config',
    index_config   TEXT                  DEFAULT NULL COMMENT 'index config',
    search_config  TEXT                  DEFAULT NULL COMMENT 'search config',
    total_docs     BIGINT                NOT NULL DEFAULT 0 COMMENT 'total docs count',
    gmt_create     TIMESTAMP             NOT NULL DEFAULT CURRENT_TIMESTAMP,
    gmt_modified   TIMESTAMP             NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creator        VARCHAR(64)           NOT NULL COMMENT 'creator uid',
    modifier       VARCHAR(64)           NOT NULL COMMENT 'modifier uid',
    tenant_id      BIGINT                DEFAULT 0,
    PRIMARY KEY (id),
    UNIQUE KEY uk_kb_id (kb_id),
    KEY idx_workspace_status_name (workspace_id)
);

/******************************************/
/*   table = mcp_server                   */
/******************************************/
DROP TABLE IF EXISTS mcp_server;
CREATE TABLE mcp_server
(
    id            BIGINT AUTO_INCREMENT NOT NULL COMMENT 'pk',
    gmt_create    TIMESTAMP             NOT NULL COMMENT 'create time',
    gmt_modified  TIMESTAMP             NOT NULL COMMENT 'modified time',
    server_code   VARCHAR(64)           NOT NULL COMMENT 'server code',
    name          VARCHAR(64)           NOT NULL COMMENT 'server name',
    description   VARCHAR(1024)         DEFAULT NULL COMMENT 'description',
    source        VARCHAR(128)          DEFAULT NULL COMMENT 'server source',
    deploy_env    VARCHAR(16)           DEFAULT NULL COMMENT 'deploy environment local/remote',
    type          VARCHAR(32)           NOT NULL COMMENT 'server type OFFICIAL/CUSTOMER',
    deploy_config TEXT                  NOT NULL COMMENT 'deploy config',
    workspace_id  VARCHAR(64)           DEFAULT NULL COMMENT 'workspace id',
    account_id    VARCHAR(64)           DEFAULT NULL COMMENT 'uid',
    status        TINYINT               NOT NULL COMMENT 'status 0 unable 1 normal 3 deleted',
    biz_type      VARCHAR(512)          DEFAULT NULL COMMENT 'biz type',
    detail_config TEXT                  DEFAULT NULL COMMENT 'server detail',
    host          VARCHAR(1024)         DEFAULT NULL COMMENT 'host address',
    install_type  VARCHAR(32)           DEFAULT NULL COMMENT 'install_type npx/uvx/sse',
    tenant_id     BIGINT                DEFAULT 0,
    PRIMARY KEY (id),
    KEY idx_code (server_code),
    KEY idx_server_name (name),
    KEY idx_name_status (status, name),
    KEY idx_type (type)
);

/******************************************/
/*   table = model                        */
/******************************************/
DROP TABLE IF EXISTS model;
CREATE TABLE model
(
    id           BIGINT AUTO_INCREMENT NOT NULL COMMENT 'pk',
    workspace_id VARCHAR(64)           DEFAULT NULL COMMENT 'workspace id',
    icon         VARCHAR(255)          DEFAULT NULL COMMENT 'icon',
    name         VARCHAR(100)          DEFAULT NULL COMMENT 'model name',
    type         VARCHAR(100)          DEFAULT 'LLM' COMMENT 'model type: LLM',
    mode         VARCHAR(100)          DEFAULT 'chat' COMMENT 'mode',
    model_id     VARCHAR(100)          NOT NULL COMMENT 'model id',
    provider     VARCHAR(100)          NOT NULL COMMENT 'provider',
    enable       TINYINT               DEFAULT 1 COMMENT 'enable, 0: disabled, 1: enabled',
    tags         VARCHAR(255)          DEFAULT NULL COMMENT 'tags',
    source       VARCHAR(100)          NOT NULL DEFAULT 'preset' COMMENT 'source, preset, custom',
    gmt_create   TIMESTAMP             DEFAULT CURRENT_TIMESTAMP COMMENT 'create time',
    gmt_modified TIMESTAMP             DEFAULT CURRENT_TIMESTAMP COMMENT 'modified time',
    creator      VARCHAR(64)           DEFAULT NULL COMMENT 'creator',
    modifier     VARCHAR(64)           DEFAULT NULL COMMENT 'modifier',
    tenant_id    BIGINT                DEFAULT 0,
    PRIMARY KEY (id),
    KEY idx_account_workspace_provider_model (workspace_id, provider, model_id)
);

/******************************************/
/*   table = plugin                       */
/******************************************/
DROP TABLE IF EXISTS plugin;
CREATE TABLE plugin
(
    id           BIGINT AUTO_INCREMENT NOT NULL COMMENT 'pk',
    plugin_id    VARCHAR(64)           NOT NULL COMMENT 'biz id',
    workspace_id VARCHAR(64)           NOT NULL COMMENT 'workspace id',
    type         VARCHAR(64)           NOT NULL COMMENT 'type: 1: official, 2: custom',
    status       TINYINT               NOT NULL DEFAULT 1 COMMENT 'status: 0-deleted, 1-normal',
    name         VARCHAR(255)          NOT NULL COMMENT 'plugin name',
    description  VARCHAR(4096)         DEFAULT NULL COMMENT 'plugin description',
    config       TEXT                  DEFAULT NULL COMMENT 'plugin config',
    source       VARCHAR(64)           NOT NULL COMMENT 'plugin source',
    gmt_create   TIMESTAMP             NOT NULL DEFAULT CURRENT_TIMESTAMP,
    gmt_modified TIMESTAMP             NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creator      VARCHAR(64)           NOT NULL COMMENT 'creator uid',
    modifier     VARCHAR(64)           NOT NULL COMMENT 'modifier uid',
    tenant_id    BIGINT                DEFAULT 0,
    PRIMARY KEY (id),
    UNIQUE KEY uk_plugin_id (plugin_id),
    KEY idx_workspace_type_plugin (workspace_id, type)
);

/******************************************/
/*   table = provider                     */
/******************************************/
DROP TABLE IF EXISTS provider;
CREATE TABLE provider
(
    id                    BIGINT AUTO_INCREMENT NOT NULL COMMENT 'pk',
    workspace_id          VARCHAR(64)           DEFAULT NULL COMMENT 'workspace id',
    icon                  VARCHAR(255)          DEFAULT NULL COMMENT 'provider icon',
    name                  VARCHAR(255)          DEFAULT NULL COMMENT 'provider name',
    description           VARCHAR(1024)         DEFAULT NULL COMMENT 'provider description',
    provider              VARCHAR(255)          NOT NULL COMMENT 'provider',
    enable                TINYINT               DEFAULT 1 COMMENT 'enable, 0: disabled, 1: enabled',
    source                VARCHAR(64)           NOT NULL DEFAULT 'preset' COMMENT 'source, preset, custom',
    credential            VARCHAR(1024)         DEFAULT NULL COMMENT 'access credential, json',
    supported_model_types VARCHAR(255)          DEFAULT NULL COMMENT 'model type',
    protocol              VARCHAR(64)           DEFAULT NULL COMMENT 'protocol, openai',
    gmt_create            TIMESTAMP             DEFAULT CURRENT_TIMESTAMP COMMENT 'create time',
    gmt_modified          TIMESTAMP             DEFAULT CURRENT_TIMESTAMP COMMENT 'modified time',
    creator               VARCHAR(64)           DEFAULT NULL COMMENT 'creator',
    modifier              VARCHAR(64)           DEFAULT NULL COMMENT 'modifier',
    tenant_id             BIGINT                DEFAULT 0,
    PRIMARY KEY (id),
    KEY idx_account_workspace_provider (workspace_id, provider)
);

/******************************************/
/*   table = reference                    */
/******************************************/
DROP TABLE IF EXISTS reference;
CREATE TABLE reference
(
    id           BIGINT AUTO_INCREMENT NOT NULL COMMENT 'pk',
    gmt_create   TIMESTAMP             NOT NULL COMMENT 'create time',
    gmt_modified TIMESTAMP             NOT NULL COMMENT 'modified time',
    main_code    VARCHAR(64)           NOT NULL COMMENT 'entity code',
    main_type    TINYINT               NOT NULL COMMENT 'entity type',
    refer_code   VARCHAR(64)           NOT NULL COMMENT 'refer code',
    refer_type   TINYINT               NOT NULL COMMENT 'refer type',
    workspace_id VARCHAR(64)           NOT NULL DEFAULT '1' COMMENT 'workspace id',
    tenant_id    BIGINT                DEFAULT 0,
    PRIMARY KEY (id),
    KEY idx_refer_code (refer_code),
    KEY idx_main_code_workspace_type (workspace_id, main_code, main_type)
);

/******************************************/
/*   table = tool                         */
/******************************************/
DROP TABLE IF EXISTS tool;
CREATE TABLE tool
(
    id           BIGINT AUTO_INCREMENT NOT NULL COMMENT 'pk',
    plugin_id    VARCHAR(64)           NOT NULL COMMENT 'plugin id',
    tool_id      VARCHAR(64)           NOT NULL COMMENT 'tool id',
    workspace_id VARCHAR(64)           NOT NULL COMMENT 'workspace id',
    status       TINYINT               NOT NULL DEFAULT 1 COMMENT 'status: 0-deleted, 1-normal',
    enabled      TINYINT               NOT NULL DEFAULT 1 COMMENT 'enabled: 0-disabled, 1-enabled',
    test_status  TINYINT               NOT NULL DEFAULT 1 COMMENT 'test status',
    name         VARCHAR(255)          NOT NULL COMMENT 'tool name',
    description  VARCHAR(4096)         DEFAULT NULL COMMENT 'tool description',
    config       LONGTEXT              NOT NULL COMMENT 'tool config',
    api_schema   LONGTEXT              NOT NULL COMMENT 'tool api schema',
    gmt_create   TIMESTAMP             NOT NULL DEFAULT CURRENT_TIMESTAMP,
    gmt_modified TIMESTAMP             NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creator      VARCHAR(64)           NOT NULL COMMENT 'creator uid',
    modifier     VARCHAR(64)           NOT NULL COMMENT 'modifier uid',
    tenant_id    BIGINT                DEFAULT 0,
    PRIMARY KEY (id),
    UNIQUE KEY uk_tool_id (tool_id),
    KEY idx_workspace_plugin (workspace_id, plugin_id)
);

/******************************************/
/*   table = workspace                    */
/******************************************/
DROP TABLE IF EXISTS workspace;
CREATE TABLE workspace
(
    id           BIGINT AUTO_INCREMENT NOT NULL COMMENT 'pk',
    workspace_id VARCHAR(64)           NOT NULL COMMENT 'workspace id',
    account_id   VARCHAR(64)           NOT NULL COMMENT 'account id',
    status       TINYINT               NOT NULL DEFAULT 1 COMMENT 'status: 0-deleted, 1-normal',
    name         VARCHAR(255)          NOT NULL COMMENT 'workspace name',
    description  VARCHAR(4096)         DEFAULT NULL COMMENT 'workspace description',
    config       TEXT                  DEFAULT NULL COMMENT 'workspace config',
    gmt_create   TIMESTAMP             NOT NULL DEFAULT CURRENT_TIMESTAMP,
    gmt_modified TIMESTAMP             NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creator      VARCHAR(64)           NOT NULL COMMENT 'creator uid',
    modifier     VARCHAR(64)           NOT NULL COMMENT 'modifier uid',
    tenant_id    BIGINT                DEFAULT 0,
    PRIMARY KEY (id),
    UNIQUE KEY uk_workspace_id (workspace_id),
    KEY idx_account_id_workspace (account_id)
);

/******************************************/
/*   Initial Data                         */
/******************************************/

-- init account
INSERT INTO account (account_id, username, email, mobile, password, type, status, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('10000', 'saa', 'ken.lj.hz@gmail.com', NULL,
        '$argon2id$v=19$m=66536,t=2,p=1$KSDQowfZxDjKLqBtxFNRng$znU0oQFQs2shR9la4S11n7d0LpGApmSBXvDOXuhbR40',
        'admin', 1, NOW(), NOW(), '10000', '10000', 0);

-- init api_key
INSERT INTO api_key (account_id, api_key, status, description, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('10000', '+Ae6iTvRFwv7auIV/RN5vxanWB07uxn3CH9Za7EPTMA9Mq4eNRK8K0sprMrUEaYM', 1, '11',
        NOW(), NOW(), '10000', '10000', 0);

-- init workspace
INSERT INTO workspace (workspace_id, account_id, status, name, description, config, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', '10000', 1, 'Default Workspace', 'Default Workspace', NULL, NOW(), NOW(), '10000', '10000', 0);

-- init provider
INSERT INTO provider (workspace_id, icon, name, description, provider, enable, source, credential, supported_model_types, protocol, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'Tongyi', 'Tongyi', 'Tongyi', 1, 'preset',
        '{"endpoint":"https://dashscope.aliyuncs.com/compatible-mode/v1","api_key":"gA64aJ7fOMLT/J2DxCxZoHbVgUHeUCyHwpWT/pzH8pyvPDacWMkTiy/hf1lxdSIkkDfeLUbDO9Jeo+Uw0bjEBhSBy6tXWfAEHwD2MXZNd+3FjCSW2w+6WHN7hxG5ObVGyabUZiDZoiTrDN83XUGCaLvc5+qOUtj0mR6pY4KuY9QaDBV/bzBNr8AgHPOZJWxmvNpQcXvZ3yieZorc4g4942ivbNks+bDYobOiZEVSig9fQTd+jWNnqtnI7S5ak29V4tNp9SOsY0v8vmlIJi+9+HpG6z+plM+7KMU0l/WfYiTi0RjZ4DHy8AUM3iJkO/VL7HmKrVUkjzfzqYc9SR552g=="}',
        NULL, 'OpenAI', NOW(), NOW(), NULL, NULL, 0);

-- init model
INSERT INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'qwen-max', 'llm', 'chat', 'qwen-max', 'Tongyi', 1, 'web_search,function_call', 'preset', NOW(), NOW(), NULL, NULL, 0);

INSERT INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'qwen-max-latest', 'llm', 'chat', 'qwen-max-latest', 'Tongyi', 1, 'web_search,function_call,reasoning', 'preset', NOW(), NOW(), NULL, NULL, 0);

INSERT INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'qwen-plus', 'llm', 'chat', 'qwen-plus', 'Tongyi', 1, 'web_search,function_call', 'preset', NOW(), NOW(), NULL, NULL, 0);

INSERT INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'qwen-plus-latest', 'llm', 'chat', 'qwen-plus-latest', 'Tongyi', 1, 'web_search,function_call,reasoning', 'preset', NOW(), NOW(), NULL, NULL, 0);

INSERT INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'qwen-turbo', 'llm', 'chat', 'qwen-turbo', 'Tongyi', 1, 'web_search,function_call', 'preset', NOW(), NOW(), NULL, NULL, 0);

INSERT INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'qwen-turbo-latest', 'llm', 'chat', 'qwen-turbo-latest', 'Tongyi', 1, 'web_search,function_call,reasoning', 'preset', NOW(), NOW(), NULL, NULL, 0);

INSERT INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'qwen3-235b-a22b', 'llm', 'chat', 'qwen3-235b-a22b', 'Tongyi', 1, 'function_call,reasoning', 'preset', NOW(), NOW(), NULL, NULL, 0);

INSERT INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'qwen3-30b-a3b', 'llm', 'chat', 'qwen3-30b-a3b', 'Tongyi', 1, 'function_call,reasoning', 'preset', NOW(), NOW(), NULL, NULL, 0);

INSERT INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'qwen3-32b', 'llm', 'chat', 'qwen3-32b', 'Tongyi', 1, 'function_call,reasoning', 'preset', NOW(), NOW(), NULL, NULL, 0);

INSERT INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'qwen3-14b', 'llm', 'chat', 'qwen3-14b', 'Tongyi', 1, 'function_call,reasoning', 'preset', NOW(), NOW(), NULL, NULL, 0);

INSERT INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'qwen3-8b', 'llm', 'chat', 'qwen3-8b', 'Tongyi', 1, 'function_call,reasoning', 'preset', NOW(), NOW(), NULL, NULL, 0);

INSERT INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'qwen3-4b', 'llm', 'chat', 'qwen3-4b', 'Tongyi', 1, 'function_call,reasoning', 'preset', NOW(), NOW(), NULL, NULL, 0);

INSERT INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'qwen3-1.7b', 'llm', 'chat', 'qwen3-1.7b', 'Tongyi', 1, 'function_call,reasoning', 'preset', NOW(), NOW(), NULL, NULL, 0);

INSERT INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'qwen3-0.6b', 'llm', 'chat', 'qwen3-0.6b', 'Tongyi', 1, 'function_call,reasoning', 'preset', NOW(), NOW(), NULL, NULL, 0);

INSERT INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'qwen-vl-max', 'llm', 'chat', 'qwen-vl-max', 'Tongyi', 1, 'vision,function_call', 'preset', NOW(), NOW(), NULL, NULL, 0);

INSERT INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'qwen-vl-plus', 'llm', 'chat', 'qwen-vl-plus', 'Tongyi', 1, 'vision,function_call', 'preset', NOW(), NOW(), NULL, NULL, 0);

INSERT INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'qvq-max', 'llm', 'chat', 'qvq-max', 'Tongyi', 1, 'vision,reasoning', 'preset', NOW(), NOW(), NULL, NULL, 0);

INSERT INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'qwq-plus', 'llm', 'chat', 'qwq-plus', 'Tongyi', 1, 'reasoning,function_call', 'preset', NOW(), NOW(), NULL, NULL, 0);

INSERT INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'text-embedding-v1', 'text_embedding', 'chat', 'text-embedding-v1', 'Tongyi', 1, 'embedding', 'preset', NOW(), NOW(), NULL, NULL, 0);

INSERT INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'text-embedding-v2', 'text_embedding', 'chat', 'text-embedding-v2', 'Tongyi', 1, 'embedding', 'preset', NOW(), NOW(), NULL, NULL, 0);

INSERT INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'text-embedding-v3', 'text_embedding', 'chat', 'text-embedding-v3', 'Tongyi', 1, 'embedding', 'preset', NOW(), NOW(), NULL, NULL, 0);

INSERT INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'gte-rerank-v2', 'rerank', 'chat', 'gte-rerank-v2', 'Tongyi', 1, NULL, 'preset', NOW(), NOW(), NULL, NULL, 0);

INSERT INTO model (workspace_id, icon, name, type, mode, model_id, provider, enable, tags, source, gmt_create, gmt_modified, creator, modifier, tenant_id)
VALUES ('1', NULL, 'deepseek-r1', 'llm', 'chat', 'deepseek-r1', 'Tongyi', 1, 'reasoning', 'preset', NOW(), NOW(), NULL, NULL, 0);

/******************************************/
/*   Admin compatibility tables           */
/******************************************/

CREATE TABLE IF NOT EXISTS dataset (
    id BIGINT NOT NULL AUTO_INCREMENT,
    name VARCHAR(255) NOT NULL,
    description TEXT DEFAULT NULL,
    columns_config LONGTEXT DEFAULT NULL,
    data_count INT NOT NULL DEFAULT '0',
    create_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted TINYINT NOT NULL DEFAULT '0',
    PRIMARY KEY (id),
    KEY idx_dataset_deleted (deleted),
    KEY idx_dataset_update_time (update_time)
);

CREATE TABLE IF NOT EXISTS dataset_version (
    id BIGINT NOT NULL AUTO_INCREMENT,
    dataset_id BIGINT NOT NULL,
    version VARCHAR(32) NOT NULL,
    description TEXT DEFAULT NULL,
    data_count INT NOT NULL DEFAULT '0',
    status VARCHAR(32) NOT NULL DEFAULT 'draft',
    experiments TEXT DEFAULT NULL,
    dataset_items TEXT DEFAULT NULL,
    columns_config LONGTEXT DEFAULT NULL,
    create_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_dataset_version (dataset_id, version),
    KEY idx_dataset_version_dataset_id (dataset_id),
    KEY idx_dataset_version_update_time (update_time)
);

CREATE TABLE IF NOT EXISTS dataset_item (
    id BIGINT NOT NULL AUTO_INCREMENT,
    dataset_id BIGINT NOT NULL,
    columns_config LONGTEXT DEFAULT NULL,
    data_content LONGTEXT DEFAULT NULL,
    remark TEXT DEFAULT NULL,
    create_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted TINYINT NOT NULL DEFAULT '0',
    PRIMARY KEY (id),
    KEY idx_dataset_item_dataset_id (dataset_id),
    KEY idx_dataset_item_deleted (deleted),
    KEY idx_dataset_item_update_time (update_time)
);

CREATE TABLE IF NOT EXISTS evaluator (
    id BIGINT NOT NULL AUTO_INCREMENT,
    name VARCHAR(255) NOT NULL,
    description TEXT DEFAULT NULL,
    latest_version VARCHAR(32) DEFAULT NULL,
    model_name VARCHAR(255) DEFAULT NULL,
    prompt LONGTEXT DEFAULT NULL,
    model_config LONGTEXT DEFAULT NULL,
    variables LONGTEXT DEFAULT NULL,
    create_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted TINYINT NOT NULL DEFAULT '0',
    PRIMARY KEY (id),
    KEY idx_evaluator_deleted (deleted),
    KEY idx_evaluator_update_time (update_time)
);

CREATE TABLE IF NOT EXISTS evaluator_version (
    id BIGINT NOT NULL AUTO_INCREMENT,
    evaluator_id BIGINT NOT NULL,
    description TEXT DEFAULT NULL,
    version VARCHAR(32) NOT NULL,
    model_config LONGTEXT DEFAULT NULL,
    prompt LONGTEXT DEFAULT NULL,
    variables LONGTEXT DEFAULT NULL,
    status VARCHAR(32) DEFAULT NULL,
    experiments TEXT DEFAULT NULL,
    create_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_evaluator_version (evaluator_id, version),
    KEY idx_evaluator_version_evaluator_id (evaluator_id),
    KEY idx_evaluator_version_update_time (update_time)
);

CREATE TABLE IF NOT EXISTS evaluator_template (
    id BIGINT NOT NULL AUTO_INCREMENT,
    evaluator_template_key VARCHAR(255) NOT NULL,
    template_desc VARCHAR(255) DEFAULT NULL,
    template LONGTEXT,
    variables LONGTEXT DEFAULT NULL,
    model_config LONGTEXT DEFAULT NULL,
    create_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_evaluator_template_key (evaluator_template_key),
    KEY idx_evaluator_template_desc (template_desc)
);

CREATE TABLE IF NOT EXISTS experiment (
    id BIGINT NOT NULL AUTO_INCREMENT,
    name VARCHAR(255) NOT NULL,
    description TEXT DEFAULT NULL,
    dataset_id BIGINT DEFAULT NULL,
    dataset_version_id BIGINT DEFAULT NULL,
    dataset_version VARCHAR(32) DEFAULT NULL,
    evaluation_object_config LONGTEXT DEFAULT NULL,
    evaluator_config LONGTEXT DEFAULT NULL,
    evaluator_id VARCHAR(64) DEFAULT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'created',
    progress INT NOT NULL DEFAULT '0',
    complete_time TIMESTAMP DEFAULT NULL,
    create_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_experiment_dataset_id (dataset_id),
    KEY idx_experiment_dataset_version_id (dataset_version_id),
    KEY idx_experiment_status (status),
    KEY idx_experiment_update_time (update_time)
);

CREATE TABLE IF NOT EXISTS experiment_result (
    id BIGINT NOT NULL AUTO_INCREMENT,
    experiment_id BIGINT NOT NULL,
    input LONGTEXT DEFAULT NULL,
    actual_output LONGTEXT DEFAULT NULL,
    reference_output LONGTEXT DEFAULT NULL,
    score DECIMAL(6,4) DEFAULT NULL,
    reason TEXT DEFAULT NULL,
    evaluation_time TIMESTAMP DEFAULT NULL,
    evaluator_version_id BIGINT DEFAULT NULL,
    create_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_experiment_result_experiment_id (experiment_id),
    KEY idx_experiment_result_evaluator_version_id (evaluator_version_id),
    KEY idx_experiment_result_update_time (update_time)
);

CREATE TABLE IF NOT EXISTS prompt (
    id BIGINT NOT NULL AUTO_INCREMENT,
    prompt_key VARCHAR(255) NOT NULL,
    prompt_desc VARCHAR(255) DEFAULT NULL,
    prompt_description VARCHAR(255) DEFAULT NULL,
    latest_version VARCHAR(32) DEFAULT NULL,
    latest_version_status VARCHAR(32) DEFAULT NULL,
    tags VARCHAR(255) DEFAULT NULL,
    create_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    update_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_prompt_key (prompt_key),
    KEY idx_prompt_update_time (update_time)
);

CREATE TABLE IF NOT EXISTS prompt_version (
    id BIGINT NOT NULL AUTO_INCREMENT,
    version VARCHAR(32) NOT NULL,
    prompt_key VARCHAR(255) NOT NULL,
    version_desc VARCHAR(255) DEFAULT NULL,
    version_description VARCHAR(255) DEFAULT NULL,
    template LONGTEXT,
    variables LONGTEXT DEFAULT NULL,
    model_config LONGTEXT DEFAULT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pre',
    create_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    update_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    previous_version VARCHAR(32) DEFAULT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_prompt_key_version (prompt_key, version),
    KEY idx_prompt_version_prompt_key (prompt_key),
    KEY idx_prompt_version_status (status),
    KEY idx_prompt_version_update_time (update_time)
);

CREATE TABLE IF NOT EXISTS prompt_build_template (
    id BIGINT NOT NULL AUTO_INCREMENT,
    prompt_template_key VARCHAR(255) NOT NULL,
    tags VARCHAR(255) DEFAULT NULL,
    template_desc VARCHAR(255) DEFAULT NULL,
    template LONGTEXT,
    variables LONGTEXT DEFAULT NULL,
    model_config LONGTEXT DEFAULT NULL,
    create_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_prompt_template_key (prompt_template_key),
    KEY idx_prompt_build_template_tags (tags)
);

CREATE TABLE IF NOT EXISTS model_config (
    id BIGINT NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    model_name VARCHAR(100) NOT NULL,
    base_url VARCHAR(500) NOT NULL,
    api_key VARCHAR(500) NOT NULL,
    default_parameters LONGTEXT DEFAULT NULL,
    supported_parameters LONGTEXT DEFAULT NULL,
    status TINYINT NOT NULL DEFAULT '1',
    create_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted TINYINT NOT NULL DEFAULT '0',
    PRIMARY KEY (id),
    UNIQUE KEY uk_model_config_name (name),
    KEY idx_model_config_provider (provider),
    KEY idx_model_config_status (status),
    KEY idx_model_config_update_time (update_time)
);
