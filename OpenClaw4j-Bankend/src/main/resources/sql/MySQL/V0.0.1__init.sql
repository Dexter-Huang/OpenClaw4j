-- MySQL dump 10.13  Distrib 8.0.19, for Win64 (x86_64)
--
-- Host: 10.1.0.201    Database: cbes_llm
-- ------------------------------------------------------
-- Server version	8.0.26

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!50503 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Table structure for table `account`
--

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `account` (
                           `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'pk',
                           `account_id` varchar(64) NOT NULL COMMENT 'account id',
                           `username` varchar(255) NOT NULL COMMENT 'account name',
                           `email` varchar(255) DEFAULT NULL COMMENT 'account email',
                           `mobile` varchar(255) DEFAULT NULL COMMENT 'account mobile',
                           `password` varchar(255) NOT NULL COMMENT 'password',
                           `nickname` varchar(255) DEFAULT NULL COMMENT 'nickname',
                           `icon` varchar(255) DEFAULT NULL COMMENT 'account icon',
                           `type` varchar(64) NOT NULL COMMENT 'type: basic, admin',
                           `status` tinyint NOT NULL DEFAULT '1' COMMENT 'status: 0- deleted, 1- normal',
                           `gmt_create` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
                           `gmt_modified` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
                           `gmt_last_login` datetime DEFAULT NULL COMMENT 'account last login time',
                           `creator` varchar(64) NOT NULL COMMENT 'creator uid',
                           `modifier` varchar(64) NOT NULL COMMENT 'modifier uid',
                           `tenant_id` bigint DEFAULT '0',
                           PRIMARY KEY (`id`),
                           UNIQUE KEY `uk_account_id` (`account_id`),
                           KEY `idx_email_password` (`username`)
) ENGINE=InnoDB AUTO_INCREMENT=10001 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='account info';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `account`
--

LOCK TABLES `account` WRITE;
/*!40000 ALTER TABLE `account` DISABLE KEYS */;
INSERT INTO `account` VALUES (10000,'10000','saa','ken.lj.hz@gmail.com',NULL,'$argon2id$v=19$m=66536,t=2,p=1$KSDQowfZxDjKLqBtxFNRng$znU0oQFQs2shR9la4S11n7d0LpGApmSBXvDOXuhbR40',NULL,NULL,'admin',1,'2025-08-22 18:20:21','2025-08-22 18:20:21','2025-10-13 14:09:18','10000','10000',0);
/*!40000 ALTER TABLE `account` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `agent_schema`
--

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `agent_schema` (
                                `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'pk',
                                `agent_id` varchar(64) DEFAULT NULL COMMENT 'agent id',
                                `workspace_id` varchar(64) NOT NULL COMMENT 'workspace id',
                                `name` varchar(255) NOT NULL COMMENT 'agent name',
                                `description` varchar(4096) DEFAULT NULL COMMENT 'agent description',
                                `type` varchar(64) NOT NULL COMMENT 'agent type: ReactAgent, ParallelAgent, SequentialAgent, LLMRoutingAgent, LoopAgent',
                                `instruction` text COMMENT 'system instruction',
                                `input_keys` text COMMENT 'input keys JSON',
                                `output_key` varchar(255) DEFAULT NULL COMMENT 'output key',
                                `handle` longtext COMMENT 'handle configuration JSON',
                                `sub_agents` longtext COMMENT 'sub agents configuration JSON',
                                `yaml_schema` longtext COMMENT 'generated YAML schema',
                                `status` varchar(64) NOT NULL DEFAULT 'DRAFT' COMMENT 'agent status: DRAFT, PUBLISHED, ARCHIVED',
                                `enabled` tinyint NOT NULL DEFAULT '1' COMMENT 'enabled: 0-disabled, 1-enabled',
                                `gmt_create` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
                                `gmt_modified` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                                `creator` varchar(64) NOT NULL COMMENT 'creator uid',
                                `modifier` varchar(64) NOT NULL COMMENT 'modifier uid',
                                `tenant_id` bigint DEFAULT '0',
                                PRIMARY KEY (`id`),
                                UNIQUE KEY `uk_agent_id` (`agent_id`),
                                KEY `idx_workspace_type` (`workspace_id`,`type`),
                                KEY `idx_workspace_status` (`workspace_id`,`status`)
) ENGINE=InnoDB AUTO_INCREMENT=10000 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='agent schema info';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `agent_schema`
--

LOCK TABLES `agent_schema` WRITE;
/*!40000 ALTER TABLE `agent_schema` DISABLE KEYS */;
/*!40000 ALTER TABLE `agent_schema` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `api_key`
--

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `api_key` (
                           `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'pk',
                           `account_id` varchar(64) NOT NULL COMMENT 'uid',
                           `api_key` varchar(512) NOT NULL COMMENT 'api key',
                           `status` tinyint NOT NULL DEFAULT '1' COMMENT 'status: 0-deleted, 1-normal',
                           `description` varchar(4096) DEFAULT NULL COMMENT 'api key description',
                           `gmt_create` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
                           `gmt_modified` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
                           `creator` varchar(64) NOT NULL COMMENT 'creator uid',
                           `modifier` varchar(64) NOT NULL COMMENT 'creator uid',
                           `tenant_id` bigint DEFAULT '0',
                           PRIMARY KEY (`id`),
                           UNIQUE KEY `uk_api_key` (`api_key`),
                           KEY `idx_account_id` (`account_id`)
) ENGINE=InnoDB AUTO_INCREMENT=10001 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='api key info';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `api_key`
--

LOCK TABLES `api_key` WRITE;
/*!40000 ALTER TABLE `api_key` DISABLE KEYS */;
INSERT INTO `api_key` VALUES (10000,'10000','+Ae6iTvRFwv7auIV/RN5vxanWB07uxn3CH9Za7EPTMA9Mq4eNRK8K0sprMrUEaYM',1,'11','2025-08-23 18:48:26','2025-08-23 18:48:26','10000','10000',0);
/*!40000 ALTER TABLE `api_key` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `application`
--

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `application` (
                               `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'pk',
                               `workspace_id` varchar(64) NOT NULL COMMENT 'workspace id',
                               `app_id` varchar(64) NOT NULL COMMENT 'app id',
                               `name` varchar(255) NOT NULL COMMENT 'app name',
                               `description` varchar(4096) DEFAULT NULL COMMENT 'app description',
                               `icon` varchar(255) DEFAULT NULL COMMENT 'app icon',
                               `source` varchar(64) NOT NULL COMMENT 'app source',
                               `type` varchar(64) NOT NULL COMMENT 'type, agent, workflow',
                               `status` tinyint NOT NULL DEFAULT '1' COMMENT 'status, 0-deleted 1-draft; 2-published; 3-publishedEditing',
                               `gmt_create` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
                               `gmt_modified` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
                               `creator` varchar(64) NOT NULL COMMENT '创建者uid',
                               `modifier` varchar(64) NOT NULL COMMENT '修改者uid',
                               `tenant_id` bigint DEFAULT '0',
                               PRIMARY KEY (`id`),
                               UNIQUE KEY `uk_app_id` (`app_id`),
                               KEY `idx_workspace_type` (`workspace_id`,`type`)
) ENGINE=InnoDB AUTO_INCREMENT=10003 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='app info';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `application`
--

LOCK TABLES `application` WRITE;
/*!40000 ALTER TABLE `application` DISABLE KEYS */;
/*!40000 ALTER TABLE `application` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `application_component`
--

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `application_component` (
                                         `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'pk',
                                         `gmt_create` datetime NOT NULL COMMENT 'create time',
                                         `gmt_modified` datetime NOT NULL COMMENT 'modified time',
                                         `code` varchar(64) NOT NULL COMMENT 'component code',
                                         `name` varchar(128) NOT NULL COMMENT 'name',
                                         `workspace_id` varchar(64) NOT NULL COMMENT 'workspace id',
                                         `type` varchar(64) NOT NULL COMMENT 'type, agent, workflow',
                                         `app_id` varchar(64) DEFAULT NULL,
                                         `config` longtext COMMENT 'component config',
                                         `description` varchar(4096) DEFAULT NULL COMMENT 'description ',
                                         `status` tinyint DEFAULT NULL COMMENT 'status：0-deleted, 1, normal, 2-published',
                                         `creator` varchar(64) DEFAULT NULL COMMENT 'creator uid',
                                         `modifier` varchar(64) DEFAULT NULL COMMENT 'modifier uid',
                                         `need_update` tinyint DEFAULT NULL COMMENT '0-no need update, 1-need update',
                                         `tenant_id` bigint DEFAULT '0',
                                         PRIMARY KEY (`id`),
                                         KEY `idx_workspace_type_status_appcode` (`workspace_id`,`type`,`app_id`)
) ENGINE=InnoDB AUTO_INCREMENT=10000 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='app component info';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `application_component`
--

LOCK TABLES `application_component` WRITE;
/*!40000 ALTER TABLE `application_component` DISABLE KEYS */;
/*!40000 ALTER TABLE `application_component` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `application_version`
--

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `application_version` (
                                       `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'pk',
                                       `app_id` varchar(64) NOT NULL COMMENT 'app id',
                                       `workspace_id` varchar(64) NOT NULL COMMENT 'workspace id',
                                       `config` longtext COMMENT 'app config',
                                       `status` tinyint NOT NULL COMMENT 'status, 0-deleted 1-draft; 2-published; 3-publishedEditing',
                                       `version` varchar(32) NOT NULL DEFAULT '0.0.1' COMMENT 'version name',
                                       `description` varchar(4096) DEFAULT NULL COMMENT 'version description',
                                       `gmt_create` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
                                       `gmt_modified` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
                                       `creator` varchar(64) NOT NULL COMMENT 'creator uid',
                                       `modifier` varchar(64) NOT NULL COMMENT 'modifier uid',
                                       `tenant_id` bigint DEFAULT '0',
                                       PRIMARY KEY (`id`),
                                       KEY `idx_workspace_app_version` (`workspace_id`,`app_id`,`version`)
) ENGINE=InnoDB AUTO_INCREMENT=10006 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='app version info';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `application_version`
--

LOCK TABLES `application_version` WRITE;
/*!40000 ALTER TABLE `application_version` DISABLE KEYS */;
/*!40000 ALTER TABLE `application_version` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `document`
--

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `document` (
                            `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'pk',
                            `workspace_id` varchar(64) NOT NULL COMMENT 'workspace id',
                            `kb_id` varchar(64) NOT NULL COMMENT 'knowledge base id',
                            `doc_id` varchar(64) NOT NULL COMMENT 'document id',
                            `type` varchar(64) NOT NULL COMMENT 'type: file, url',
                            `status` tinyint NOT NULL DEFAULT '1' COMMENT 'status: 0-deleted, 1-normal',
                            `enabled` tinyint NOT NULL DEFAULT '1' COMMENT 'enabled: 0-disabled, 1-enabled',
                            `name` varchar(255) NOT NULL COMMENT 'document name',
                            `format` varchar(64) NOT NULL COMMENT 'document format',
                            `size` bigint NOT NULL DEFAULT '0' COMMENT 'document size',
                            `metadata` text COMMENT 'document metadata',
                            `index_status` tinyint NOT NULL DEFAULT '1' COMMENT 'Index status: 1: pending, 2: processing, 3: completed',
                            `path` varchar(512) NOT NULL COMMENT 'storage path',
                            `parsed_path` varchar(512) DEFAULT NULL COMMENT 'parsed path',
                            `process_config` text COMMENT 'document chunk config',
                            `source` varchar(255) DEFAULT NULL COMMENT 'doc source',
                            `error` text,
                            `gmt_create` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
                            `gmt_modified` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
                            `creator` varchar(64) NOT NULL COMMENT 'creator uid',
                            `modifier` varchar(64) NOT NULL COMMENT 'modifier uid',
                            `tenant_id` bigint DEFAULT '0',
                            PRIMARY KEY (`id`),
                            UNIQUE KEY `uk_document_id` (`doc_id`),
                            KEY `idx_workspace_kb` (`workspace_id`,`kb_id`)
) ENGINE=InnoDB AUTO_INCREMENT=10000 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='document info';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `document`
--


--
-- Table structure for table `knowledge_base`
--

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `knowledge_base` (
                                  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'pk',
                                  `workspace_id` varchar(64) NOT NULL COMMENT 'workspace id',
                                  `kb_id` varchar(64) NOT NULL COMMENT 'knowledge base id',
                                  `type` varchar(64) NOT NULL COMMENT 'unstructured',
                                  `status` tinyint NOT NULL DEFAULT '1' COMMENT 'status: 0-deleted, 1-normal',
                                  `name` varchar(255) NOT NULL COMMENT 'knowledge base name',
                                  `description` varchar(4096) DEFAULT NULL COMMENT 'knowledge base description',
                                  `process_config` text COMMENT 'process config',
                                  `index_config` text COMMENT 'index config',
                                  `search_config` text COMMENT 'search config',
                                  `total_docs` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'total docs count',
                                  `gmt_create` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
                                  `gmt_modified` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
                                  `creator` varchar(64) NOT NULL COMMENT 'creator uid',
                                  `modifier` varchar(64) NOT NULL COMMENT 'modifier uid',
                                  `tenant_id` bigint DEFAULT '0',
                                  PRIMARY KEY (`id`),
                                  UNIQUE KEY `uk_kb_id` (`kb_id`),
                                  KEY `idx_workspace_status_name` (`workspace_id`)
) ENGINE=InnoDB AUTO_INCREMENT=10000 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='knowledge base info';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `knowledge_base`
--

LOCK TABLES `knowledge_base` WRITE;
/*!40000 ALTER TABLE `knowledge_base` DISABLE KEYS */;
/*!40000 ALTER TABLE `knowledge_base` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `mcp_server`
--

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `mcp_server` (
                              `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'pk',
                              `gmt_create` datetime NOT NULL COMMENT 'create time',
                              `gmt_modified` datetime NOT NULL COMMENT 'modified time',
                              `server_code` varchar(64) NOT NULL COMMENT 'server code',
                              `name` varchar(64) NOT NULL COMMENT 'server name',
                              `description` varchar(1024) DEFAULT NULL COMMENT 'description',
                              `source` varchar(128) DEFAULT NULL COMMENT 'server source',
                              `deploy_env` varchar(16) DEFAULT NULL COMMENT 'deploy environment local/remote',
                              `type` varchar(32) NOT NULL COMMENT 'server type OFFICIAL/CUSTOMER',
                              `deploy_config` text NOT NULL COMMENT 'deploy config',
                              `workspace_id` varchar(64) DEFAULT NULL COMMENT 'workspace id',
                              `account_id` varchar(64) DEFAULT NULL COMMENT 'uid',
                              `status` tinyint NOT NULL COMMENT 'status 0 unable 1 normal 3 deleted',
                              `biz_type` varchar(512) DEFAULT NULL COMMENT 'biz type',
                              `detail_config` text COMMENT 'server detail',
                              `host` varchar(1024) DEFAULT NULL COMMENT 'host address',
                              `install_type` varchar(32) DEFAULT NULL COMMENT 'install_type npx/uvx/sse',
                              `tenant_id` bigint DEFAULT '0',
                              PRIMARY KEY (`id`),
                              KEY `idx_code` (`server_code`),
                              KEY `idx_server_name` (`name`),
                              KEY `idx_name_status` (`status`,`name`),
                              KEY `idx_type` (`type`)
) ENGINE=InnoDB AUTO_INCREMENT=10000 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='mcp server info';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `mcp_server`
--

LOCK TABLES `mcp_server` WRITE;
/*!40000 ALTER TABLE `mcp_server` DISABLE KEYS */;
/*!40000 ALTER TABLE `mcp_server` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `model`
--

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `model` (
                         `id` bigint NOT NULL AUTO_INCREMENT COMMENT 'pk',
                         `workspace_id` varchar(64) DEFAULT NULL COMMENT 'workspace id',
                         `icon` varchar(255) DEFAULT NULL COMMENT 'icon',
                         `name` varchar(100) DEFAULT NULL COMMENT 'model name',
                         `type` varchar(100) DEFAULT 'LLM' COMMENT 'model type: LLM',
                         `mode` varchar(100) DEFAULT 'chat' COMMENT 'mode',
                         `model_id` varchar(100) NOT NULL COMMENT 'model id',
                         `provider` varchar(100) NOT NULL COMMENT 'provider',
                         `enable` tinyint(1) DEFAULT '1' COMMENT 'enable, 0: disabled, 1: enabled',
                         `tags` varchar(255) DEFAULT NULL COMMENT 'tags',
                         `source` varchar(100) NOT NULL DEFAULT 'preset' COMMENT 'source, preset, custom',
                         `gmt_create` datetime DEFAULT CURRENT_TIMESTAMP COMMENT 'create time',
                         `gmt_modified` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'modified time',
                         `creator` varchar(64) DEFAULT NULL COMMENT 'creator',
                         `modifier` varchar(64) DEFAULT NULL COMMENT 'modifier',
                         `tenant_id` bigint DEFAULT '0',
                         PRIMARY KEY (`id`),
                         KEY `idx_account_workspace_provider_model` (`workspace_id`,`provider`,`model_id`)
) ENGINE=InnoDB AUTO_INCREMENT=10023 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='model info';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `model`
--

LOCK TABLES `model` WRITE;
/*!40000 ALTER TABLE `model` DISABLE KEYS */;
INSERT INTO `model` VALUES (10000,'1',NULL,'qwen-max','llm','chat','qwen-max','Tongyi',1,'web_search,function_call','preset','2025-08-22 18:20:21','2025-08-22 18:20:21',NULL,NULL,0),(10001,'1',NULL,'qwen-max-latest','llm','chat','qwen-max-latest','Tongyi',1,'web_search,function_call,reasoning','preset','2025-08-22 18:20:21','2025-08-22 18:20:21',NULL,NULL,0),(10002,'1',NULL,'qwen-plus','llm','chat','qwen-plus','Tongyi',1,'web_search,function_call','preset','2025-08-22 18:20:21','2025-08-22 18:20:21',NULL,NULL,0),(10003,'1',NULL,'qwen-plus-latest','llm','chat','qwen-plus-latest','Tongyi',1,'web_search,function_call,reasoning','preset','2025-08-22 18:20:21','2025-08-22 18:20:21',NULL,NULL,0),(10004,'1',NULL,'qwen-turbo','llm','chat','qwen-turbo','Tongyi',1,'web_search,function_call','preset','2025-08-22 18:20:21','2025-08-22 18:20:21',NULL,NULL,0),(10005,'1',NULL,'qwen-turbo-latest','llm','chat','qwen-turbo-latest','Tongyi',1,'web_search,function_call,reasoning','preset','2025-08-22 18:20:21','2025-08-22 18:20:21',NULL,NULL,0),(10006,'1',NULL,'qwen3-235b-a22b','llm','chat','qwen3-235b-a22b','Tongyi',1,'function_call,reasoning','preset','2025-08-22 18:20:21','2025-08-22 18:20:21',NULL,NULL,0),(10007,'1',NULL,'qwen3-30b-a3b','llm','chat','qwen3-30b-a3b','Tongyi',1,'function_call,reasoning','preset','2025-08-22 18:20:21','2025-08-22 18:20:21',NULL,NULL,0),(10008,'1',NULL,'qwen3-32b','llm','chat','qwen3-32b','Tongyi',1,'function_call,reasoning','preset','2025-08-22 18:20:21','2025-08-22 18:20:21',NULL,NULL,0),(10009,'1',NULL,'qwen3-14b','llm','chat','qwen3-14b','Tongyi',1,'function_call,reasoning','preset','2025-08-22 18:20:21','2025-08-22 18:20:21',NULL,NULL,0),(10010,'1',NULL,'qwen3-8b','llm','chat','qwen3-8b','Tongyi',1,'function_call,reasoning','preset','2025-08-22 18:20:21','2025-08-22 18:20:21',NULL,NULL,0),(10011,'1',NULL,'qwen3-4b','llm','chat','qwen3-4b','Tongyi',1,'function_call,reasoning','preset','2025-08-22 18:20:21','2025-08-22 18:20:21',NULL,NULL,0),(10012,'1',NULL,'qwen3-1.7b','llm','chat','qwen3-1.7b','Tongyi',1,'function_call,reasoning','preset','2025-08-22 18:20:21','2025-08-22 18:20:21',NULL,NULL,0),(10013,'1',NULL,'qwen3-0.6b','llm','chat','qwen3-0.6b','Tongyi',1,'function_call,reasoning','preset','2025-08-22 18:20:21','2025-08-22 18:20:21',NULL,NULL,0),(10014,'1',NULL,'qwen-vl-max','llm','chat','qwen-vl-max','Tongyi',1,'vision,function_call','preset','2025-08-22 18:20:21','2025-08-22 18:20:21',NULL,NULL,0),(10015,'1',NULL,'qwen-vl-plus','llm','chat','qwen-vl-plus','Tongyi',1,'vision,function_call','preset','2025-08-22 18:20:21','2025-08-22 18:20:21',NULL,NULL,0),(10016,'1',NULL,'qvq-max','llm','chat','qvq-max','Tongyi',1,'vision,reasoning','preset','2025-08-22 18:20:21','2025-08-22 18:20:21',NULL,NULL,0),(10017,'1',NULL,'qwq-plus','llm','chat','qwq-plus','Tongyi',1,'reasoning,function_call','preset','2025-08-22 18:20:21','2025-08-22 18:20:21',NULL,NULL,0),(10018,'1',NULL,'text-embedding-v1','text_embedding','chat','text-embedding-v1','Tongyi',1,'embedding','preset','2025-08-22 18:20:21','2025-08-22 18:20:21',NULL,NULL,0),(10019,'1',NULL,'text-embedding-v2','text_embedding','chat','text-embedding-v2','Tongyi',1,'embedding','preset','2025-08-22 18:20:22','2025-08-22 18:20:22',NULL,NULL,0),(10020,'1',NULL,'text-embedding-v3','text_embedding','chat','text-embedding-v3','Tongyi',1,'embedding','preset','2025-08-22 18:20:22','2025-08-22 18:20:22',NULL,NULL,0),(10021,'1',NULL,'gte-rerank-v2','rerank','chat','gte-rerank-v2','Tongyi',1,NULL,'preset','2025-08-22 18:20:22','2025-08-22 18:20:22',NULL,NULL,0),(10022,'1',NULL,'deepseek-r1','llm','chat','deepseek-r1','Tongyi',1,'reasoning','preset','2025-08-22 18:20:22','2025-08-22 18:20:22',NULL,NULL,0);
/*!40000 ALTER TABLE `model` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `plugin`
--

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `plugin` (
                          `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'pk',
                          `plugin_id` varchar(64) NOT NULL COMMENT 'biz id',
                          `workspace_id` varchar(64) NOT NULL COMMENT 'workspace id',
                          `type` varchar(64) NOT NULL COMMENT 'type: 1: official, 2: custom',
                          `status` tinyint NOT NULL DEFAULT '1' COMMENT 'status: 0-deleted, 1-normal',
                          `name` varchar(255) NOT NULL COMMENT 'plugin name',
                          `description` varchar(4096) DEFAULT NULL COMMENT 'plugin description',
                          `config` text COMMENT 'plugin config',
                          `source` varchar(64) NOT NULL COMMENT 'plugin source',
                          `gmt_create` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
                          `gmt_modified` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
                          `creator` varchar(64) NOT NULL COMMENT 'creator uid',
                          `modifier` varchar(64) NOT NULL COMMENT 'modifier uid',
                          `tenant_id` bigint DEFAULT '0',
                          PRIMARY KEY (`id`),
                          UNIQUE KEY `uk_plugin_id` (`plugin_id`),
                          KEY `idx_workspace_type` (`workspace_id`,`type`)
) ENGINE=InnoDB AUTO_INCREMENT=10000 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='plugin info';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `plugin`
--

LOCK TABLES `plugin` WRITE;
/*!40000 ALTER TABLE `plugin` DISABLE KEYS */;
/*!40000 ALTER TABLE `plugin` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `provider`
--

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `provider` (
                            `id` bigint NOT NULL AUTO_INCREMENT COMMENT 'pk',
                            `workspace_id` varchar(64) DEFAULT NULL COMMENT 'workspace id',
                            `icon` varchar(255) DEFAULT NULL COMMENT 'provider icon',
                            `name` varchar(255) DEFAULT NULL COMMENT 'provider name',
                            `description` varchar(1024) DEFAULT NULL COMMENT 'provider description',
                            `provider` varchar(255) NOT NULL COMMENT 'provider',
                            `enable` tinyint(1) DEFAULT '1' COMMENT 'enable, 0: disabled, 1: enabled',
                            `source` varchar(64) NOT NULL DEFAULT 'preset' COMMENT 'source, preset, custom',
                            `credential` varchar(1024) DEFAULT NULL COMMENT 'access credential，json',
                            `supported_model_types` varchar(255) DEFAULT NULL COMMENT 'model type',
                            `protocol` varchar(64) DEFAULT NULL COMMENT 'protocol, openai',
                            `gmt_create` datetime DEFAULT CURRENT_TIMESTAMP COMMENT 'create time',
                            `gmt_modified` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'modified time',
                            `creator` varchar(64) DEFAULT NULL COMMENT 'creator',
                            `modifier` varchar(64) DEFAULT NULL COMMENT 'modifier',
                            `tenant_id` bigint DEFAULT '0',
                            PRIMARY KEY (`id`),
                            KEY `idx_account_workspace_provider` (`workspace_id`,`provider`)
) ENGINE=InnoDB AUTO_INCREMENT=10001 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='provider info';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `provider`
--

LOCK TABLES `provider` WRITE;
/*!40000 ALTER TABLE `provider` DISABLE KEYS */;
INSERT INTO `provider` VALUES (10000,'1',NULL,'Tongyi','Tongyi','Tongyi',1,'preset','{\"endpoint\":\"https://dashscope.aliyuncs.com/compatible-mode/v1\",\"api_key\":\"gA64aJ7fOMLT/J2DxCxZoHbVgUHeUCyHwpWT/pzH8pyvPDacWMkTiy/hf1lxdSIkkDfeLUbDO9Jeo+Uw0bjEBhSBy6tXWfAEHwD2MXZNd+3FjCSW2w+6WHN7hxG5ObVGyabUZiDZoiTrDN83XUGCaLvc5+qOUtj0mR6pY4KuY9QaDBV/bzBNr8AgHPOZJWxmvNpQcXvZ3yieZorc4g4942ivbNks+bDYobOiZEVSig9fQTd+jWNnqtnI7S5ak29V4tNp9SOsY0v8vmlIJi+9+HpG6z+plM+7KMU0l/WfYiTi0RjZ4DHy8AUM3iJkO/VL7HmKrVUkjzfzqYc9SR552g==\"}',NULL,'OpenAI','2025-08-22 18:20:21','2025-08-24 17:37:34',NULL,'10000',0);
/*!40000 ALTER TABLE `provider` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `reference`
--
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `reference` (
                             `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'pk',
                             `gmt_create` datetime NOT NULL COMMENT 'create time',
                             `gmt_modified` datetime NOT NULL COMMENT 'modified time',
                             `main_code` varchar(64) NOT NULL COMMENT 'entity code',
                             `main_type` tinyint NOT NULL COMMENT 'entity time',
                             `refer_code` varchar(64) NOT NULL COMMENT 'refer code',
                             `refer_type` tinyint NOT NULL COMMENT 'refer type',
                             `workspace_id` varchar(64) NOT NULL DEFAULT '1' COMMENT 'workspace id',
                             `tenant_id` bigint DEFAULT '0',
                             PRIMARY KEY (`id`),
                             KEY `idx_refer_code` (`refer_code`),
                             KEY `idx_main_code_workspace_type` (`workspace_id`,`main_code`,`main_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='reference info';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `reference`
--

LOCK TABLES `reference` WRITE;
/*!40000 ALTER TABLE `reference` DISABLE KEYS */;
/*!40000 ALTER TABLE `reference` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `tool`
--

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `tool` (
                        `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'pk',
                        `plugin_id` varchar(64) NOT NULL COMMENT 'plugin id',
                        `tool_id` varchar(64) NOT NULL COMMENT 'tool id',
                        `workspace_id` varchar(64) NOT NULL COMMENT 'workspace id',
                        `status` tinyint NOT NULL DEFAULT '1' COMMENT 'status: 0-deleted, 1-normal',
                        `enabled` tinyint NOT NULL DEFAULT '1' COMMENT 'enabled: 0-disabled, 1-enabled',
                        `test_status` tinyint NOT NULL DEFAULT '1' COMMENT 'test status: 1: not tested, 2: passed, 3: failed',
                        `name` varchar(255) NOT NULL COMMENT 'tool name',
                        `description` varchar(4096) DEFAULT NULL COMMENT 'tool description',
                        `config` longtext NOT NULL COMMENT 'tool config',
                        `api_schema` longtext NOT NULL COMMENT 'tool api schema',
                        `gmt_create` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
                        `gmt_modified` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
                        `creator` varchar(64) NOT NULL COMMENT 'creator uid',
                        `modifier` varchar(64) NOT NULL COMMENT 'modifier uid',
                        `tenant_id` bigint DEFAULT '0',
                        PRIMARY KEY (`id`),
                        UNIQUE KEY `uk_tool_id` (`tool_id`),
                        KEY `idx_workspace_plugin` (`workspace_id`,`plugin_id`)
) ENGINE=InnoDB AUTO_INCREMENT=10000 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='tool info';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `tool`
--

LOCK TABLES `tool` WRITE;
/*!40000 ALTER TABLE `tool` DISABLE KEYS */;
/*!40000 ALTER TABLE `tool` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `workspace`
--
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `workspace` (
                             `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'pk',
                             `workspace_id` varchar(64) NOT NULL COMMENT 'workspace id',
                             `account_id` varchar(64) NOT NULL COMMENT 'account id',
                             `status` tinyint NOT NULL DEFAULT '1' COMMENT 'status: 0-deleted, 1-normal',
                             `name` varchar(255) NOT NULL COMMENT 'workspace name',
                             `description` varchar(4096) DEFAULT NULL COMMENT 'workspace description',
                             `config` text COMMENT 'workspace config',
                             `gmt_create` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
                             `gmt_modified` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
                             `creator` varchar(64) NOT NULL COMMENT 'creator uid',
                             `modifier` varchar(64) NOT NULL COMMENT 'creator uid',
                             `tenant_id` bigint DEFAULT '0',
                             PRIMARY KEY (`id`),
                             UNIQUE KEY `uk_workspace_id` (`workspace_id`),
                             KEY `idx_account_id` (`account_id`)
) ENGINE=InnoDB AUTO_INCREMENT=10001 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='workspace info';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `workspace`
--

LOCK TABLES `workspace` WRITE;
/*!40000 ALTER TABLE `workspace` DISABLE KEYS */;
INSERT INTO `workspace` VALUES (10000,'1','10000',1,'Default Workspace','Default Workspace',NULL,'2025-08-22 18:20:21','2025-08-22 18:20:21','10000','10000',0);
/*!40000 ALTER TABLE `workspace` ENABLE KEYS */;
UNLOCK TABLES;

/******************************************/
/*   Admin compatibility tables           */
/******************************************/

CREATE TABLE IF NOT EXISTS `dataset` (
                           `id` bigint unsigned NOT NULL AUTO_INCREMENT,
                           `name` varchar(255) NOT NULL,
                           `description` text DEFAULT NULL,
                           `columns_config` longtext DEFAULT NULL,
                           `data_count` int NOT NULL DEFAULT '0',
                           `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
                           `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                           `deleted` tinyint(1) NOT NULL DEFAULT '0',
                           PRIMARY KEY (`id`),
                           KEY `idx_dataset_deleted` (`deleted`),
                           KEY `idx_dataset_update_time` (`update_time`)
) ENGINE=InnoDB AUTO_INCREMENT=10000 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='admin dataset';

CREATE TABLE IF NOT EXISTS `dataset_version` (
                                   `id` bigint unsigned NOT NULL AUTO_INCREMENT,
                                   `dataset_id` bigint unsigned NOT NULL,
                                   `version` varchar(32) NOT NULL,
                                   `description` text DEFAULT NULL,
                                   `data_count` int NOT NULL DEFAULT '0',
                                   `status` varchar(32) NOT NULL DEFAULT 'draft',
                                   `experiments` text DEFAULT NULL,
                                   `dataset_items` text DEFAULT NULL,
                                   `columns_config` longtext DEFAULT NULL,
                                   `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
                                   `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                                   PRIMARY KEY (`id`),
                                   UNIQUE KEY `uk_dataset_version` (`dataset_id`,`version`),
                                   KEY `idx_dataset_version_dataset_id` (`dataset_id`),
                                   KEY `idx_dataset_version_update_time` (`update_time`)
) ENGINE=InnoDB AUTO_INCREMENT=10000 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='admin dataset version';

CREATE TABLE IF NOT EXISTS `dataset_item` (
                                `id` bigint unsigned NOT NULL AUTO_INCREMENT,
                                `dataset_id` bigint unsigned NOT NULL,
                                `columns_config` longtext DEFAULT NULL,
                                `data_content` longtext DEFAULT NULL,
                                `remark` text DEFAULT NULL,
                                `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
                                `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                                `deleted` tinyint(1) NOT NULL DEFAULT '0',
                                PRIMARY KEY (`id`),
                                KEY `idx_dataset_item_dataset_id` (`dataset_id`),
                                KEY `idx_dataset_item_deleted` (`deleted`),
                                KEY `idx_dataset_item_update_time` (`update_time`)
) ENGINE=InnoDB AUTO_INCREMENT=10000 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='admin dataset item';

CREATE TABLE IF NOT EXISTS `evaluator` (
                             `id` bigint unsigned NOT NULL AUTO_INCREMENT,
                             `name` varchar(255) NOT NULL,
                             `description` text DEFAULT NULL,
                             `latest_version` varchar(32) DEFAULT NULL,
                             `model_name` varchar(255) DEFAULT NULL,
                             `prompt` longtext DEFAULT NULL,
                             `model_config` longtext DEFAULT NULL,
                             `variables` longtext DEFAULT NULL,
                             `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
                             `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                             `deleted` tinyint(1) NOT NULL DEFAULT '0',
                             PRIMARY KEY (`id`),
                             KEY `idx_evaluator_deleted` (`deleted`),
                             KEY `idx_evaluator_update_time` (`update_time`)
) ENGINE=InnoDB AUTO_INCREMENT=10000 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='admin evaluator';

CREATE TABLE IF NOT EXISTS `evaluator_version` (
                                     `id` bigint unsigned NOT NULL AUTO_INCREMENT,
                                     `evaluator_id` bigint unsigned NOT NULL,
                                     `description` text DEFAULT NULL,
                                     `version` varchar(32) NOT NULL,
                                     `model_config` longtext DEFAULT NULL,
                                     `prompt` longtext DEFAULT NULL,
                                     `variables` longtext DEFAULT NULL,
                                     `status` varchar(32) DEFAULT NULL,
                                     `experiments` text DEFAULT NULL,
                                     `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
                                     `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                                     PRIMARY KEY (`id`),
                                     UNIQUE KEY `uk_evaluator_version` (`evaluator_id`,`version`),
                                     KEY `idx_evaluator_version_evaluator_id` (`evaluator_id`),
                                     KEY `idx_evaluator_version_update_time` (`update_time`)
) ENGINE=InnoDB AUTO_INCREMENT=10000 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='admin evaluator version';

CREATE TABLE IF NOT EXISTS `evaluator_template` (
                                      `id` bigint unsigned NOT NULL AUTO_INCREMENT,
                                      `evaluator_template_key` varchar(255) NOT NULL,
                                      `template_desc` varchar(255) DEFAULT NULL,
                                      `template` longtext,
                                      `variables` longtext DEFAULT NULL,
                                      `model_config` longtext DEFAULT NULL,
                                      `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
                                      `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                                      PRIMARY KEY (`id`),
                                      UNIQUE KEY `uk_evaluator_template_key` (`evaluator_template_key`),
                                      KEY `idx_evaluator_template_desc` (`template_desc`)
) ENGINE=InnoDB AUTO_INCREMENT=10000 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='admin evaluator template';

CREATE TABLE IF NOT EXISTS `experiment` (
                              `id` bigint unsigned NOT NULL AUTO_INCREMENT,
                              `name` varchar(255) NOT NULL,
                              `description` text DEFAULT NULL,
                              `dataset_id` bigint unsigned DEFAULT NULL,
                              `dataset_version_id` bigint unsigned DEFAULT NULL,
                              `dataset_version` varchar(32) DEFAULT NULL,
                              `evaluation_object_config` longtext DEFAULT NULL,
                              `evaluator_config` longtext DEFAULT NULL,
                              `evaluator_id` varchar(64) DEFAULT NULL,
                              `status` varchar(32) NOT NULL DEFAULT 'created',
                              `progress` int NOT NULL DEFAULT '0',
                              `complete_time` datetime DEFAULT NULL,
                              `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
                              `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                              PRIMARY KEY (`id`),
                              KEY `idx_experiment_dataset_id` (`dataset_id`),
                              KEY `idx_experiment_dataset_version_id` (`dataset_version_id`),
                              KEY `idx_experiment_status` (`status`),
                              KEY `idx_experiment_update_time` (`update_time`)
) ENGINE=InnoDB AUTO_INCREMENT=10000 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='admin experiment';

CREATE TABLE IF NOT EXISTS `experiment_result` (
                                     `id` bigint unsigned NOT NULL AUTO_INCREMENT,
                                     `experiment_id` bigint unsigned NOT NULL,
                                     `input` longtext DEFAULT NULL,
                                     `actual_output` longtext DEFAULT NULL,
                                     `reference_output` longtext DEFAULT NULL,
                                     `score` decimal(6,4) DEFAULT NULL,
                                     `reason` text DEFAULT NULL,
                                     `evaluation_time` datetime DEFAULT NULL,
                                     `evaluator_version_id` bigint unsigned DEFAULT NULL,
                                     `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
                                     `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                                     PRIMARY KEY (`id`),
                                     KEY `idx_experiment_result_experiment_id` (`experiment_id`),
                                     KEY `idx_experiment_result_evaluator_version_id` (`evaluator_version_id`),
                                     KEY `idx_experiment_result_update_time` (`update_time`)
) ENGINE=InnoDB AUTO_INCREMENT=10000 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='admin experiment result';

CREATE TABLE IF NOT EXISTS `prompt` (
                          `id` bigint unsigned NOT NULL AUTO_INCREMENT,
                          `prompt_key` varchar(255) NOT NULL,
                          `prompt_desc` varchar(255) DEFAULT NULL,
                          `prompt_description` varchar(255) DEFAULT NULL,
                          `latest_version` varchar(32) DEFAULT NULL,
                          `latest_version_status` varchar(32) DEFAULT NULL,
                          `tags` varchar(255) DEFAULT NULL,
                          `create_time` datetime(3) DEFAULT CURRENT_TIMESTAMP(3),
                          `update_time` datetime(3) DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
                          PRIMARY KEY (`id`),
                          UNIQUE KEY `uk_prompt_key` (`prompt_key`),
                          KEY `idx_prompt_update_time` (`update_time`)
) ENGINE=InnoDB AUTO_INCREMENT=10000 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='admin prompt';

CREATE TABLE IF NOT EXISTS `prompt_version` (
                                  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
                                  `version` varchar(32) NOT NULL,
                                  `prompt_key` varchar(255) NOT NULL,
                                  `version_desc` varchar(255) DEFAULT NULL,
                                  `version_description` varchar(255) DEFAULT NULL,
                                  `template` longtext,
                                  `variables` longtext DEFAULT NULL,
                                  `model_config` longtext DEFAULT NULL,
                                  `status` varchar(32) NOT NULL DEFAULT 'pre',
                                  `create_time` datetime(3) DEFAULT CURRENT_TIMESTAMP(3),
                                  `update_time` datetime(3) DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
                                  `previous_version` varchar(32) DEFAULT NULL,
                                  PRIMARY KEY (`id`),
                                  UNIQUE KEY `uk_prompt_key_version` (`prompt_key`,`version`),
                                  KEY `idx_prompt_version_prompt_key` (`prompt_key`),
                                  KEY `idx_prompt_version_status` (`status`),
                                  KEY `idx_prompt_version_update_time` (`update_time`)
) ENGINE=InnoDB AUTO_INCREMENT=10000 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='admin prompt version';

CREATE TABLE IF NOT EXISTS `prompt_build_template` (
                                         `id` bigint unsigned NOT NULL AUTO_INCREMENT,
                                         `prompt_template_key` varchar(255) NOT NULL,
                                         `tags` varchar(255) DEFAULT NULL,
                                         `template_desc` varchar(255) DEFAULT NULL,
                                         `template` longtext,
                                         `variables` longtext DEFAULT NULL,
                                         `model_config` longtext DEFAULT NULL,
                                         `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
                                         `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                                         PRIMARY KEY (`id`),
                                         UNIQUE KEY `uk_prompt_template_key` (`prompt_template_key`),
                                         KEY `idx_prompt_build_template_tags` (`tags`)
) ENGINE=InnoDB AUTO_INCREMENT=10000 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='admin prompt build template';

CREATE TABLE IF NOT EXISTS `model_config` (
                                `id` bigint unsigned NOT NULL AUTO_INCREMENT,
                                `name` varchar(100) NOT NULL,
                                `provider` varchar(50) NOT NULL,
                                `model_name` varchar(100) NOT NULL,
                                `base_url` varchar(500) NOT NULL,
                                `api_key` varchar(500) NOT NULL,
                                `default_parameters` json DEFAULT NULL,
                                `supported_parameters` json DEFAULT NULL,
                                `status` tinyint NOT NULL DEFAULT '1',
                                `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
                                `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                                `deleted` tinyint(1) NOT NULL DEFAULT '0',
                                PRIMARY KEY (`id`),
                                UNIQUE KEY `uk_model_config_name` (`name`),
                                KEY `idx_model_config_provider` (`provider`),
                                KEY `idx_model_config_status` (`status`),
                                KEY `idx_model_config_update_time` (`update_time`)
) ENGINE=InnoDB AUTO_INCREMENT=10000 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='admin model config';

INSERT IGNORE INTO `prompt_build_template` (`prompt_template_key`, `tags`, `template_desc`, `template`, `variables`, `model_config`)
VALUES
    ('conversational_ai', 'chat,dialogue', 'Conversational AI template',
     'You are a {{role}}. Traits: {{personality}}. User: {{user_input}}. Reply:',
     'role,personality,user_input',
     '{"model": "qwen-max", "temperature": 0.7, "max_tokens": 2000}'),
    ('task_executor', 'task,execution', 'Task execution template',
     'As a {{domain}} expert, complete this task: {{task_description}}. Input: {{input_data}}. Requirements: {{output_requirements}}.',
     'domain,task_description,input_data,output_requirements',
     '{"model": "qwen-max", "temperature": 0.3, "max_tokens": 3000}');

INSERT IGNORE INTO `evaluator_template` (`evaluator_template_key`, `template_desc`, `template`, `variables`, `model_config`)
VALUES
    ('text_similarity', 'Text similarity evaluator',
     'Evaluate similarity between reference output {{reference_output}} and actual output {{actual_output}}. Return a score from 0 to 1.',
     'reference_output,actual_output',
     '{"modelId": "qwen-max", "temperature": 0.1, "max_tokens": 100}'),
    ('code_quality', 'Code quality evaluator',
     'Evaluate the following code for readability, efficiency, and best practices: {{code}}.',
     'code',
     '{"modelId": "qwen-max", "temperature": 0.2, "max_tokens": 1000}');

/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2025-10-13 15:27:39
