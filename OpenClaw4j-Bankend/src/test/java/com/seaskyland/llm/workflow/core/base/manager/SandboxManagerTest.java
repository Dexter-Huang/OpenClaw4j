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

import static org.assertj.core.api.Assertions.assertThat;

import com.seaskyland.llm.workflow.core.config.SandboxProperties;
import com.seaskyland.llm.workflow.runtime.domain.Result;
import com.seaskyland.llm.workflow.runtime.utils.JsonUtils;
import com.sun.net.httpserver.HttpServer;
import java.io.IOException;
import java.net.InetSocketAddress;
import java.nio.charset.StandardCharsets;
import java.util.Map;
import java.util.concurrent.atomic.AtomicReference;
import org.junit.jupiter.api.Test;

class SandboxManagerTest {

  @Test
  void delegatesJavascriptExecutionToRustSandboxHttpService() throws Exception {
    AtomicReference<String> requestBody = new AtomicReference<>();
    HttpServer server =
        startServer(
            exchange -> {
              requestBody.set(
                  new String(exchange.getRequestBody().readAllBytes(), StandardCharsets.UTF_8));
              byte[] response =
                  """
                  {"success":true,"data":{"sum":3},"stdout":"","stderr":"","exit_code":0,"duration_ms":5}
                  """
                      .getBytes(StandardCharsets.UTF_8);
              exchange.getResponseHeaders().add("Content-Type", "application/json");
              exchange.sendResponseHeaders(200, response.length);
              exchange.getResponseBody().write(response);
              exchange.close();
            });
    try {
      SandboxManager manager = new SandboxManager(properties(server));

      Result<String> result =
          manager.executeJavaScript(
              "function main(params) { return { sum: params.a + params.b }; }",
              Map.of("a", 1, "b", 2),
              "req-1");

      assertThat(result.isSuccess()).isTrue();
      Map<String, Object> response = JsonUtils.fromJsonToMap(result.getData());
      assertThat(response).containsEntry("success", true);
      assertThat(response.get("data")).isEqualTo(Map.of("sum", 3));
      Map<String, Object> remoteRequest = JsonUtils.fromJsonToMap(requestBody.get());
      assertThat(remoteRequest).containsEntry("language", "javascript");
      assertThat(remoteRequest).containsEntry("request_id", "req-1");
      assertThat(remoteRequest.get("params")).isEqualTo(Map.of("a", 1, "b", 2));
    } finally {
      server.stop(0);
    }
  }

  @Test
  void wrapsUnavailableRustSandboxAsScriptFailure() {
    SandboxProperties properties = new SandboxProperties();
    properties.setBaseUrl("http://127.0.0.1:1");
    properties.setTimeoutMs(100);
    SandboxManager manager = new SandboxManager(properties);

    Result<String> result =
        manager.executePython3Script("def main(params): return params", Map.of(), "req-2");

    assertThat(result.isSuccess()).isTrue();
    Map<String, Object> response = JsonUtils.fromJsonToMap(result.getData());
    assertThat(response).containsEntry("success", false);
    assertThat(response).containsEntry("code", "SANDBOX_UNAVAILABLE");
  }

  private HttpServer startServer(ThrowingHandler handler) throws IOException {
    HttpServer server = HttpServer.create(new InetSocketAddress("127.0.0.1", 0), 0);
    server.createContext(
        "/v1/execute",
        exchange -> {
          try {
            handler.handle(exchange);
          } catch (Exception e) {
            byte[] response = e.getMessage().getBytes(StandardCharsets.UTF_8);
            exchange.sendResponseHeaders(500, response.length);
            exchange.getResponseBody().write(response);
            exchange.close();
          }
        });
    server.start();
    return server;
  }

  private SandboxProperties properties(HttpServer server) {
    SandboxProperties properties = new SandboxProperties();
    properties.setBaseUrl("http://127.0.0.1:" + server.getAddress().getPort());
    properties.setTimeoutMs(1000);
    return properties;
  }

  @FunctionalInterface
  private interface ThrowingHandler {
    void handle(com.sun.net.httpserver.HttpExchange exchange) throws Exception;
  }
}
