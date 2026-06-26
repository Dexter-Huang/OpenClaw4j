package com.seaskyland.llm.workflow.admin.compat;

import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.jdbc.support.GeneratedKeyHolder;
import org.springframework.jdbc.support.KeyHolder;
import org.springframework.stereotype.Service;
import org.springframework.util.StringUtils;

import java.sql.PreparedStatement;
import java.sql.Statement;
import java.util.List;
import java.util.Map;

@Service
public class AdminCompatService {

	private final JdbcTemplate jdbcTemplate;

	public AdminCompatService(JdbcTemplate jdbcTemplate) {
		this.jdbcTemplate = jdbcTemplate;
	}

	public AdminPage<Map<String, Object>> list(String table, String orderColumn, long pageNumber,
			long pageSize) {
		long safePageNumber = pageNumber <= 0 ? 1 : pageNumber;
		long safePageSize = pageSize <= 0 ? 10 : pageSize;
		long offset = (safePageNumber - 1) * safePageSize;
		Long total = jdbcTemplate.queryForObject("SELECT COUNT(1) FROM " + table, Long.class);
		List<Map<String, Object>> rows = jdbcTemplate.queryForList(
				"SELECT * FROM " + table + " ORDER BY " + orderColumn + " DESC LIMIT ? OFFSET ?", safePageSize,
				offset);
		return AdminPage.of(total == null ? 0 : total, safePageNumber, safePageSize, rows.stream()
			.map(this::camelize)
			.toList());
	}

	public Map<String, Object> getById(String table, Long id) {
		List<Map<String, Object>> rows = jdbcTemplate.queryForList("SELECT * FROM " + table + " WHERE id = ?", id);
		return rows.isEmpty() ? null : camelize(rows.get(0));
	}

	public Map<String, Object> getPrompt(String promptKey) {
		List<Map<String, Object>> rows = jdbcTemplate.queryForList("SELECT * FROM prompt WHERE prompt_key = ?",
				promptKey);
		return rows.isEmpty() ? null : camelize(rows.get(0));
	}

	public void deleteById(String table, Long id) {
		jdbcTemplate.update("DELETE FROM " + table + " WHERE id = ?", id);
	}

	public void deletePrompt(String promptKey) {
		jdbcTemplate.update("DELETE FROM prompt WHERE prompt_key = ?", promptKey);
	}

	public Map<String, Object> createPrompt(Map<String, Object> body) {
		String promptKey = string(body, "promptKey");
		String description = string(body, "promptDescription");
		String tags = string(body, "tags");
		int updated = jdbcTemplate.update("""
				UPDATE prompt
				SET prompt_desc = ?, prompt_description = ?, tags = ?, update_time = CURRENT_TIMESTAMP
				WHERE prompt_key = ?
				""", description, description, tags, promptKey);
		if (updated == 0) {
			insertAndReturnId("""
					INSERT INTO prompt (prompt_key, prompt_desc, prompt_description, latest_version, tags, create_time, update_time)
					VALUES (?, ?, ?, '1.0.0', ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
					""", promptKey, description, description, tags);
		}
		return getPrompt(promptKey);
	}

	public Map<String, Object> getPromptVersion(String promptKey, String version) {
		List<Map<String, Object>> rows = jdbcTemplate.queryForList(
				"SELECT * FROM prompt_version WHERE prompt_key = ? AND version = ?", promptKey, version);
		return rows.isEmpty() ? null : camelize(rows.get(0));
	}

	public Map<String, Object> createPromptVersion(Map<String, Object> body) {
		String promptKey = string(body, "promptKey");
		String version = string(body, "version");
		if (!StringUtils.hasText(version)) {
			version = nextVersion("prompt_version", "prompt_key", promptKey);
		}
		int updated = jdbcTemplate.update("""
				UPDATE prompt_version
				SET version_description = ?, template = ?, variables = ?, model_config = ?, status = ?, update_time = CURRENT_TIMESTAMP
				WHERE prompt_key = ? AND version = ?
				""", string(body, "versionDescription"), string(body, "template"), string(body, "variables"),
				string(body, "modelConfig"), string(body, "status"), promptKey, version);
		if (updated == 0) {
			insertAndReturnId("""
					INSERT INTO prompt_version
					(prompt_key, version, version_description, template, variables, model_config, previous_version, status, create_time, update_time)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
					""", promptKey, version, string(body, "versionDescription"), string(body, "template"),
					string(body, "variables"), string(body, "modelConfig"), string(body, "previousVersion"),
					string(body, "status"));
		}
		jdbcTemplate.update("""
				UPDATE prompt SET latest_version = ?, latest_version_status = ?, update_time = CURRENT_TIMESTAMP
				WHERE prompt_key = ?
				""", version, string(body, "status"), promptKey);
		return getPromptVersion(promptKey, version);
	}

	public Map<String, Object> getPromptTemplate(String promptTemplateKey) {
		List<Map<String, Object>> rows = jdbcTemplate.queryForList(
				"SELECT * FROM prompt_build_template WHERE prompt_template_key = ?", promptTemplateKey);
		return rows.isEmpty() ? null : camelize(rows.get(0));
	}

	public Map<String, Object> getEvaluatorTemplate(Long templateId) {
		return getById("evaluator_template", templateId);
	}

	public Map<String, Object> updatePrompt(Map<String, Object> body) {
		String promptKey = string(body, "promptKey");
		jdbcTemplate.update(
				"UPDATE prompt SET prompt_desc = ?, prompt_description = ?, tags = ?, update_time = CURRENT_TIMESTAMP WHERE prompt_key = ?",
				string(body, "promptDescription"), string(body, "promptDescription"), string(body, "tags"), promptKey);
		return getPrompt(promptKey);
	}

	public Map<String, Object> createNamed(String table, Map<String, Object> body) {
		Long id = insertAndReturnId("INSERT INTO " + table
				+ " (name, description, create_time, update_time) VALUES (?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)",
				string(body, "name"), string(body, "description"));
		return getById(table, id);
	}

	public Map<String, Object> createDatasetVersion(Map<String, Object> body) {
		Long datasetId = longValue(body, "datasetId");
		String version = nextVersion("dataset_version", "dataset_id", datasetId);
		Long id = insertAndReturnId("""
				INSERT INTO dataset_version
				(dataset_id, version, description, data_count, status, dataset_items, columns_config, create_time, update_time)
				VALUES (?, ?, ?, 0, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
				""", datasetId, version, string(body, "description"), string(body, "status"), jsonString(body, "datasetItems"),
				jsonString(body, "columnsConfig"));
		return getById("dataset_version", id);
	}

	public Map<String, Object> createEvaluatorVersion(Map<String, Object> body) {
		Long evaluatorId = longValue(body, "evaluatorId");
		String version = StringUtils.hasText(string(body, "version")) ? string(body, "version")
				: nextVersion("evaluator_version", "evaluator_id", evaluatorId);
		Long id = insertAndReturnId("""
				INSERT INTO evaluator_version
				(evaluator_id, description, version, model_config, prompt, variables, status, create_time, update_time)
				VALUES (?, ?, ?, ?, ?, ?, 'draft', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
				""", evaluatorId, string(body, "description"), version, string(body, "modelConfig"), string(body, "prompt"),
				string(body, "variables"));
		jdbcTemplate.update("""
				UPDATE evaluator SET latest_version = ?, prompt = ?, model_config = ?, variables = ?, update_time = CURRENT_TIMESTAMP
				WHERE id = ?
				""", version, string(body, "prompt"), string(body, "modelConfig"), string(body, "variables"), evaluatorId);
		return getById("evaluator_version", id);
	}

	public Map<String, Object> createExperiment(Map<String, Object> body) {
		Long id = insertAndReturnId("""
				INSERT INTO experiment
				(name, description, dataset_id, dataset_version_id, dataset_version, evaluation_object_config, evaluator_config, status, progress, create_time, update_time)
				VALUES (?, ?, ?, ?, ?, ?, ?, 'created', 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
				""", string(body, "name"), string(body, "description"), longValue(body, "datasetId"),
				longValue(body, "datasetVersionId"), string(body, "datasetVersion"),
				string(body, "evaluationObjectConfig"), string(body, "evaluatorConfig"));
		return getById("experiment", id);
	}

	public Map<String, Object> updateNamed(String table, Long id, Map<String, Object> body) {
		jdbcTemplate.update("UPDATE " + table + " SET name = ?, description = ?, update_time = CURRENT_TIMESTAMP WHERE id = ?",
				string(body, "name"), string(body, "description"), id);
		return getById(table, id);
	}

	private Map<String, Object> camelize(Map<String, Object> row) {
		Map<String, Object> data = new java.util.LinkedHashMap<>();
		row.forEach((key, value) -> data.put(toCamel(key), value));
		copyAlias(data, "promptDesc", "promptDescription");
		copyAlias(data, "templateDesc", "templateDescription");
		return data;
	}

	private void copyAlias(Map<String, Object> data, String from, String to) {
		if (data.containsKey(from) && (!data.containsKey(to) || data.get(to) == null || "".equals(data.get(to)))) {
			data.put(to, data.get(from));
		}
	}

	private String toCamel(String key) {
		String lower = key.toLowerCase();
		StringBuilder result = new StringBuilder();
		boolean upper = false;
		for (char ch : lower.toCharArray()) {
			if (ch == '_') {
				upper = true;
			}
			else if (upper) {
				result.append(Character.toUpperCase(ch));
				upper = false;
			}
			else {
				result.append(ch);
			}
		}
		return result.toString();
	}

	private String string(Map<String, Object> body, String key) {
		Object value = body.get(key);
		return StringUtils.hasText(value == null ? null : value.toString()) ? value.toString() : "";
	}

	private Long longValue(Map<String, Object> body, String key) {
		Object value = body.get(key);
		if (value == null) {
			return null;
		}
		return Long.valueOf(String.valueOf(value));
	}

	private String jsonString(Map<String, Object> body, String key) {
		Object value = body.get(key);
		if (value == null) {
			return "[]";
		}
		return value instanceof String ? value.toString()
				: com.seaskyland.llm.workflow.runtime.utils.JsonUtils.toJson(value);
	}

	private Long insertAndReturnId(String sql, Object... args) {
		KeyHolder keyHolder = new GeneratedKeyHolder();
		jdbcTemplate.update(connection -> {
			PreparedStatement statement = connection.prepareStatement(sql, Statement.RETURN_GENERATED_KEYS);
			for (int i = 0; i < args.length; i++) {
				statement.setObject(i + 1, args[i]);
			}
			return statement;
		}, keyHolder);
		Number key = keyHolder.getKeyList().stream()
			.findFirst()
			.map(keys -> keys.getOrDefault("id", keys.get("ID")))
			.filter(Number.class::isInstance)
			.map(Number.class::cast)
			.orElseGet(keyHolder::getKey);
		if (key == null) {
			throw new IllegalStateException("Insert did not return a generated id");
		}
		return key.longValue();
	}

	private String nextVersion(String table, String ownerColumn, Long ownerId) {
		Long count = jdbcTemplate.queryForObject("SELECT COUNT(1) FROM " + table + " WHERE " + ownerColumn + " = ?",
				Long.class, ownerId);
		return "1.0." + ((count == null ? 0 : count) + 1);
	}

	private String nextVersion(String table, String ownerColumn, String ownerId) {
		Long count = jdbcTemplate.queryForObject("SELECT COUNT(1) FROM " + table + " WHERE " + ownerColumn + " = ?",
				Long.class, ownerId);
		return "1.0." + ((count == null ? 0 : count) + 1);
	}

}
