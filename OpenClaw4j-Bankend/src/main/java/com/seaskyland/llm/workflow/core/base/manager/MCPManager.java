/*
 * Copyright 2025 the original author or authors.
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

import static com.seaskyland.llm.workflow.core.utils.LogUtils.FAIL;
import static com.seaskyland.llm.workflow.core.utils.LogUtils.SUCCESS;
import static com.seaskyland.llm.workflow.runtime.enums.ErrorCode.MCP_PARSE_CONFIG_ERROR;
import static com.seaskyland.llm.workflow.runtime.enums.ErrorCode.MCP_PARSE_URL_ERROR;

import com.seaskyland.llm.workflow.core.base.constants.CacheConstants;
import com.seaskyland.llm.workflow.core.base.entity.McpServerEntity;
import com.seaskyland.llm.workflow.core.utils.LogUtils;
import com.seaskyland.llm.workflow.core.utils.concurrent.ThreadPoolUtils;
import com.seaskyland.llm.workflow.runtime.domain.Result;
import com.seaskyland.llm.workflow.runtime.domain.mcp.Content;
import com.seaskyland.llm.workflow.runtime.domain.mcp.McpServerCallToolRequest;
import com.seaskyland.llm.workflow.runtime.domain.mcp.McpServerCallToolResponse;
import com.seaskyland.llm.workflow.runtime.domain.mcp.McpServerDeployConfig;
import com.seaskyland.llm.workflow.runtime.domain.mcp.McpTool;
import com.seaskyland.llm.workflow.runtime.domain.mcp.TextContent;
import com.seaskyland.llm.workflow.runtime.domain.tool.InputSchema;
import com.seaskyland.llm.workflow.runtime.enums.McpInstallTypeEnum;
import com.seaskyland.llm.workflow.runtime.enums.McpServerStatusEnum;
import com.seaskyland.llm.workflow.runtime.utils.JsonUtils;
import io.modelcontextprotocol.client.McpClient;
import io.modelcontextprotocol.client.McpSyncClient;
import io.modelcontextprotocol.client.transport.HttpClientSseClientTransport;
import io.modelcontextprotocol.spec.McpClientTransport;
import io.modelcontextprotocol.spec.McpSchema;
import java.net.URI;
import java.net.URL;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.Callable;
import java.util.concurrent.Future;
import java.util.concurrent.TimeUnit;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.stereotype.Component;

/**
 * Manager class for handling Model Context Protocol (MCP) operations. Provides functionality for
 * client creation, tool management, and server configuration.
 */
@Slf4j
@Component
public class MCPManager {

  /** Redis manager for caching operations */
  private final CacheManager cacheManager;

  public MCPManager(CacheManager cacheManager) {
    this.cacheManager = cacheManager;
  }

  /**
   * Creates a synchronous MCP client for the given server entity.
   *
   * @param entity Server entity containing deployment configuration
   * @return Synchronous MCP client instance
   */
  public McpSyncClient getMcpSyncClient(McpServerEntity entity) {
    McpServerDeployConfig deployConfig =
        JsonUtils.fromJson(entity.getDeployConfig(), McpServerDeployConfig.class);

    String remoteEndpoint = deployConfig.getRemoteEndpoint();
    if (StringUtils.isBlank(remoteEndpoint)) {
      remoteEndpoint = "/sse";
    }
    String host = entity.getHost();
    LogUtils.error("Client host", host, entity.getServerCode());
    HttpClient.Builder builder1 =
        HttpClient.newBuilder()
            .version(HttpClient.Version.HTTP_1_1)
            .connectTimeout(Duration.ofSeconds(60));
    McpClientTransport transport =
        HttpClientSseClientTransport.builder(host)
            .clientBuilder(builder1)
            .sseEndpoint(remoteEndpoint)
            .build();
    McpClient.SyncSpec builder =
        McpClient.sync(transport)
            .requestTimeout(Duration.ofSeconds(60))
            .initializationTimeout(Duration.ofSeconds(60))
            .capabilities(McpSchema.ClientCapabilities.builder().roots(true).build());
    return builder.build();
  }

  /**
   * Retrieves available tools from the specified MCP server. Implements caching mechanism for
   * non-SSE installations.
   *
   * @param entity Server entity to query tools from
   * @return List of available tools
   */
  public List<McpTool> getTools(McpServerEntity entity) {
    long start = System.currentTimeMillis();
    if (entity == null || !entity.getStatus().equals(McpServerStatusEnum.Normal.getCode())) {
      return new ArrayList<>();
    }
    String cacheKey = "mcp_tools_cache_" + entity.getServerCode();
    boolean needCache = !"SSE".equals(entity.getInstallType());
    if (needCache && cacheManager != null) {
      List<McpTool> toolList = cacheManager.get(cacheKey);
      if (toolList != null) {
        return toolList;
      }
    }
    Future<Object> future =
        ThreadPoolUtils.TOOL_TASK_EXECUTOR.submit(
            new Callable<Object>() {
              @Override
              public Object call() throws Exception {
                if (isStreamableHttp(entity)) {
                  return getStreamableHttpTools(entity);
                }
                McpSyncClient client = getMcpSyncClient(entity);
                try {
                  client.initialize();
                  McpSchema.ListToolsResult toolsResult = client.listTools();
                  List<McpSchema.Tool> toolsList = toolsResult.tools();
                  List<McpTool> tools = new ArrayList<>();
                  if (CollectionUtils.isEmpty(toolsList)) {
                    return tools;
                  }
                  toolsList.forEach(
                      e -> {
                        McpTool tool = new McpTool();
                        tool.setName(e.name());
                        tool.setDescription(e.description());
                        Map<String, Object> jsonSchema = e.inputSchema();
                        InputSchema inputSchema = new InputSchema();
                        inputSchema.setType((String) jsonSchema.get("type"));
                        inputSchema.setProperties(toObjectMap(jsonSchema.get("properties")));
                        inputSchema.setRequired(toStringList(jsonSchema.get("required")));
                        inputSchema.setAdditionalProperties(
                            (Boolean) jsonSchema.get("additionalProperties"));
                        tool.setInputSchema(inputSchema);
                        tools.add(tool);
                      });
                  LogUtils.error(
                      "FromServer GetTool",
                      entity.getServerCode(),
                      System.currentTimeMillis() - start,
                      tools.size());
                  return tools;
                } catch (Exception ex) {
                  LogUtils.error("getTools error", ex, entity.getServerCode(), entity.getHost());
                } finally {
                  if (client != null) {
                    client.close();
                  }
                }
                return new ArrayList<>();
              }
            });
    try {
      List<McpTool> result = (List<McpTool>) future.get(3, TimeUnit.SECONDS);
      if (result != null && !result.isEmpty() && needCache && cacheManager != null) {
        cacheManager.put(cacheKey, result, CacheConstants.CACHE_EMPTY_TTL);
      }
      LogUtils.error(
          "FromServer Last", entity.getServerCode(), System.currentTimeMillis() - start, result);
      return result;
    } catch (Exception ex) {
      LogUtils.error("toolsGetFuture Exception", ex, entity.getServerCode());
      return new ArrayList<>();
    }
  }

  @SuppressWarnings("unchecked")
  private Map<String, Object> toObjectMap(Object value) {
    return value instanceof Map<?, ?> ? (Map<String, Object>) value : null;
  }

  @SuppressWarnings("unchecked")
  private List<String> toStringList(Object value) {
    return value instanceof List<?> ? (List<String>) value : null;
  }

  /**
   * Executes a specific tool on the MCP server.
   *
   * @param request Tool execution request parameters
   * @param entity Server entity where the tool resides
   * @return Response from the tool execution
   */
  public McpServerCallToolResponse callTool(
      McpServerCallToolRequest request, McpServerEntity entity) {
    Long start = System.currentTimeMillis();
    if (isStreamableHttp(entity)) {
      return callStreamableHttpTool(request, entity, start);
    }
    McpSyncClient client = getMcpSyncClient(entity);
    McpServerCallToolResponse response = new McpServerCallToolResponse();
    try {
      client.initialize();
      McpSchema.CallToolRequest callToolRequest =
          new McpSchema.CallToolRequest(request.getToolName(), request.getToolParams());
      McpSchema.CallToolResult callToolResult = client.callTool(callToolRequest);
      response.setIsError(callToolResult.isError());
      List<McpSchema.Content> contentList = callToolResult.content();
      List<Content> content = new ArrayList<>();
      contentList.forEach(
          e -> {
            if (e.type().equals("text")) {
              TextContent textContent = new TextContent();
              textContent.setType(e.type());
              McpSchema.TextContent tContent = (McpSchema.TextContent) e;
              textContent.setText(tContent.text());
              content.add(textContent);
            }
          });
      response.setContent(content);
    } catch (Exception ex) {
      LogUtils.monitor("McpService", "callTool", start, FAIL, request, ex.getMessage(), ex);
      LogUtils.error("McpServerManager callTool exception", ex);
      throw ex;
    } finally {
      if (client != null) {
        client.close();
      }
    }
    LogUtils.monitor("McpService", "callTool", start, SUCCESS, request, response);
    return response;
  }

  private boolean isStreamableHttp(McpServerEntity entity) {
    return entity != null
        && McpInstallTypeEnum.STREAMABLE_HTTP.name().equals(entity.getInstallType());
  }

  private List<McpTool> getStreamableHttpTools(McpServerEntity entity) {
    try {
      Map<String, Object> response =
          sendStreamableHttpRequest(entity, "tools/list", new HashMap<>());
      if (response.containsKey("error")) {
        LogUtils.error("streamableHttpToolsError", entity.getServerCode(), response.get("error"));
        return new ArrayList<>();
      }
      Map<String, Object> result = toObjectMap(response.get("result"));
      List<Object> toolValues = result == null ? null : toRawList(result.get("tools"));
      List<McpTool> tools = new ArrayList<>();
      if (toolValues == null) {
        return tools;
      }
      toolValues.forEach(
          toolValue -> {
            Map<String, Object> toolMap = toObjectMap(toolValue);
            if (toolMap == null) {
              return;
            }
            McpTool tool = new McpTool();
            tool.setName((String) toolMap.get("name"));
            tool.setDescription((String) toolMap.get("description"));
            Object schema = toolMap.get("inputSchema");
            if (schema == null) {
              schema = toolMap.get("input_schema");
            }
            tool.setInputSchema(toInputSchema(schema));
            tools.add(tool);
          });
      return tools;
    } catch (Exception ex) {
      LogUtils.error("getStreamableHttpTools error", ex, entity.getServerCode(), entity.getHost());
      return new ArrayList<>();
    }
  }

  private McpServerCallToolResponse callStreamableHttpTool(
      McpServerCallToolRequest request, McpServerEntity entity, Long start) {
    try {
      HashMap<String, Object> params = new HashMap<>();
      params.put("name", request.getToolName());
      params.put(
          "arguments", request.getToolParams() == null ? new HashMap<>() : request.getToolParams());
      Map<String, Object> jsonRpcResponse = sendStreamableHttpRequest(entity, "tools/call", params);
      McpServerCallToolResponse response = toCallToolResponse(jsonRpcResponse);
      LogUtils.monitor("McpService", "callTool", start, SUCCESS, request, response);
      return response;
    } catch (Exception ex) {
      LogUtils.monitor("McpService", "callTool", start, FAIL, request, ex.getMessage(), ex);
      LogUtils.error("McpServerManager callStreamableHttpTool exception", ex);
      throw ex;
    }
  }

  private Map<String, Object> sendStreamableHttpRequest(
      McpServerEntity entity, String method, Map<String, Object> params) {
    try {
      McpServerDeployConfig deployConfig =
          JsonUtils.fromJson(entity.getDeployConfig(), McpServerDeployConfig.class);
      String remoteEndpoint = StringUtils.defaultIfBlank(deployConfig.getRemoteEndpoint(), "/mcp");
      HttpRequest.Builder requestBuilder =
          HttpRequest.newBuilder()
              .uri(URI.create(entity.getHost() + remoteEndpoint))
              .timeout(Duration.ofSeconds(60))
              .header("Content-Type", "application/json")
              .header("Accept", "application/json");
      if (deployConfig.getRemoteHeader() != null) {
        deployConfig
            .getRemoteHeader()
            .forEach(
                (name, value) -> {
                  if (StringUtils.isNotBlank(name) && value != null) {
                    requestBuilder.header(name, value);
                  }
                });
      }

      HashMap<String, Object> payload = new HashMap<>();
      payload.put("jsonrpc", "2.0");
      payload.put("id", System.currentTimeMillis());
      payload.put("method", method);
      payload.put("params", params == null ? new HashMap<>() : params);

      HttpRequest httpRequest =
          requestBuilder
              .POST(HttpRequest.BodyPublishers.ofString(JsonUtils.toJson(payload)))
              .build();
      HttpResponse<String> response =
          HttpClient.newBuilder()
              .version(HttpClient.Version.HTTP_1_1)
              .connectTimeout(Duration.ofSeconds(60))
              .build()
              .send(httpRequest, HttpResponse.BodyHandlers.ofString());
      if (response.statusCode() < 200 || response.statusCode() >= 300) {
        throw new IllegalStateException(
            "Streamable HTTP MCP request failed with status " + response.statusCode());
      }
      return JsonUtils.fromJsonToMap(response.body());
    } catch (InterruptedException ex) {
      Thread.currentThread().interrupt();
      throw new IllegalStateException("Streamable HTTP MCP request interrupted", ex);
    } catch (Exception ex) {
      throw new IllegalStateException("Streamable HTTP MCP request failed", ex);
    }
  }

  private McpServerCallToolResponse toCallToolResponse(Map<String, Object> jsonRpcResponse) {
    McpServerCallToolResponse response = new McpServerCallToolResponse();
    Map<String, Object> error = toObjectMap(jsonRpcResponse.get("error"));
    if (error != null) {
      response.setIsError(true);
      response.setContent(List.of(textContent(String.valueOf(error.get("message")))));
      return response;
    }
    Map<String, Object> result = toObjectMap(jsonRpcResponse.get("result"));
    if (result == null) {
      response.setIsError(true);
      response.setContent(List.of(textContent("Empty MCP response")));
      return response;
    }
    Object isError = result.get("isError");
    if (isError == null) {
      isError = result.get("is_error");
    }
    response.setIsError(isError instanceof Boolean ? (Boolean) isError : false);
    response.setContent(toContentList(result.get("content")));
    return response;
  }

  private InputSchema toInputSchema(Object schema) {
    Map<String, Object> schemaMap = toObjectMap(schema);
    if (schemaMap == null) {
      return null;
    }
    InputSchema inputSchema = new InputSchema();
    inputSchema.setType((String) schemaMap.get("type"));
    inputSchema.setProperties(toObjectMap(schemaMap.get("properties")));
    inputSchema.setRequired(toStringList(schemaMap.get("required")));
    Object additionalProperties = schemaMap.get("additionalProperties");
    if (additionalProperties == null) {
      additionalProperties = schemaMap.get("additional_properties");
    }
    inputSchema.setAdditionalProperties(
        additionalProperties instanceof Boolean ? (Boolean) additionalProperties : null);
    return inputSchema;
  }

  @SuppressWarnings("unchecked")
  private List<Object> toRawList(Object value) {
    return value instanceof List<?> ? (List<Object>) value : null;
  }

  private List<Content> toContentList(Object value) {
    List<Object> values = toRawList(value);
    List<Content> contents = new ArrayList<>();
    if (values == null) {
      return contents;
    }
    values.forEach(
        item -> {
          Map<String, Object> contentMap = toObjectMap(item);
          if (contentMap == null) {
            return;
          }
          String type = (String) contentMap.get("type");
          if ("text".equals(type)) {
            contents.add(textContent((String) contentMap.get("text")));
          }
        });
    return contents;
  }

  private TextContent textContent(String text) {
    TextContent textContent = new TextContent();
    textContent.setType("text");
    textContent.setText(text);
    return textContent;
  }

  /**
   * Processes and validates installation configuration. Handles URL validation and configuration
   * transformation based on installation type.
   *
   * @param originDeployConfig Original deployment configuration string
   * @param installType Installation type (e.g., SSE or STREAMABLE_HTTP)
   * @return Result containing processed configuration or error
   */
  public Result<String> processInstallConfig(String originDeployConfig, String installType) {
    try {
      McpInstallTypeEnum installTypeEnum = McpInstallTypeEnum.of(installType);
      HashMap<String, Object> targetDeployConfig = new HashMap<>();
      Map<String, Object> originConfig = JsonUtils.fromJsonToMap(originDeployConfig);
      Map<String, Object> mcpServers =
          JsonUtils.fromJsonToMap(JsonUtils.toJson(originConfig.get("mcpServers")));

      if (mcpServers == null || mcpServers.keySet().size() != 1) {
        LogUtils.error(
            "ParseConfigError", "mcpServers must be configured and only one is supported");
        return Result.error(MCP_PARSE_CONFIG_ERROR);
      }
      targetDeployConfig.put("install_config", originDeployConfig);
      if (installTypeEnum == McpInstallTypeEnum.SSE
          || installTypeEnum == McpInstallTypeEnum.STREAMABLE_HTTP) {
        // check request header and host address
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
            HashMap<String, String> headers =
                toStringHeaderMap(
                    JsonUtils.fromJsonToMap(JsonUtils.toJson(singleServerConfig.get("headers"))));
            targetDeployConfig.put("remote_header", headers);
          } catch (Exception urlCheckEx) {
            LogUtils.error("processInstallConfig", url, urlCheckEx);
            return Result.error(MCP_PARSE_URL_ERROR);
          }
          break;
        }
      }
      return Result.success(JsonUtils.toJson(targetDeployConfig));
    } catch (Exception ex) {
      LogUtils.error("processInstallConfig exception", ex, originDeployConfig);
      throw ex;
    }
  }

  private HashMap<String, String> toStringHeaderMap(Map<String, Object> headers) {
    HashMap<String, String> result = new HashMap<>();
    if (headers == null) {
      return result;
    }
    headers.forEach(
        (key, value) -> {
          if (StringUtils.isNotBlank(key) && value != null) {
            result.put(key, String.valueOf(value));
          }
        });
    return result;
  }
}
