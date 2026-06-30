package com.seaskyland.llm.workflow.admin.compat;

import static org.assertj.core.api.Assertions.assertThat;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.webmvc.test.autoconfigure.AutoConfigureMockMvc;
import org.springframework.http.MediaType;
import org.springframework.test.context.TestPropertySource;
import org.springframework.test.web.servlet.MockMvc;

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

  @Test
  void adminListApisReturnLegacyPageEnvelope() throws Exception {
    assertLegacyPage("/api/prompts?pageNo=1&pageSize=10");
    assertLegacyPage("/api/dataset/datasets?pageNumber=1&pageSize=10");
    assertLegacyPage("/api/evaluator/evaluators?pageNumber=1&pageSize=10");
    assertLegacyPage("/api/experiments?pageNumber=1&pageSize=10");
    assertLegacyPage("/api/models");
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
    String content =
        mockMvc
            .perform(get(url))
            .andExpect(status().isOk())
            .andReturn()
            .getResponse()
            .getContentAsString();
    return objectMapper.readTree(content);
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
