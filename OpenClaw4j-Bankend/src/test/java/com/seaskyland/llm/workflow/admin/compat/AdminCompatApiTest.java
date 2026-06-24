package com.seaskyland.llm.workflow.admin.compat;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.seaskyland.llm.workflow.core.base.entity.AppEntity;
import com.seaskyland.llm.workflow.core.base.mapper.AppMapper;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.webmvc.test.autoconfigure.AutoConfigureMockMvc;
import org.springframework.http.MediaType;
import org.springframework.jdbc.core.JdbcTemplate;
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

	@Autowired
	private JdbcTemplate jdbcTemplate;

	@Autowired
	private AppMapper appMapper;

	@Test
	void sqliteDateHandlerReadsEpochMillisTextFromApplicationRows() {
		jdbcTemplate.update("""
				INSERT INTO application (
				  workspace_id, app_id, name, source, type, status,
				  gmt_create, gmt_modified, creator, modifier
				)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
				""",
				"test-workspace", "epoch-millis-app", "epoch millis app", "test", "basic", 1,
				"1782306319293", "1782306319293", "10000", "10000");

		AppEntity app = appMapper
			.selectOne(new LambdaQueryWrapper<AppEntity>().eq(AppEntity::getAppId, "epoch-millis-app"));

		assertThat(app.getGmtCreate()).isNotNull();
		assertThat(app.getGmtCreate().getTime()).isEqualTo(1782306319293L);
	}

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
