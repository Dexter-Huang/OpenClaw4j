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

import com.seaskyland.llm.workflow.core.base.entity.McpServerEntity;
import com.seaskyland.llm.workflow.runtime.domain.Result;
import com.seaskyland.llm.workflow.runtime.domain.mcp.McpServerCallToolRequest;
import com.seaskyland.llm.workflow.runtime.domain.mcp.McpServerCallToolResponse;
import com.seaskyland.llm.workflow.runtime.domain.mcp.McpServerDeployConfig;
import com.seaskyland.llm.workflow.runtime.domain.mcp.McpTool;
import com.seaskyland.llm.workflow.runtime.domain.mcp.TextContent;
import com.seaskyland.llm.workflow.runtime.enums.ErrorCode;
import com.seaskyland.llm.workflow.runtime.enums.McpInstallTypeEnum;
import com.seaskyland.llm.workflow.runtime.enums.McpServerStatusEnum;
import com.seaskyland.llm.workflow.runtime.utils.JsonUtils;
import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;
import java.io.OutputStream;
import java.net.InetAddress;
import java.net.ServerSocket;
import java.net.Socket;
import java.net.SocketException;
import java.nio.charset.StandardCharsets;
import java.util.HashMap;
import java.util.List;
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
    assertThat(JsonUtils.fromJsonToMap(deployConfig.getInstallConfig())).containsKey("mcpServers");
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
      assertThat(tools.getFirst().getName()).isEqualTo("file_read");
      assertThat(tools.getFirst().getDescription()).isEqualTo("Read a file");
      assertThat(tools.getFirst().getInputSchema().getType()).isEqualTo("object");
      assertThat(tools.getFirst().getInputSchema().getRequired()).containsExactly("path");
      assertThat(tools.getFirst().getInputSchema().getAdditionalProperties()).isFalse();
    }
  }

  @Test
  void getToolsReadsToolsFromStreamableHttpSseEnvelope() throws Exception {
    try (TestMcpHttpServer server =
        TestMcpHttpServer.start(
            """
            event: message
            data: {"jsonrpc":"2.0","id":1,"result":{"tools":[{"name":"camofox_create_tab","description":"Create a new Camofox browser tab.","inputSchema":{"type":"object","properties":{"url":{"type":"string"}},"required":["url"]}}]}}

            """,
            "text/event-stream")) {
      McpServerEntity entity = streamableHttpEntity(server.url("/mcp"));

      List<McpTool> tools = manager.getTools(entity);

      assertThat(server.lastRequestHeaders().get("Accept"))
          .contains("application/json")
          .contains("text/event-stream");
      assertThat(tools).hasSize(1);
      assertThat(tools.getFirst().getName()).isEqualTo("camofox_create_tab");
      assertThat(tools.getFirst().getInputSchema().getRequired()).containsExactly("url");
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
      assertThat((TextContent) response.getContent().getFirst())
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
      assertThat((TextContent) response.getContent().getFirst())
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
    assertThat(config.isSuccess()).isTrue();

    McpServerDeployConfig deployConfig =
        JsonUtils.fromJson(config.getData(), McpServerDeployConfig.class);
    McpServerEntity entity = new McpServerEntity();
    entity.setServerCode("aio-sandbox");
    entity.setStatus(McpServerStatusEnum.Normal.getCode());
    entity.setInstallType(McpInstallTypeEnum.STREAMABLE_HTTP.name());
    entity.setDeployConfig(config.getData());
    entity.setHost(deployConfig.getRemoteAddress());
    return entity;
  }

  private static final class TestMcpHttpServer implements AutoCloseable {

    private final ServerSocket serverSocket;
    private final Thread serverThread;
    private volatile String responseBody;
    private volatile String contentType;
    private volatile String lastRequestBody;
    private volatile Map<String, String> lastRequestHeaders = new HashMap<>();

    private TestMcpHttpServer(ServerSocket serverSocket, String responseBody, String contentType) {
      this.serverSocket = serverSocket;
      this.responseBody = responseBody;
      this.contentType = contentType;
      this.serverThread = new Thread(this::handleNextRequest, "test-mcp-http-server");
    }

    static TestMcpHttpServer start(String responseBody) throws IOException {
      return start(responseBody, "application/json");
    }

    static TestMcpHttpServer start(String responseBody, String contentType) throws IOException {
      ServerSocket serverSocket = new ServerSocket(0, 1, InetAddress.getByName("127.0.0.1"));
      TestMcpHttpServer testServer = new TestMcpHttpServer(serverSocket, responseBody, contentType);
      testServer.serverThread.setDaemon(true);
      testServer.serverThread.start();
      return testServer;
    }

    String url(String path) {
      return "http://127.0.0.1:" + serverSocket.getLocalPort() + path;
    }

    String lastRequestBody() {
      return lastRequestBody;
    }

    Map<String, String> lastRequestHeaders() {
      return lastRequestHeaders;
    }

    private void handleNextRequest() {
      try (Socket socket = serverSocket.accept()) {
        socket.setSoTimeout(5000);
        BufferedReader reader =
            new BufferedReader(
                new InputStreamReader(socket.getInputStream(), StandardCharsets.UTF_8));
        reader.readLine();
        Map<String, String> headers = readHeaders(reader);
        lastRequestHeaders = headers;
        lastRequestBody = readBody(reader, headers);
        writeResponse(socket.getOutputStream());
      } catch (SocketException ex) {
        if (!serverSocket.isClosed()) {
          throw new IllegalStateException(ex);
        }
      } catch (IOException ex) {
        throw new IllegalStateException(ex);
      }
    }

    private Map<String, String> readHeaders(BufferedReader reader) throws IOException {
      Map<String, String> headers = new HashMap<>();
      String line;
      while ((line = reader.readLine()) != null && !line.isEmpty()) {
        int separator = line.indexOf(':');
        if (separator > 0) {
          headers.put(line.substring(0, separator), line.substring(separator + 1).trim());
        }
      }
      return headers;
    }

    private String readBody(BufferedReader reader, Map<String, String> headers) throws IOException {
      int length = Integer.parseInt(headers.getOrDefault("Content-Length", "0"));
      char[] body = new char[length];
      int offset = 0;
      while (offset < length) {
        int count = reader.read(body, offset, length - offset);
        if (count < 0) {
          break;
        }
        offset += count;
      }
      return new String(body, 0, offset);
    }

    private void writeResponse(OutputStream outputStream) throws IOException {
      byte[] bytes = responseBody.getBytes(StandardCharsets.UTF_8);
      String headers =
          "HTTP/1.1 200 OK\r\n"
              + "Content-Type: "
              + contentType
              + "\r\n"
              + "Content-Length: "
              + bytes.length
              + "\r\n"
              + "Connection: close\r\n\r\n";
      outputStream.write(headers.getBytes(StandardCharsets.UTF_8));
      outputStream.write(bytes);
      outputStream.flush();
    }

    @Override
    public void close() throws IOException {
      serverSocket.close();
    }
  }
}
