package com.seaskyland.llm.workflow.admin.compat;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.webmvc.test.autoconfigure.AutoConfigureMockMvc;
import org.springframework.http.MediaType;
import org.springframework.test.context.TestPropertySource;
import org.springframework.test.web.servlet.MockMvc;

import static org.assertj.core.api.Assertions.assertThat;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@SpringBootTest
@AutoConfigureMockMvc
@TestPropertySource(properties = {
		"spring.datasource.url=jdbc:sqlite:file:admin-compat-test?mode=memory&cache=shared",
		"spring.sql.init.mode=always"
})
class AdminCompatApiTest {

	@Autowired
	private MockMvc mockMvc;

	@Autowired
	private ObjectMapper objectMapper;

	@Test
	void adminListApisReturnLegacyPageEnvelope() throws Exception {
		assertLegacyPage("/api/prompts?pageNo=1&pageSize=10");
		assertLegacyPage("/api/dataset/datasets?pageNumber=1&pageSize=10");
		assertLegacyPage("/api/evaluator/evaluators?pageNumber=1&pageSize=10");
		assertLegacyPage("/api/experiments?pageNumber=1&pageSize=10");
		assertLegacyPage("/api/models");
	}

	@Test
	void observabilityStartsEmptyAndCanQueryInMemorySpans() throws Exception {
		JsonNode emptyTraces = getJson("/api/observability/traces?pageNumber=1&pageSize=10");
		assertThat(emptyTraces.has("code")).isFalse();
		assertThat(emptyTraces.path("data").path("totalCount").asLong()).isZero();
		assertThat(emptyTraces.path("data").path("pageItems")).isEmpty();

		mockMvc.perform(post("/api/observability/traces")
				.contentType(MediaType.APPLICATION_JSON)
				.content("""
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
		assertThat(traces.path("data").path("pageItems").get(0).path("traceId").asText()).isEqualTo("trace-1");

		JsonNode services = getJson("/api/observability/services");
		assertThat(services.path("data").path("services").get(0).path("name").asText()).isEqualTo("openclaw4j");
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
		String content = mockMvc.perform(get(url))
			.andExpect(status().isOk())
			.andReturn()
			.getResponse()
			.getContentAsString();
		return objectMapper.readTree(content);
	}

}
