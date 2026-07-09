package com.seaskyland.llm.workflow.admin.compat;

import static org.assertj.core.api.Assertions.assertThat;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.delete;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.multipart;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.put;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.seaskyland.llm.workflow.core.base.manager.TokenManager;
import java.io.ByteArrayOutputStream;
import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;
import java.net.InetSocketAddress;
import java.net.ServerSocket;
import java.net.Socket;
import java.nio.charset.StandardCharsets;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.atomic.AtomicReference;
import java.util.zip.ZipEntry;
import java.util.zip.ZipOutputStream;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.webmvc.test.autoconfigure.AutoConfigureMockMvc;
import org.springframework.http.MediaType;
import org.springframework.mock.web.MockMultipartFile;
import org.springframework.test.context.TestPropertySource;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.request.MockHttpServletRequestBuilder;

@SpringBootTest
@AutoConfigureMockMvc
@TestPropertySource(
    properties = {
      "spring.datasource.url=jdbc:h2:mem:admin-compat-test;MODE=MySQL;DATABASE_TO_LOWER=TRUE;DB_CLOSE_DELAY=-1",
      "spring.datasource.driver-class-name=org.h2.Driver",
      "spring.sql.init.mode=always",
      "spring.sql.init.schema-locations=classpath:sql/H2/V0.0.1__init.sql",
      "cache.type=JVM",
      "mq.type=JVM"
    })
class AdminCompatApiTest {

  @Autowired private MockMvc mockMvc;

  @Autowired private ObjectMapper objectMapper;

  @Autowired private TokenManager tokenManager;

  @Test
  void adminListApisReturnLegacyPageEnvelope() throws Exception {
    assertLegacyPage("/api/prompts?pageNo=1&pageSize=10");
    assertLegacyPage("/api/dataset/datasets?pageNumber=1&pageSize=10");
    assertLegacyPage("/api/evaluator/evaluators?pageNumber=1&pageSize=10");
    assertLegacyPage("/api/experiments?pageNumber=1&pageSize=10");
    assertLegacyPage("/api/models");
  }

  @Test
  void providerDetailReturnsModelCountConsistentWithModelList() throws Exception {
    JsonNode models = getConsoleJson("/console/v1/providers/Tongyi/models");
    JsonNode provider = getConsoleJson("/console/v1/providers/Tongyi");

    JsonNode modelCount = provider.path("data").path("model_count");
    assertThat(modelCount.isNumber()).isTrue();
    assertThat(modelCount.asInt()).isEqualTo(models.path("data").size());
  }

  @Test
  void addingOpenAiProviderSynchronizesRemoteModels() throws Exception {
    AtomicReference<String> authorization = new AtomicReference<>();
    TestModelsServer server =
        startModelsServer(
            200,
            """
            {"object":"list","data":[{"id":"qwen-plus"},{"id":"text-embedding-v3"}]}
            """,
            authorization);

    try {
      String providerName = "remote_sync_provider";
      String endpoint = modelsEndpoint(server);
      postConsoleJson(
          "/console/v1/providers",
          """
          {
            "name": "%s",
            "protocol": "OpenAI",
            "credential_config": {
              "api_key": "sk-test",
              "endpoint": "%s",
              "completions_path": "/v1/chat/completions",
              "embeddings_path": "/v1/embeddings"
            }
          }
          """
              .formatted(providerName, endpoint));

      JsonNode provider = firstProviderByName(providerName);
      String providerCode = provider.path("provider").asText();
      assertThat(provider.path("model_count").asInt()).isEqualTo(2);

      JsonNode models = getConsoleJson("/console/v1/providers/" + providerCode + "/models");
      JsonNode llmModel = findModel(models.path("data"), "qwen-plus");
      JsonNode embeddingModel = findModel(models.path("data"), "text-embedding-v3");
      assertThat(llmModel.path("type").asText()).isEqualTo("llm");
      assertThat(embeddingModel.path("type").asText()).isEqualTo("text_embedding");
      assertThat(authorization.get()).isEqualTo("sk-test");
    } finally {
      server.close();
    }
  }

  @Test
  void addingOpenAiProviderBlocksWhenRemoteModelsCannotBeFetched() throws Exception {
    AtomicReference<String> authorization = new AtomicReference<>();
    TestModelsServer server =
        startModelsServer(401, "{\"error\":{\"message\":\"invalid api key\"}}", authorization);

    try {
      String providerName = "remote_sync_failure_provider";
      String endpoint = modelsEndpoint(server);
      String accessToken = tokenManager.generateAccessToken("10000");
      String content =
          mockMvc
              .perform(
                  post("/console/v1/providers")
                      .header("X-SAA-TOKEN", "Bearer " + accessToken)
                      .contentType(MediaType.APPLICATION_JSON)
                      .content(
                          """
                          {
                            "name": "%s",
                            "protocol": "OpenAI",
                            "credential_config": {
                              "api_key": "sk-test",
                              "endpoint": "%s",
                              "completions_path": "/v1/chat/completions",
                              "embeddings_path": "/v1/embeddings"
                            }
                          }
                          """
                              .formatted(providerName, endpoint)))
              .andExpect(status().isBadRequest())
              .andReturn()
              .getResponse()
              .getContentAsString();

      JsonNode response = objectMapper.readTree(content);
      assertThat(response.path("code").asText()).isEqualTo("InvalidParameter");
      assertThat(response.path("message").asText()).contains("remote models");
      assertThat(firstProviderByName(providerName).isMissingNode()).isTrue();
      assertThat(authorization.get()).isEqualTo("sk-test");
    } finally {
      server.close();
    }
  }

  @Test
  void updatingOpenAiProviderBlocksWhenRemoteModelsCannotBeFetched() throws Exception {
    AtomicReference<String> authorization = new AtomicReference<>();
    TestModelsServer successServer =
        startModelsServer(200, "{\"object\":\"list\",\"data\":[{\"id\":\"qwen-plus\"}]}", null);
    TestModelsServer failureServer =
        startModelsServer(401, "{\"error\":{\"message\":\"invalid api key\"}}", authorization);

    try {
      String providerName = "remote_sync_update_failure_provider";
      postConsoleJson(
          "/console/v1/providers",
          """
          {
            "name": "%s",
            "protocol": "OpenAI",
            "credential_config": {
              "api_key": "sk-test",
              "endpoint": "%s",
              "completions_path": "/v1/chat/completions",
              "embeddings_path": "/v1/embeddings"
            }
          }
          """
              .formatted(providerName, modelsEndpoint(successServer)));
      String providerCode = firstProviderByName(providerName).path("provider").asText();

      String accessToken = tokenManager.generateAccessToken("10000");
      mockMvc
          .perform(
              put("/console/v1/providers/" + providerCode)
                  .header("X-SAA-TOKEN", "Bearer " + accessToken)
                  .contentType(MediaType.APPLICATION_JSON)
                  .content(
                      """
                      {
                        "name": "%s",
                        "protocol": "OpenAI",
                        "credential_config": {
                          "api_key": "sk-test",
                          "endpoint": "%s",
                          "completions_path": "/v1/chat/completions",
                          "embeddings_path": "/v1/embeddings"
                        }
                      }
                      """
                          .formatted(providerName + "_blocked", modelsEndpoint(failureServer))))
          .andExpect(status().isBadRequest());

      JsonNode provider = getConsoleJson("/console/v1/providers/" + providerCode).path("data");
      assertThat(provider.path("name").asText()).isEqualTo(providerName);
      assertThat(authorization.get()).isEqualTo("sk-test");
    } finally {
      successServer.close();
      failureServer.close();
    }
  }

  @Test
  void promptWriteApisCreateAndUpdateCompatibilityRecords() throws Exception {
    JsonNode created =
        postJson(
            "/api/prompt",
            """
            {
              "promptKey": "compat_prompt",
              "promptDescription": "Initial prompt",
              "tags": "compat"
            }
            """);
    assertThat(created.path("data").path("promptKey").asText()).isEqualTo("compat_prompt");
    assertThat(created.path("data").path("promptDescription").asText()).isEqualTo("Initial prompt");

    JsonNode updated =
        postJson(
            "/api/prompt",
            """
            {
              "promptKey": "compat_prompt",
              "promptDescription": "Updated prompt",
              "tags": "compat,updated"
            }
            """);
    assertThat(updated.path("data").path("promptDescription").asText()).isEqualTo("Updated prompt");
    assertThat(updated.path("data").path("tags").asText()).isEqualTo("compat,updated");

    JsonNode version =
        postJson(
            "/api/prompt/version",
            """
            {
              "promptKey": "compat_prompt",
              "version": "1.0.1",
              "versionDescription": "First test version",
              "template": "Hello {{name}}",
              "variables": "[{\\"name\\":\\"name\\"}]",
              "modelConfig": "{\\"model\\":\\"qwen-max\\"}",
              "status": "pre"
            }
            """);
    assertThat(version.path("data").path("promptKey").asText()).isEqualTo("compat_prompt");
    assertThat(version.path("data").path("version").asText()).isEqualTo("1.0.1");
    assertThat(version.path("data").path("versionDescription").asText())
        .isEqualTo("First test version");

    JsonNode prompt = getJson("/api/prompt?promptKey=compat_prompt");
    assertThat(prompt.path("data").path("latestVersion").asText()).isEqualTo("1.0.1");
  }

  @Test
  void skillConsoleApisManageVersionedSkillPackages() throws Exception {
    JsonNode uploaded = postConsoleMultipart(createSkillPackageZip("SQL 审查 Skill", false));
    JsonNode uploadedPackage = uploaded.path("data");
    assertThat(uploadedPackage.path("storage_type").asText()).isEqualTo("file");
    assertThat(uploadedPackage.path("storage_prefix").asText()).startsWith("skills/1/packages/");
    assertThat(uploadedPackage.path("package_object_key").asText()).endsWith("/package.zip");
    assertThat(uploadedPackage.path("main_file_path").asText()).isEqualTo("SKILL.md");
    assertThat(uploadedPackage.path("file_count").asInt()).isEqualTo(2);
    assertThat(uploadedPackage.path("total_size_bytes").asLong()).isGreaterThan(0);
    assertThat(uploadedPackage.path("manifest").asText()).contains("script_policy");

    JsonNode created =
        postConsoleJson(
            "/console/v1/skills",
            """
            {
              "name": "SQL 审查",
              "description": "审查 SQL、索引和慢查询",
              "tags": "sql,review",
              "main_file_path": "SKILL.md",
              "storage_prefix": "%s",
              "package_object_key": "%s",
              "file_count": %d,
              "total_size_bytes": %d,
              "manifest": %s,
              "content_hash": "%s"
            }
            """
                .formatted(
                    uploadedPackage.path("storage_prefix").asText(),
                    uploadedPackage.path("package_object_key").asText(),
                    uploadedPackage.path("file_count").asInt(),
                    uploadedPackage.path("total_size_bytes").asLong(),
                    objectMapper.writeValueAsString(uploadedPackage.path("manifest").asText()),
                    uploadedPackage.path("content_hash").asText()));
    String skillCode = created.path("data").asText();
    assertThat(skillCode).isNotBlank();

    JsonNode detail =
        getConsoleJson("/console/v1/skills/" + skillCode + "?version=1&need_files=true");
    assertThat(detail.path("data").path("name").asText()).isEqualTo("SQL 审查");
    assertThat(detail.path("data").path("current_version").asText()).isEqualTo("1");
    assertThat(detail.path("data").path("version").asText()).isEqualTo("1");
    assertThat(detail.path("data").path("status").asInt()).isEqualTo(1);
    assertThat(detail.path("data").path("storage_type").asText()).isEqualTo("file");
    assertThat(detail.path("data").path("storage_prefix").asText())
        .isEqualTo(uploadedPackage.path("storage_prefix").asText());
    assertThat(detail.path("data").path("main_file_path").asText()).isEqualTo("SKILL.md");
    assertThat(detail.path("data").path("package_object_key").asText())
        .isEqualTo(uploadedPackage.path("package_object_key").asText());
    assertThat(detail.path("data").path("file_count").asInt()).isEqualTo(2);
    assertThat(detail.path("data").path("total_size_bytes").asLong())
        .isEqualTo(uploadedPackage.path("total_size_bytes").asLong());
    assertThat(detail.path("data").has("files")).isFalse();

    JsonNode queried =
        postConsoleJson(
            "/console/v1/skills/query-by-codes",
            """
            {
              "skill_codes": ["%s"],
              "need_files": true
            }
            """
                .formatted(skillCode));
    assertThat(queried.path("data")).hasSize(1);
    assertThat(queried.path("data").get(0).path("version").asText()).isEqualTo("1");
    assertThat(queried.path("data").get(0).path("storage_type").asText()).isEqualTo("file");
    assertThat(queried.path("data").get(0).path("storage_prefix").asText())
        .isEqualTo(uploadedPackage.path("storage_prefix").asText());

    JsonNode draftUpload = postConsoleMultipart(createSkillPackageZip("SQL 草稿审查 Skill", true));
    JsonNode draftPackage = draftUpload.path("data");
    putConsoleJson(
        "/console/v1/skills",
        """
        {
          "skill_code": "%s",
          "name": "SQL 草稿审查",
          "description": "草稿更新也应创建新版本",
          "tags": "sql,review,draft",
          "main_file_path": "SKILL.md",
          "storage_prefix": "%s",
          "package_object_key": "%s",
          "file_count": %d,
          "total_size_bytes": %d,
          "content_hash": "%s"
        }
        """
            .formatted(
                skillCode,
                draftPackage.path("storage_prefix").asText(),
                draftPackage.path("package_object_key").asText(),
                draftPackage.path("file_count").asInt(),
                draftPackage.path("total_size_bytes").asLong(),
                draftPackage.path("content_hash").asText()));
    JsonNode draftUpdated = getConsoleJson("/console/v1/skills/" + skillCode + "?need_files=true");
    assertThat(draftUpdated.path("data").path("status").asInt()).isEqualTo(1);
    assertThat(draftUpdated.path("data").path("current_version").asText()).isEqualTo("2");
    assertThat(draftUpdated.path("data").path("storage_prefix").asText())
        .isEqualTo(draftPackage.path("storage_prefix").asText());

    postConsoleJson("/console/v1/skills/" + skillCode + "/publish", "{}");
    JsonNode published = getConsoleJson("/console/v1/skills/" + skillCode);
    assertThat(published.path("data").path("status").asInt()).isEqualTo(2);
    assertThat(published.path("data").path("current_version").asText()).isEqualTo("2");

    JsonNode updatedUpload = postConsoleMultipart(createSkillPackageZip("SQL 深度审查 Skill", true));
    JsonNode updatedPackage = updatedUpload.path("data");
    putConsoleJson(
        "/console/v1/skills",
        """
        {
          "skill_code": "%s",
          "name": "SQL 深度审查",
          "description": "审查 SQL、索引、慢查询和回归风险",
          "tags": "sql,review,slow-query",
          "main_file_path": "SKILL.md",
          "storage_prefix": "%s",
          "package_object_key": "%s",
          "file_count": %d,
          "total_size_bytes": %d,
          "content_hash": "%s"
        }
        """
            .formatted(
                skillCode,
                updatedPackage.path("storage_prefix").asText(),
                updatedPackage.path("package_object_key").asText(),
                updatedPackage.path("file_count").asInt(),
                updatedPackage.path("total_size_bytes").asLong(),
                updatedPackage.path("content_hash").asText()));

    JsonNode updated = getConsoleJson("/console/v1/skills/" + skillCode + "?need_files=true");
    assertThat(updated.path("data").path("name").asText()).isEqualTo("SQL 深度审查");
    assertThat(updated.path("data").path("status").asInt()).isEqualTo(3);
    assertThat(updated.path("data").path("current_version").asText()).isEqualTo("3");
    assertThat(updated.path("data").path("version").asText()).isEqualTo("3");
    assertThat(updated.path("data").path("storage_type").asText()).isEqualTo("file");
    assertThat(updated.path("data").path("storage_prefix").asText())
        .isEqualTo(updatedPackage.path("storage_prefix").asText());
    assertThat(updated.path("data").path("file_count").asInt()).isEqualTo(3);
    assertThat(updated.path("data").path("total_size_bytes").asLong())
        .isEqualTo(updatedPackage.path("total_size_bytes").asLong());
    assertThat(updated.path("data").has("files")).isFalse();

    deleteConsole("/console/v1/skills/" + skillCode);
    JsonNode list = getConsoleJson("/console/v1/skills?name=SQL&current=1&size=10");
    assertThat(list.path("data").path("records")).isEmpty();
  }

  @Test
  void observabilityStartsEmptyAndCanQueryInMemorySpans() throws Exception {
    JsonNode emptyTraces = getJson("/api/observability/traces?pageNumber=1&pageSize=10");
    assertThat(emptyTraces.has("code")).isFalse();
    assertThat(emptyTraces.path("data").path("totalCount").asLong()).isZero();
    assertThat(emptyTraces.path("data").path("pageItems")).isEmpty();

    mockMvc
        .perform(
            post("/api/observability/traces")
                .contentType(MediaType.APPLICATION_JSON)
                .content(
                    """
                    {
                      "traceId": "trace-1",
                      "spanId": "span-1",
                      "service": "openclaw4j",
                      "spanName": "POST /chat",
                      "startTime": "2026-06-24T03:05:27.842Z",
                      "endTime": "2026-06-24T03:05:28.842Z",
                      "durationNs": 1000000000,
                      "status": "Ok",
                      "attributes": {"model.name": "qwen-max", "usage.total_tokens": "42"}
                    }
                    """))
        .andExpect(status().isOk());

    JsonNode traces = getJson("/api/observability/traces?pageNumber=1&pageSize=10");
    assertThat(traces.path("data").path("totalCount").asLong()).isEqualTo(1);
    assertThat(traces.path("data").path("pageItems").get(0).path("traceId").asText())
        .isEqualTo("trace-1");

    JsonNode services = getJson("/api/observability/services");
    assertThat(services.path("data").path("services").get(0).path("name").asText())
        .isEqualTo("openclaw4j");
    assertThat(services.path("data").path("services").get(0).path("operations").get(0).asText())
        .isEqualTo("POST /chat");

    JsonNode overview = getJson("/api/observability/overview?detail=true");
    assertThat(overview.path("data").path("span.count").path("total").asInt()).isEqualTo(1);
    assertThat(overview.path("data").path("usage.tokens").path("total").asInt()).isEqualTo(42);
  }

  private void assertLegacyPage(String url) throws Exception {
    JsonNode response = getJson(url);
    JsonNode data = response.path("data");
    assertThat(response.has("code")).isFalse();
    assertThat(response.path("message").isMissingNode()).isTrue();
    assertThat(data.path("totalCount").isNumber()).isTrue();
    assertThat(data.path("totalPage").isNumber()).isTrue();
    assertThat(data.path("pageNumber").isNumber()).isTrue();
    assertThat(data.path("pageSize").isNumber()).isTrue();
    assertThat(data.path("pageItems").isArray()).isTrue();
  }

  private JsonNode getJson(String url) throws Exception {
    return getJson(get(url));
  }

  private JsonNode getConsoleJson(String url) throws Exception {
    String accessToken = tokenManager.generateAccessToken("10000");
    return getJson(get(url).header("X-SAA-TOKEN", "Bearer " + accessToken));
  }

  private JsonNode postConsoleJson(String url, String body) throws Exception {
    String accessToken = tokenManager.generateAccessToken("10000");
    return getJson(
        post(url)
            .header("X-SAA-TOKEN", "Bearer " + accessToken)
            .contentType(MediaType.APPLICATION_JSON)
            .content(body));
  }

  private JsonNode postConsoleMultipart(MockMultipartFile file) throws Exception {
    String accessToken = tokenManager.generateAccessToken("10000");
    String content =
        mockMvc
            .perform(
                multipart("/console/v1/skills/package")
                    .file(file)
                    .header("X-SAA-TOKEN", "Bearer " + accessToken))
            .andExpect(status().isOk())
            .andReturn()
            .getResponse()
            .getContentAsString();
    return objectMapper.readTree(content);
  }

  private JsonNode putConsoleJson(String url, String body) throws Exception {
    String accessToken = tokenManager.generateAccessToken("10000");
    return getJson(
        put(url)
            .header("X-SAA-TOKEN", "Bearer " + accessToken)
            .contentType(MediaType.APPLICATION_JSON)
            .content(body));
  }

  private void deleteConsole(String url) throws Exception {
    String accessToken = tokenManager.generateAccessToken("10000");
    mockMvc
        .perform(delete(url).header("X-SAA-TOKEN", "Bearer " + accessToken))
        .andExpect(status().isOk());
  }

  private JsonNode getJson(MockHttpServletRequestBuilder requestBuilder) throws Exception {
    String content =
        mockMvc
            .perform(requestBuilder)
            .andExpect(status().isOk())
            .andReturn()
            .getResponse()
            .getContentAsString();
    return objectMapper.readTree(content);
  }

  private MockMultipartFile createSkillPackageZip(String skillContent, boolean includeScript)
      throws IOException {
    ByteArrayOutputStream outputStream = new ByteArrayOutputStream();
    try (ZipOutputStream zipOutputStream = new ZipOutputStream(outputStream, StandardCharsets.UTF_8)) {
      writeZipEntry(zipOutputStream, "SKILL.md", "# " + skillContent + "\n");
      writeZipEntry(
          zipOutputStream,
          "manifest.json",
          """
          {"script_policy":{"mode":"disabled"}}
          """);
      if (includeScript) {
        writeZipEntry(zipOutputStream, "scripts/check.sql", "select 1;\n");
      }
    }
    return new MockMultipartFile(
        "file", "skill-package.zip", "application/zip", outputStream.toByteArray());
  }

  private void writeZipEntry(ZipOutputStream zipOutputStream, String name, String content)
      throws IOException {
    zipOutputStream.putNextEntry(new ZipEntry(name));
    zipOutputStream.write(content.getBytes(StandardCharsets.UTF_8));
    zipOutputStream.closeEntry();
  }

  private JsonNode firstProviderByName(String name) throws Exception {
    JsonNode providers = getConsoleJson("/console/v1/providers?name=" + name);
    JsonNode data = providers.path("data");
    if (data.isArray()) {
      for (JsonNode provider : data) {
        if (name.equals(provider.path("name").asText())) {
          return provider;
        }
      }
    }
    return objectMapper.missingNode();
  }

  private JsonNode findModel(JsonNode models, String modelId) {
    for (JsonNode model : models) {
      if (modelId.equals(model.path("model_id").asText())) {
        return model;
      }
    }
    return objectMapper.missingNode();
  }

  private TestModelsServer startModelsServer(
      int status, String body, AtomicReference<String> authorization) throws Exception {
    return TestModelsServer.start(status, body, authorization);
  }

  private String modelsEndpoint(TestModelsServer server) {
    return server.modelsEndpoint();
  }

  private static final class TestModelsServer implements AutoCloseable {

    private final ServerSocket serverSocket;

    private final ExecutorService executorService;

    private final int status;

    private final String body;

    private final AtomicReference<String> authorization;

    private volatile boolean running = true;

    private TestModelsServer(
        int status, String body, AtomicReference<String> authorization, ServerSocket serverSocket) {
      this.status = status;
      this.body = body;
      this.authorization = authorization;
      this.serverSocket = serverSocket;
      this.executorService = Executors.newSingleThreadExecutor();
    }

    static TestModelsServer start(int status, String body, AtomicReference<String> authorization)
        throws IOException {
      ServerSocket serverSocket = new ServerSocket();
      serverSocket.bind(new InetSocketAddress("127.0.0.1", 0));
      TestModelsServer server = new TestModelsServer(status, body, authorization, serverSocket);
      server.executorService.submit(server::acceptLoop);
      return server;
    }

    String modelsEndpoint() {
      return "http://127.0.0.1:" + serverSocket.getLocalPort() + "/compatible-mode/v1";
    }

    private void acceptLoop() {
      while (running) {
        try {
          handle(serverSocket.accept());
        } catch (IOException e) {
          if (running) {
            throw new IllegalStateException(e);
          }
        }
      }
    }

    private void handle(Socket socket) throws IOException {
      try (socket) {
        BufferedReader reader =
            new BufferedReader(
                new InputStreamReader(socket.getInputStream(), StandardCharsets.UTF_8));
        String line;
        while ((line = reader.readLine()) != null && !line.isEmpty()) {
          if (authorization != null
              && authorization.get() == null
              && line.regionMatches(true, 0, "Authorization:", 0, "Authorization:".length())) {
            authorization.set(line.substring("Authorization:".length()).trim());
          }
        }

        byte[] response = body.getBytes(StandardCharsets.UTF_8);
        String headers =
            "HTTP/1.1 "
                + status
                + " "
                + (status == 200 ? "OK" : "Error")
                + "\r\nContent-Type: application/json\r\nContent-Length: "
                + response.length
                + "\r\nConnection: close\r\n\r\n";
        socket.getOutputStream().write(headers.getBytes(StandardCharsets.US_ASCII));
        socket.getOutputStream().write(response);
        socket.getOutputStream().flush();
      }
    }

    @Override
    public void close() throws IOException {
      running = false;
      serverSocket.close();
      executorService.shutdownNow();
    }
  }

  private JsonNode postJson(String url, String body) throws Exception {
    String content =
        mockMvc
            .perform(post(url).contentType(MediaType.APPLICATION_JSON).content(body))
            .andExpect(status().isOk())
            .andReturn()
            .getResponse()
            .getContentAsString();
    return objectMapper.readTree(content);
  }
}
