# AIO Sandbox MCP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 OpenClaw4j 现有 MCP 页面可以注册并调用 AIO Sandbox 的 `http://<host>:8080/mcp` endpoint。

**Architecture:** 后端在现有 MCP 管理链路中新增 `STREAMABLE_HTTP` 安装类型，保持 SSE 路径不变，对 `/mcp` 使用一个小型 JDK `HttpClient` JSON-RPC 客户端分流处理 `tools/list` 和 `tools/call`。前端只在 MCP 创建页增加安装类型选项并在编辑态恢复已保存的安装类型。

**Tech Stack:** Java 26, Spring Boot 4.1, JUnit 5, AssertJ, JDK `HttpClient`, Umi/React/TypeScript, `@spark-ai/design`。

## Global Constraints

- Git 根目录固定为 `D:\IDEA_project\OpenClaw4j`。
- 后端根目录为 `OpenClaw4j-Bankend/`，使用系统 `mvn`，本地仓库固定为 `D:\apache-maven-3.9.1\m2\repository`。
- 后端 Maven 命令前设置 `$env:JAVA_HOME='D:\jdk-26'` 与 `$env:Path='D:\jdk-26\bin;' + $env:Path`。
- 前端根目录为 `OpenClaw4j-Frontend/`，主应用在 `OpenClaw4j-Frontend/packages/main`。
- 项目文档默认使用中文；代码标识、API path、配置 key 保持原始拼写。
- 保持现有 SSE MCP Server 行为不变。
- 不引入 AIO Sandbox Java SDK；将 AIO Sandbox 作为可选 profile 集成进 `deploy/docker-compose.middleware.yml`，默认不随基础中间件自动启动。
- 不修改数据库表结构。
- 不提交已有的 `OpenClaw4j-Frontend/packages/main/src/layouts/Header.tsx` 和 `OpenClaw4j-Frontend/packages/main/src/layouts/SideMenuLayout.tsx` 改动。

---

## File Structure

- Modify: `OpenClaw4j-Bankend/src/main/java/com/seaskyland/llm/workflow/runtime/enums/McpInstallTypeEnum.java`
  - 增加 `STREAMABLE_HTTP` 枚举值，供后端解析和前端提交值保存。
- Modify: `OpenClaw4j-Bankend/src/main/java/com/seaskyland/llm/workflow/core/base/manager/MCPManager.java`
  - 保留 SSE transport。
  - 新增 streamable HTTP JSON-RPC 请求、响应解析和分流方法。
- Create: `OpenClaw4j-Bankend/src/test/java/com/seaskyland/llm/workflow/core/base/manager/MCPManagerTest.java`
  - 覆盖配置解析、HTTP 工具列表、HTTP 工具调用、JSON-RPC error。
- Modify: `OpenClaw4j-Frontend/packages/main/src/pages/MCP/utils/constant.ts`
  - 增加 `STREAMABLE_HTTP` 安装类型选项和 AIO Sandbox 提示。
- Modify: `OpenClaw4j-Frontend/packages/main/src/pages/MCP/Create.tsx`
  - 编辑已有 MCP 时恢复 `install_type`。
  - 允许切换安装类型。

---

### Task 1: 后端安装类型和配置解析

**Files:**
- Modify: `OpenClaw4j-Bankend/src/main/java/com/seaskyland/llm/workflow/runtime/enums/McpInstallTypeEnum.java`
- Modify: `OpenClaw4j-Bankend/src/main/java/com/seaskyland/llm/workflow/core/base/manager/MCPManager.java`
- Test: `OpenClaw4j-Bankend/src/test/java/com/seaskyland/llm/workflow/core/base/manager/MCPManagerTest.java`

**Interfaces:**
- Consumes: existing `MCPManager.processInstallConfig(String originDeployConfig, String installType)`.
- Produces: `McpInstallTypeEnum.STREAMABLE_HTTP`.
- Produces: `processInstallConfig` accepts `/mcp` only when `installType` is `STREAMABLE_HTTP`.

- [ ] **Step 1: Write the failing configuration parsing tests**

Create `OpenClaw4j-Bankend/src/test/java/com/seaskyland/llm/workflow/core/base/manager/MCPManagerTest.java`:

```java
/*
 * Copyright 2026 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package com.seaskyland.llm.workflow.core.base.manager;

import static org.assertj.core.api.Assertions.assertThat;

import com.seaskyland.llm.workflow.runtime.domain.Result;
import com.seaskyland.llm.workflow.runtime.domain.mcp.McpServerDeployConfig;
import com.seaskyland.llm.workflow.runtime.enums.ErrorCode;
import com.seaskyland.llm.workflow.runtime.enums.McpInstallTypeEnum;
import com.seaskyland.llm.workflow.runtime.utils.JsonUtils;
import java.util.Map;
import org.junit.jupiter.api.Test;

class MCPManagerTest {

  private final MCPManager manager = new MCPManager(null);

  @Test
  void processInstallConfigAcceptsStreamableHttpMcpEndpoint() {
    String config =
        """
        {
          "mcpServers": {
            "aio-sandbox": {
              "url": "http://localhost:8080/mcp",
              "headers": {
                "Authorization": "Bearer test-token"
              }
            }
          }
        }
        """;

    Result<String> result =
        manager.processInstallConfig(config, McpInstallTypeEnum.STREAMABLE_HTTP.name());

    assertThat(result.isSuccess()).isTrue();
    McpServerDeployConfig deployConfig =
        JsonUtils.fromJson(result.getData(), McpServerDeployConfig.class);
    assertThat(deployConfig.getRemoteAddress()).isEqualTo("http://localhost:8080");
    assertThat(deployConfig.getRemoteEndpoint()).isEqualTo("/mcp");
    assertThat(deployConfig.getRemoteHeader()).containsEntry("Authorization", "Bearer test-token");
    assertThat(deployConfig.getInstallConfig()).isEqualTo(JsonUtils.toJson(JsonUtils.fromJsonToMap(config)));
  }

  @Test
  void processInstallConfigKeepsSseEndpointRequirementForSseInstallType() {
    String config =
        """
        {
          "mcpServers": {
            "aio-sandbox": {
              "url": "http://localhost:8080/mcp",
              "headers": {}
            }
          }
        }
        """;

    Result<String> result = manager.processInstallConfig(config, McpInstallTypeEnum.SSE.name());

    assertThat(result.isSuccess()).isFalse();
    assertThat(result.getCode()).isEqualTo(ErrorCode.MCP_PARSE_URL_ERROR.getCode());
  }

  @Test
  void processInstallConfigStillAcceptsSseEndpointForSseInstallType() {
    String config =
        """
        {
          "mcpServers": {
            "legacy-server": {
              "url": "http://localhost:9000/sse",
              "headers": {}
            }
          }
        }
        """;

    Result<String> result = manager.processInstallConfig(config, McpInstallTypeEnum.SSE.name());

    assertThat(result.isSuccess()).isTrue();
    McpServerDeployConfig deployConfig =
        JsonUtils.fromJson(result.getData(), McpServerDeployConfig.class);
    assertThat(deployConfig.getRemoteAddress()).isEqualTo("http://localhost:9000");
    assertThat(deployConfig.getRemoteEndpoint()).isEqualTo("/sse");
  }
}
```

- [ ] **Step 2: Run the test and verify it fails**

Run:

```powershell
cd OpenClaw4j-Bankend
$env:JAVA_HOME='D:\jdk-26'
$env:Path='D:\jdk-26\bin;' + $env:Path
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' '-Dtest=MCPManagerTest' test
```

Expected: compilation fails because `McpInstallTypeEnum.STREAMABLE_HTTP` is not defined.

- [ ] **Step 3: Add the enum value**

Modify `McpInstallTypeEnum.java`:

```java
public enum McpInstallTypeEnum {

  /** NPX installation type */
  NPX,

  /** UVX installation type */
  UVX,

  /** SSE installation type */
  SSE,

  /** Streamable HTTP installation type used by AIO Sandbox MCP Hub. */
  STREAMABLE_HTTP;
```

- [ ] **Step 4: Update config parsing**

In `MCPManager.processInstallConfig`, replace the SSE-only URL parsing block with a branch that parses URLs for both `SSE` and `STREAMABLE_HTTP`:

```java
if (installTypeEnum == McpInstallTypeEnum.SSE
    || installTypeEnum == McpInstallTypeEnum.STREAMABLE_HTTP) {
  for (String singleServer : mcpServers.keySet()) {
    Map<String, Object> singleServerConfig =
        JsonUtils.fromJsonToMap(JsonUtils.toJson(mcpServers.get(singleServer)));
    String url = (String) singleServerConfig.get("url");
    try {
      URL urlObj = new URL(url);
      if (installTypeEnum == McpInstallTypeEnum.SSE && !urlObj.getPath().endsWith("/sse")) {
        return Result.error(MCP_PARSE_URL_ERROR);
      }
      if (installTypeEnum == McpInstallTypeEnum.STREAMABLE_HTTP
          && StringUtils.isBlank(urlObj.getPath())) {
        return Result.error(MCP_PARSE_URL_ERROR);
      }

      String remoteAddress = urlObj.getProtocol() + "://" + urlObj.getHost();
      if (urlObj.getPort() != -1) {
        remoteAddress += ":" + urlObj.getPort();
      }
      targetDeployConfig.put("remote_address", remoteAddress);
      targetDeployConfig.put("remote_endpoint", urlObj.getPath());
      String query = urlObj.getQuery();
      if (StringUtils.isNotBlank(query)) {
        targetDeployConfig.put("remote_endpoint", urlObj.getPath() + "?" + query);
      }
      Map<Object, Object> headers =
          JsonUtils.fromJsonToMap(JsonUtils.toJson(singleServerConfig.get("headers")));
      targetDeployConfig.put("remote_header", headers);
    } catch (Exception urlCheckEx) {
      LogUtils.error("processInstallConfig", url, urlCheckEx);
      return Result.error(MCP_PARSE_URL_ERROR);
    }
    break;
  }
}
```

- [ ] **Step 5: Run the focused test and verify it passes**

Run:

```powershell
cd OpenClaw4j-Bankend
$env:JAVA_HOME='D:\jdk-26'
$env:Path='D:\jdk-26\bin;' + $env:Path
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' '-Dtest=MCPManagerTest' test
```

Expected: `BUILD SUCCESS`.

- [ ] **Step 6: Commit task 1**

```powershell
git add OpenClaw4j-Bankend/src/main/java/com/seaskyland/llm/workflow/runtime/enums/McpInstallTypeEnum.java OpenClaw4j-Bankend/src/main/java/com/seaskyland/llm/workflow/core/base/manager/MCPManager.java OpenClaw4j-Bankend/src/test/java/com/seaskyland/llm/workflow/core/base/manager/MCPManagerTest.java
git commit -m "feat: accept streamable http mcp endpoints"
```

---

### Task 2: Streamable HTTP JSON-RPC 工具调用

**Files:**
- Modify: `OpenClaw4j-Bankend/src/main/java/com/seaskyland/llm/workflow/core/base/manager/MCPManager.java`
- Test: `OpenClaw4j-Bankend/src/test/java/com/seaskyland/llm/workflow/core/base/manager/MCPManagerTest.java`

**Interfaces:**
- Consumes: `McpInstallTypeEnum.STREAMABLE_HTTP`.
- Produces: `MCPManager.getTools(McpServerEntity entity)` can call JSON-RPC `tools/list` for streamable HTTP.
- Produces: `MCPManager.callTool(McpServerCallToolRequest request, McpServerEntity entity)` can call JSON-RPC `tools/call` for streamable HTTP.

- [ ] **Step 1: Add failing HTTP tests**

Append these tests and helpers to `MCPManagerTest.java`:

```java
@Test
void getToolsReadsToolsFromStreamableHttpEndpoint() throws Exception {
  try (TestMcpHttpServer server =
      TestMcpHttpServer.start(
          """
          {
            "jsonrpc": "2.0",
            "id": 1,
            "result": {
              "tools": [
                {
                  "name": "file_read",
                  "description": "Read a file",
                  "inputSchema": {
                    "type": "object",
                    "properties": {
                      "path": {
                        "type": "string"
                      }
                    },
                    "required": ["path"],
                    "additionalProperties": false
                  }
                }
              ]
            }
          }
          """)) {
    McpServerEntity entity = streamableHttpEntity(server.url("/mcp"));

    List<McpTool> tools = manager.getTools(entity);

    assertThat(server.lastRequestBody()).contains("\"method\":\"tools/list\"");
    assertThat(server.lastRequestHeaders()).containsEntry("Authorization", "Bearer test-token");
    assertThat(tools).hasSize(1);
    assertThat(tools.get(0).getName()).isEqualTo("file_read");
    assertThat(tools.get(0).getDescription()).isEqualTo("Read a file");
    assertThat(tools.get(0).getInputSchema().getType()).isEqualTo("object");
    assertThat(tools.get(0).getInputSchema().getRequired()).containsExactly("path");
    assertThat(tools.get(0).getInputSchema().getAdditionalProperties()).isFalse();
  }
}

@Test
void callToolReadsTextContentFromStreamableHttpEndpoint() throws Exception {
  try (TestMcpHttpServer server =
      TestMcpHttpServer.start(
          """
          {
            "jsonrpc": "2.0",
            "id": 1,
            "result": {
              "isError": false,
              "content": [
                {
                  "type": "text",
                  "text": "hello from aio"
                }
              ]
            }
          }
          """)) {
    McpServerEntity entity = streamableHttpEntity(server.url("/mcp"));
    McpServerCallToolRequest request = new McpServerCallToolRequest();
    request.setToolName("file_read");
    request.setToolParams(Map.of("path", "/home/gem/readme.txt"));

    McpServerCallToolResponse response = manager.callTool(request, entity);

    assertThat(server.lastRequestBody()).contains("\"method\":\"tools/call\"");
    assertThat(server.lastRequestBody()).contains("\"name\":\"file_read\"");
    assertThat(response.getIsError()).isFalse();
    assertThat(response.getContent()).hasSize(1);
    assertThat((TextContent) response.getContent().get(0))
        .extracting(TextContent::getText)
        .isEqualTo("hello from aio");
  }
}

@Test
void callToolConvertsJsonRpcErrorToTextContent() throws Exception {
  try (TestMcpHttpServer server =
      TestMcpHttpServer.start(
          """
          {
            "jsonrpc": "2.0",
            "id": 1,
            "error": {
              "code": -32602,
              "message": "invalid params"
            }
          }
          """)) {
    McpServerEntity entity = streamableHttpEntity(server.url("/mcp"));
    McpServerCallToolRequest request = new McpServerCallToolRequest();
    request.setToolName("file_read");
    request.setToolParams(Map.of("path", "/missing"));

    McpServerCallToolResponse response = manager.callTool(request, entity);

    assertThat(response.getIsError()).isTrue();
    assertThat(response.getContent()).hasSize(1);
    assertThat((TextContent) response.getContent().get(0))
        .extracting(TextContent::getText)
        .isEqualTo("invalid params");
  }
}

private McpServerEntity streamableHttpEntity(String url) {
  Result<String> config =
      manager.processInstallConfig(
          """
          {
            "mcpServers": {
              "aio-sandbox": {
                "url": "%s",
                "headers": {
                  "Authorization": "Bearer test-token"
                }
              }
            }
          }
          """
              .formatted(url),
          McpInstallTypeEnum.STREAMABLE_HTTP.name());
  McpServerEntity entity = new McpServerEntity();
  entity.setServerCode("aio-sandbox");
  entity.setStatus(McpServerStatusEnum.Normal.getCode());
  entity.setInstallType(McpInstallTypeEnum.STREAMABLE_HTTP.name());
  entity.setDeployConfig(config.getData());
  entity.setHost(JsonUtils.fromJson(config.getData(), McpServerDeployConfig.class).getRemoteAddress());
  return entity;
}
```

Add these imports to `MCPManagerTest.java`:

```java
import com.seaskyland.llm.workflow.core.base.entity.McpServerEntity;
import com.seaskyland.llm.workflow.runtime.domain.mcp.McpServerCallToolRequest;
import com.seaskyland.llm.workflow.runtime.domain.mcp.McpServerCallToolResponse;
import com.seaskyland.llm.workflow.runtime.domain.mcp.McpTool;
import com.seaskyland.llm.workflow.runtime.domain.mcp.TextContent;
import com.seaskyland.llm.workflow.runtime.enums.McpServerStatusEnum;
import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpServer;
import java.io.IOException;
import java.net.InetSocketAddress;
import java.nio.charset.StandardCharsets;
import java.util.List;
import java.util.concurrent.Executors;
```

Add the test server helper as a nested static class:

```java
private static final class TestMcpHttpServer implements AutoCloseable {

  private final HttpServer server;
  private volatile String lastRequestBody;
  private volatile Map<String, String> lastRequestHeaders = Map.of();

  private TestMcpHttpServer(HttpServer server) {
    this.server = server;
  }

  static TestMcpHttpServer start(String responseBody) throws IOException {
    HttpServer httpServer = HttpServer.create(new InetSocketAddress("127.0.0.1", 0), 0);
    TestMcpHttpServer wrapper = new TestMcpHttpServer(httpServer);
    httpServer.createContext(
        "/mcp",
        exchange -> {
          wrapper.capture(exchange);
          byte[] response = responseBody.getBytes(StandardCharsets.UTF_8);
          exchange.getResponseHeaders().add("Content-Type", "application/json");
          exchange.sendResponseHeaders(200, response.length);
          exchange.getResponseBody().write(response);
          exchange.close();
        });
    httpServer.setExecutor(Executors.newSingleThreadExecutor());
    httpServer.start();
    return wrapper;
  }

  String url(String path) {
    return "http://127.0.0.1:" + server.getAddress().getPort() + path;
  }

  String lastRequestBody() {
    return lastRequestBody;
  }

  Map<String, String> lastRequestHeaders() {
    return lastRequestHeaders;
  }

  private void capture(HttpExchange exchange) throws IOException {
    lastRequestBody = new String(exchange.getRequestBody().readAllBytes(), StandardCharsets.UTF_8);
    lastRequestHeaders =
        exchange.getRequestHeaders().entrySet().stream()
            .collect(
                java.util.stream.Collectors.toMap(
                    Map.Entry::getKey, entry -> String.join(",", entry.getValue())));
  }

  @Override
  public void close() {
    server.stop(0);
  }
}
```

- [ ] **Step 2: Run the test and verify it fails**

Run:

```powershell
cd OpenClaw4j-Bankend
$env:JAVA_HOME='D:\jdk-26'
$env:Path='D:\jdk-26\bin;' + $env:Path
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' '-Dtest=MCPManagerTest' test
```

Expected: tests fail because `getTools` and `callTool` still try SSE transport for `STREAMABLE_HTTP`.

- [ ] **Step 3: Add streamable HTTP branch and helpers**

In `MCPManager.java`, add imports:

```java
import java.io.IOException;
import java.net.URI;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.util.UUID;
```

Add branch logic at the start of `getTools` after the status check:

```java
if (McpInstallTypeEnum.STREAMABLE_HTTP.name().equals(entity.getInstallType())) {
  return getStreamableHttpTools(entity);
}
```

Add branch logic at the start of `callTool`:

```java
if (McpInstallTypeEnum.STREAMABLE_HTTP.name().equals(entity.getInstallType())) {
  return callStreamableHttpTool(request, entity);
}
```

Add these private methods to `MCPManager`:

```java
private List<McpTool> getStreamableHttpTools(McpServerEntity entity) {
  try {
    Map<String, Object> response =
        sendStreamableHttpRequest(entity, "tools/list", Map.of());
    Map<String, Object> error = toObjectMap(response.get("error"));
    if (error != null) {
      LogUtils.error("streamable http tools/list error", entity.getServerCode(), error);
      return new ArrayList<>();
    }
    Map<String, Object> result = toObjectMap(response.get("result"));
    List<Object> toolsList = toRawList(result == null ? null : result.get("tools"));
    List<McpTool> tools = new ArrayList<>();
    if (CollectionUtils.isEmpty(toolsList)) {
      return tools;
    }
    for (Object rawTool : toolsList) {
      Map<String, Object> toolMap = toObjectMap(rawTool);
      if (toolMap == null) {
        continue;
      }
      McpTool tool = new McpTool();
      tool.setName((String) toolMap.get("name"));
      tool.setDescription((String) toolMap.get("description"));
      Map<String, Object> inputSchemaMap =
          toObjectMap(
              toolMap.containsKey("inputSchema")
                  ? toolMap.get("inputSchema")
                  : toolMap.get("input_schema"));
      tool.setInputSchema(toInputSchema(inputSchemaMap));
      tools.add(tool);
    }
    return tools;
  } catch (Exception ex) {
    LogUtils.error("getStreamableHttpTools error", ex, entity.getServerCode(), entity.getHost());
    return new ArrayList<>();
  }
}

private McpServerCallToolResponse callStreamableHttpTool(
    McpServerCallToolRequest request, McpServerEntity entity) {
  Map<String, Object> params =
      Map.of(
          "name", request.getToolName(),
          "arguments", request.getToolParams() == null ? Map.of() : request.getToolParams());
  Map<String, Object> response = sendStreamableHttpRequest(entity, "tools/call", params);
  McpServerCallToolResponse toolResponse = new McpServerCallToolResponse();
  Map<String, Object> error = toObjectMap(response.get("error"));
  if (error != null) {
    toolResponse.setIsError(true);
    toolResponse.setContent(List.of(textContent(String.valueOf(error.get("message")))));
    return toolResponse;
  }
  Map<String, Object> result = toObjectMap(response.get("result"));
  if (result == null) {
    toolResponse.setIsError(true);
    toolResponse.setContent(List.of(textContent("Empty MCP response result.")));
    return toolResponse;
  }
  Object isError = result.containsKey("isError") ? result.get("isError") : result.get("is_error");
  toolResponse.setIsError(Boolean.TRUE.equals(isError));
  toolResponse.setContent(toContentList(result.get("content")));
  return toolResponse;
}

private Map<String, Object> sendStreamableHttpRequest(
    McpServerEntity entity, String method, Map<String, Object> params) {
  try {
    McpServerDeployConfig deployConfig =
        JsonUtils.fromJson(entity.getDeployConfig(), McpServerDeployConfig.class);
    String endpoint =
        StringUtils.defaultIfBlank(deployConfig.getRemoteEndpoint(), "/mcp");
    URI uri = URI.create(entity.getHost() + endpoint);
    Map<String, Object> body =
        new HashMap<>(
            Map.of(
                "jsonrpc", "2.0",
                "id", UUID.randomUUID().toString(),
                "method", method,
                "params", params));
    HttpRequest.Builder requestBuilder =
        HttpRequest.newBuilder(uri)
            .timeout(Duration.ofSeconds(60))
            .header("Content-Type", "application/json")
            .POST(HttpRequest.BodyPublishers.ofString(JsonUtils.toJson(body)));
    if (deployConfig.getRemoteHeader() != null) {
      deployConfig.getRemoteHeader().forEach(requestBuilder::header);
    }
    HttpResponse<String> response =
        HttpClient.newHttpClient().send(requestBuilder.build(), HttpResponse.BodyHandlers.ofString());
    if (response.statusCode() < 200 || response.statusCode() >= 300) {
      throw new IllegalStateException("MCP HTTP status " + response.statusCode());
    }
    return JsonUtils.fromJsonToMap(response.body());
  } catch (IOException ex) {
    throw new IllegalStateException("Failed to call streamable HTTP MCP endpoint", ex);
  } catch (InterruptedException ex) {
    Thread.currentThread().interrupt();
    throw new IllegalStateException("Interrupted while calling streamable HTTP MCP endpoint", ex);
  }
}

private InputSchema toInputSchema(Map<String, Object> inputSchemaMap) {
  InputSchema inputSchema = new InputSchema();
  if (inputSchemaMap == null) {
    inputSchema.setType("object");
    inputSchema.setProperties(Map.of());
    inputSchema.setRequired(List.of());
    inputSchema.setAdditionalProperties(false);
    return inputSchema;
  }
  inputSchema.setType((String) inputSchemaMap.get("type"));
  inputSchema.setProperties(toObjectMap(inputSchemaMap.get("properties")));
  inputSchema.setRequired(toStringList(inputSchemaMap.get("required")));
  Object additionalProperties =
      inputSchemaMap.containsKey("additionalProperties")
          ? inputSchemaMap.get("additionalProperties")
          : inputSchemaMap.get("additional_properties");
  inputSchema.setAdditionalProperties(Boolean.TRUE.equals(additionalProperties));
  return inputSchema;
}

private List<Content> toContentList(Object rawContent) {
  List<Object> rawList = toRawList(rawContent);
  if (rawList == null) {
    return new ArrayList<>();
  }
  List<Content> content = new ArrayList<>();
  for (Object rawItem : rawList) {
    Map<String, Object> item = toObjectMap(rawItem);
    if (item == null) {
      continue;
    }
    if ("text".equals(item.get("type"))) {
      content.add(textContent(String.valueOf(item.get("text"))));
    }
  }
  return content;
}

private TextContent textContent(String text) {
  TextContent textContent = new TextContent();
  textContent.setType("text");
  textContent.setText(text);
  return textContent;
}

@SuppressWarnings("unchecked")
private List<Object> toRawList(Object value) {
  return value instanceof List<?> ? (List<Object>) value : null;
}
```

- [ ] **Step 4: Run focused tests**

Run:

```powershell
cd OpenClaw4j-Bankend
$env:JAVA_HOME='D:\jdk-26'
$env:Path='D:\jdk-26\bin;' + $env:Path
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' '-Dtest=MCPManagerTest' test
```

Expected: `BUILD SUCCESS`.

- [ ] **Step 5: Commit task 2**

```powershell
git add OpenClaw4j-Bankend/src/main/java/com/seaskyland/llm/workflow/core/base/manager/MCPManager.java OpenClaw4j-Bankend/src/test/java/com/seaskyland/llm/workflow/core/base/manager/MCPManagerTest.java
git commit -m "feat: call streamable http mcp tools"
```

---

### Task 3: 前端 MCP 创建页支持 AIO 安装类型

**Files:**
- Modify: `OpenClaw4j-Frontend/packages/main/src/pages/MCP/utils/constant.ts`
- Modify: `OpenClaw4j-Frontend/packages/main/src/pages/MCP/Create.tsx`

**Interfaces:**
- Consumes: backend accepts `install_type: "STREAMABLE_HTTP"`.
- Produces: UI option value `STREAMABLE_HTTP`.
- Produces: edit page restores `mcpData.install_type`.

- [ ] **Step 1: Add the frontend option**

Modify `installTypeOptions` in `constant.ts`:

```ts
export const installTypeOptions: IRadioItemProps[] = [
  {
    label: 'SSE',
    value: 'SSE',
    logo: 'spark-internet-line',
  },
  {
    label: 'Streamable HTTP / AIO Sandbox',
    value: 'STREAMABLE_HTTP',
    logo: 'spark-internet-line',
  },
];
```

Add an AIO Sandbox tip section at the beginning of `MCP_TIP_SECTIONS`:

```ts
{
  title: 'AIO Sandbox',
  linkButtons: [
    {
      text: 'AIO Sandbox',
      url: 'https://sandbox.agent-infra.com/zh/guide/start/quick-start',
    },
  ],
  description:
    'AIO Sandbox 的 MCP Hub 默认入口为 http://localhost:8080/mcp。请选择 Streamable HTTP / AIO Sandbox，并使用 mcpServers JSON 配置该地址。',
},
```

- [ ] **Step 2: Restore install type on edit**

In `Create.tsx`, inside the `getMcpServer` success block that handles `if (res && res.data)`, after `setInitialData(mcpData);`, add:

```ts
setInstallType(mcpData.install_type || installTypeOptions[0].value);
```

- [ ] **Step 3: Allow users to select the new option**

In `Create.tsx`, change the `RadioItem` disabled prop:

```tsx
disabled={!!server_code && deployStatus === McpStatus.ENABLED}
```

This keeps running MCP services read-only while allowing new MCP creation to choose `STREAMABLE_HTTP`.

- [ ] **Step 4: Run frontend build**

Run:

```powershell
cd OpenClaw4j-Frontend
npm run build:app
```

Expected: output includes `Webpack: Compiled successfully`.

- [ ] **Step 5: Commit task 3**

```powershell
git add OpenClaw4j-Frontend/packages/main/src/pages/MCP/utils/constant.ts OpenClaw4j-Frontend/packages/main/src/pages/MCP/Create.tsx
git commit -m "feat: add aio sandbox mcp install option"
```

---

### Task 4: Final Verification

**Files:**
- Verify: all files changed by Tasks 1-3.

**Interfaces:**
- Consumes: completed backend and frontend changes.
- Produces: evidence for final status.

- [ ] **Step 1: Check working tree scope**

Run:

```powershell
git status --short
git diff --stat
```

Expected: only expected files are modified, plus the pre-existing layout files may remain modified but unstaged.

- [ ] **Step 2: Run backend focused tests**

Run:

```powershell
cd OpenClaw4j-Bankend
$env:JAVA_HOME='D:\jdk-26'
$env:Path='D:\jdk-26\bin;' + $env:Path
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' '-Dtest=MCPManagerTest' test
```

Expected: `BUILD SUCCESS`.

- [ ] **Step 3: Run backend quality gate**

Run:

```powershell
cd OpenClaw4j-Bankend
$env:JAVA_HOME='D:\jdk-26'
$env:Path='D:\jdk-26\bin;' + $env:Path
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' spotless:check
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' checkstyle:check
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' spotbugs:check
```

Expected: each command exits `0`. If a command fails due to existing baseline issues, record the exact command, exit code, and first relevant failures.

- [ ] **Step 4: Run frontend build**

Run:

```powershell
cd OpenClaw4j-Frontend
npm run build:app
```

Expected: output includes `Webpack: Compiled successfully`.

- [ ] **Step 5: Run whitespace check**

Run:

```powershell
git diff --check
```

Expected: no output and exit code `0`.

- [ ] **Step 6: Final commit if verification required formatting fixes**

If formatting fixes were required after previous task commits, run:

```powershell
git add OpenClaw4j-Bankend/src/main/java/com/seaskyland/llm/workflow/runtime/enums/McpInstallTypeEnum.java OpenClaw4j-Bankend/src/main/java/com/seaskyland/llm/workflow/core/base/manager/MCPManager.java OpenClaw4j-Bankend/src/test/java/com/seaskyland/llm/workflow/core/base/manager/MCPManagerTest.java OpenClaw4j-Frontend/packages/main/src/pages/MCP/utils/constant.ts OpenClaw4j-Frontend/packages/main/src/pages/MCP/Create.tsx
git commit -m "chore: polish aio sandbox mcp integration"
```

If no formatting fixes were required, skip this step and report the verification evidence.
