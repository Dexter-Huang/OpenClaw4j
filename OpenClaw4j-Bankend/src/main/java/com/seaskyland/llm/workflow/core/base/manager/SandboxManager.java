/*
 * Copyright 2024-2025 the original author or authors.
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

import com.google.common.collect.Maps;
import com.seaskyland.llm.workflow.core.config.SandboxProperties;
import com.seaskyland.llm.workflow.runtime.domain.Result;
import com.seaskyland.llm.workflow.runtime.enums.ErrorCode;
import com.seaskyland.llm.workflow.runtime.utils.JsonUtils;
import java.io.IOException;
import java.io.PrintWriter;
import java.io.StringWriter;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;
import java.util.Map;
import javax.script.ScriptException;
import lombok.Data;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.stereotype.Component;

/**
 * Script sandbox manager.
 *
 * <p>Python and JavaScript are delegated to the Rust/Axum sandbox service so untrusted workflow
 * code runs in a Linux sandlock process boundary. Java script execution keeps the existing JVM path
 * for compatibility until it has a separate isolation design.
 */
@Slf4j
@Component
public class SandboxManager {

  private static final String LANGUAGE_PYTHON = "python";
  private static final String LANGUAGE_JAVASCRIPT = "javascript";

  private final SandboxProperties sandboxProperties;

  private final HttpClient httpClient;

  public SandboxManager(SandboxProperties sandboxProperties) {
    this.sandboxProperties = sandboxProperties;
    this.httpClient =
        HttpClient.newBuilder()
            .connectTimeout(Duration.ofMillis(sandboxProperties.getTimeoutMs()))
            .build();
  }

  /**
   * Executes JavaScript through the Rust sandbox service.
   *
   * @param scriptContent Script content
   * @param localVariableMap Map of variable values
   * @param requestId Request ID
   * @return Execution result
   */
  public Result<String> executeJavaScript(
      String scriptContent, Map<String, Object> localVariableMap, String requestId) {
    return executeRemoteScript(LANGUAGE_JAVASCRIPT, scriptContent, localVariableMap, requestId);
  }

  /**
   * Executes Java code through the legacy JVM implementation.
   *
   * @param scriptContent Script content
   * @param localVariableMap Map of variable values
   * @param requestId Request ID
   * @return Execution result
   */
  public Result<String> executeJava(
      String scriptContent, Map<String, Object> localVariableMap, String requestId) {
    if (StringUtils.isBlank(scriptContent)) {
      log.error("Script content cannot be empty");
      return Result.error(requestId, ErrorCode.INVALID_PARAMS);
    }

    try {
      Object result;
      try {
        result = ASMCodeExecutor.execute(scriptContent, localVariableMap);
      } catch (ScriptException e) {
        log.error("Script execution error: {}", e.getMessage());
        return Result.success(requestId, buildScriptFailure("SCRIPT_ERROR", e.getMessage()));
      }

      Map<String, Object> resultMap = Maps.newHashMap();
      resultMap.put("success", true);
      resultMap.put("data", result);
      return Result.success(requestId, JsonUtils.toJson(resultMap));
    } catch (Exception e) {
      log.error("Script execution exception", e);
      StringWriter sw = new StringWriter();
      e.printStackTrace(new PrintWriter(sw));
      return Result.error(requestId, ErrorCode.SYSTEM_ERROR);
    }
  }

  /**
   * Executes Python3 script through the Rust sandbox service.
   *
   * @param scriptContent Python script content
   * @param variables Script variable mapping
   * @param requestId Request ID
   * @return Execution result
   */
  public Result<String> executePython3Script(
      String scriptContent, Map<String, Object> variables, String requestId) {
    return executeRemoteScript(LANGUAGE_PYTHON, scriptContent, variables, requestId);
  }

  private Result<String> executeRemoteScript(
      String language, String scriptContent, Map<String, Object> variables, String requestId) {
    if (StringUtils.isBlank(scriptContent)) {
      log.error("Script content cannot be empty");
      return Result.error(requestId, ErrorCode.INVALID_PARAMS);
    }
    if (!sandboxProperties.isEnabled()) {
      return Result.success(
          requestId,
          buildScriptFailure("SANDBOX_DISABLED", "Rust script sandbox service is disabled"));
    }

    RemoteScriptRequest request =
        new RemoteScriptRequest()
            .setRequestId(requestId)
            .setLanguage(language)
            .setCode(scriptContent)
            .setParams(variables == null ? Maps.newHashMap() : variables)
            .setTimeoutMs(sandboxProperties.getTimeoutMs());

    try {
      HttpRequest httpRequest =
          HttpRequest.newBuilder()
              .uri(URI.create(normalizeBaseUrl(sandboxProperties.getBaseUrl()) + "/v1/execute"))
              .timeout(Duration.ofMillis(sandboxProperties.getTimeoutMs()))
              .header("Content-Type", "application/json")
              .header("Accept", "application/json")
              .POST(HttpRequest.BodyPublishers.ofString(JsonUtils.toJson(request)))
              .build();
      long start = System.currentTimeMillis();
      HttpResponse<String> response =
          httpClient.send(httpRequest, HttpResponse.BodyHandlers.ofString());
      long duration = System.currentTimeMillis() - start;
      if (response.statusCode() < 200 || response.statusCode() >= 300) {
        log.warn(
            "Rust sandbox HTTP call failed, requestId={}, language={}, status={}, durationMs={}",
            requestId,
            language,
            response.statusCode(),
            duration);
        return Result.success(
            requestId,
            buildScriptFailure(
                "SANDBOX_UNAVAILABLE", "Rust sandbox returned HTTP " + response.statusCode()));
      }

      RemoteScriptResponse remoteResponse =
          JsonUtils.fromJson(response.body(), RemoteScriptResponse.class);
      log.info(
          "Rust sandbox execution finished, requestId={}, language={}, success={}, code={}, durationMs={}",
          requestId,
          language,
          remoteResponse.getSuccess(),
          remoteResponse.getCode(),
          duration);
      return Result.success(requestId, convertRemoteResponse(remoteResponse));
    } catch (IOException e) {
      log.warn(
          "Rust sandbox is unavailable, requestId={}, language={}, message={}",
          requestId,
          language,
          e.getMessage());
      return Result.success(requestId, buildScriptFailure("SANDBOX_UNAVAILABLE", e.getMessage()));
    } catch (InterruptedException e) {
      Thread.currentThread().interrupt();
      return Result.success(
          requestId, buildScriptFailure("SANDBOX_INTERRUPTED", "Sandbox request interrupted"));
    } catch (Exception e) {
      log.warn(
          "Rust sandbox response parse failed, requestId={}, language={}, message={}",
          requestId,
          language,
          e.getMessage());
      return Result.success(requestId, buildScriptFailure("SANDBOX_ERROR", e.getMessage()));
    }
  }

  private String convertRemoteResponse(RemoteScriptResponse remoteResponse) {
    Map<String, Object> resultMap = Maps.newHashMap();
    resultMap.put("success", Boolean.TRUE.equals(remoteResponse.getSuccess()));
    if (Boolean.TRUE.equals(remoteResponse.getSuccess())) {
      resultMap.put("data", remoteResponse.getData());
    } else {
      resultMap.put("message", remoteResponse.getMessage());
      resultMap.put("code", remoteResponse.getCode());
    }
    return JsonUtils.toJson(resultMap);
  }

  private String buildScriptFailure(String code, String message) {
    Map<String, Object> errorResult = Maps.newHashMap();
    errorResult.put("success", false);
    errorResult.put("message", message);
    errorResult.put("code", code);
    return JsonUtils.toJson(errorResult);
  }

  private String normalizeBaseUrl(String baseUrl) {
    return StringUtils.removeEnd(baseUrl, "/");
  }

  @Data
  @lombok.experimental.Accessors(chain = true)
  static class RemoteScriptRequest {
    @com.fasterxml.jackson.annotation.JsonProperty("request_id")
    private String requestId;

    private String language;

    private String code;

    private Map<String, Object> params;

    @com.fasterxml.jackson.annotation.JsonProperty("timeout_ms")
    private Integer timeoutMs;
  }

  @Data
  static class RemoteScriptResponse {
    private Boolean success;

    private Object data;

    private String message;

    private String code;
  }
}
