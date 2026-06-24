package com.seaskyland.llm.workflow.admin.compat;

import com.seaskyland.llm.workflow.runtime.domain.Result;
import org.springframework.web.bind.annotation.*;

import java.util.List;
import java.util.Map;

@RestController
public class AdminCompatController {

	private final AdminCompatService service;

	public AdminCompatController(AdminCompatService service) {
		this.service = service;
	}

	@GetMapping("/api/prompts")
	public Result<AdminPage<Map<String, Object>>> prompts(
			@RequestParam(defaultValue = "1") long pageNo, @RequestParam(defaultValue = "10") long pageSize) {
		return Result.success(service.list("prompt", "update_time", pageNo, pageSize));
	}

	@GetMapping("/api/models")
	public Result<AdminPage<Map<String, Object>>> models(
			@RequestParam(defaultValue = "1") long page, @RequestParam(defaultValue = "10") long size) {
		return Result.success(service.list("model", "gmt_modified", page, size));
	}

	@GetMapping("/api/prompt")
	public Result<Map<String, Object>> prompt(@RequestParam String promptKey) {
		return Result.success(service.getPrompt(promptKey));
	}

	@PostMapping("/api/prompt")
	public Result<Map<String, Object>> createPrompt(@RequestBody Map<String, Object> body) {
		return Result.success(service.createPrompt(body));
	}

	@PutMapping("/api/prompt")
	public Result<Map<String, Object>> updatePrompt(@RequestBody Map<String, Object> body) {
		return Result.success(service.updatePrompt(body));
	}

	@DeleteMapping("/api/prompt")
	public Result<Boolean> deletePrompt(@RequestParam String promptKey) {
		service.deletePrompt(promptKey);
		return Result.success(true);
	}

	@GetMapping("/api/prompt/versions")
	public Result<AdminPage<Map<String, Object>>> promptVersions(
			@RequestParam(defaultValue = "1") long pageNo, @RequestParam(defaultValue = "10") long pageSize) {
		return Result.success(service.list("prompt_version", "update_time", pageNo, pageSize));
	}

	@GetMapping("/api/prompt/version")
	public Result<Map<String, Object>> promptVersion(@RequestParam String promptKey,
			@RequestParam String version) {
		return Result.success(service.getPromptVersion(promptKey, version));
	}

	@PostMapping("/api/prompt/version")
	public Result<Map<String, Object>> createPromptVersion(@RequestBody Map<String, Object> body) {
		return Result.success(service.createPromptVersion(body));
	}

	@GetMapping("/api/prompt/templates")
	public Result<AdminPage<Map<String, Object>>> promptTemplates(
			@RequestParam(defaultValue = "1") long pageNo, @RequestParam(defaultValue = "10") long pageSize) {
		return Result.success(service.list("prompt_build_template", "id", pageNo, pageSize));
	}

	@GetMapping("/api/prompt/template")
	public Result<Map<String, Object>> promptTemplate(@RequestParam String promptTemplateKey) {
		return Result.success(service.getPromptTemplate(promptTemplateKey));
	}

	@PostMapping("/api/prompt/run")
	public Result<Map<String, Object>> runPrompt(@RequestBody Map<String, Object> body) {
		String sessionId = String.valueOf(body.getOrDefault("sessionId", java.util.UUID.randomUUID().toString()));
		return Result.success(Map.of("sessionId", sessionId, "type", "message", "content", "",
				"messageCount", 0));
	}

	@GetMapping("/api/prompt/session")
	public Result<Map<String, Object>> promptSession(@RequestParam String sessionId) {
		return Result.success(Map.of("sessionId", sessionId, "messages", List.of(), "messageCount", 0));
	}

	@DeleteMapping("/api/prompt/session")
	public Result<Boolean> deletePromptSession(@RequestParam String sessionId) {
		return Result.success(true);
	}

	@GetMapping("/api/dataset/datasets")
	public Result<AdminPage<Map<String, Object>>> datasets(
			@RequestParam(defaultValue = "1") long pageNumber, @RequestParam(defaultValue = "10") long pageSize) {
		return Result.success(service.list("dataset", "update_time", pageNumber, pageSize));
	}

	@GetMapping("/api/dataset/dataset")
	public Result<Map<String, Object>> dataset(@RequestParam Long datasetId) {
		return Result.success(service.getById("dataset", datasetId));
	}

	@PostMapping("/api/dataset/dataset")
	public Result<Map<String, Object>> createDataset(@RequestBody Map<String, Object> body) {
		Map<String, Object> data = service.createNamed("dataset", body);
		return Result.success(data);
	}

	@PutMapping("/api/dataset/dataset")
	public Result<Map<String, Object>> updateDataset(@RequestBody Map<String, Object> body) {
		Long id = Long.valueOf(String.valueOf(body.get("datasetId")));
		return Result.success(service.updateNamed("dataset", id, body));
	}

	@DeleteMapping("/api/dataset/dataset")
	public Result<Boolean> deleteDataset(@RequestParam Long datasetId) {
		service.deleteById("dataset", datasetId);
		return Result.success(true);
	}

	@GetMapping("/api/dataset/datasetVersions")
	public Result<AdminPage<Map<String, Object>>> datasetVersions(
			@RequestParam(defaultValue = "1") long pageNumber, @RequestParam(defaultValue = "10") long pageSize) {
		return Result.success(service.list("dataset_version", "update_time", pageNumber, pageSize));
	}

	@PostMapping("/api/dataset/datasetVersion")
	public Result<Map<String, Object>> createDatasetVersion(@RequestBody Map<String, Object> body) {
		return Result.success(service.createDatasetVersion(body));
	}

	@PutMapping("/api/dataset/datasetVersion")
	public Result<Map<String, Object>> updateDatasetVersion(@RequestBody Map<String, Object> body) {
		return Result.success(service.getById("dataset_version", Long.valueOf(String.valueOf(body.get("id")))));
	}

	@GetMapping("/api/dataset/dataItems")
	public Result<AdminPage<Map<String, Object>>> dataItems(
			@RequestParam(defaultValue = "1") long pageNumber, @RequestParam(defaultValue = "10") long pageSize) {
		return Result.success(service.list("dataset_item", "update_time", pageNumber, pageSize));
	}

	@PostMapping("/api/dataset/dataItem")
	public Result<List<Map<String, Object>>> createDataItem() {
		return Result.success(List.of());
	}

	@GetMapping("/api/dataset/dataItem")
	public Result<Map<String, Object>> dataItem(@RequestParam Long id) {
		return Result.success(service.getById("dataset_item", id));
	}

	@PutMapping("/api/dataset/dataItem")
	public Result<Map<String, Object>> updateDataItem(@RequestBody Map<String, Object> body) {
		return Result.success(service.getById("dataset_item", Long.valueOf(String.valueOf(body.get("id")))));
	}

	@DeleteMapping({"/api/dataset/dataItem", "/api/dataset/dataItems"})
	public Result<Boolean> deleteDataItem() {
		return Result.success(true);
	}

	@PostMapping("/api/dataset/dataItemFromTrace")
	public Result<List<Map<String, Object>>> dataItemFromTrace() {
		return Result.success(List.of());
	}

	@GetMapping("/api/dataset/experiments")
	public Result<AdminPage<Map<String, Object>>> datasetExperiments(
			@RequestParam(defaultValue = "1") long pageNumber, @RequestParam(defaultValue = "10") long pageSize) {
		return Result.success(AdminPage.of(0, pageNumber, pageSize, List.of()));
	}

	@GetMapping("/api/evaluator/evaluators")
	public Result<AdminPage<Map<String, Object>>> evaluators(
			@RequestParam(defaultValue = "1") long pageNumber, @RequestParam(defaultValue = "10") long pageSize) {
		return Result.success(service.list("evaluator", "update_time", pageNumber, pageSize));
	}

	@GetMapping("/api/evaluator/evaluator")
	public Result<Map<String, Object>> evaluator(@RequestParam Long id) {
		return Result.success(service.getById("evaluator", id));
	}

	@PostMapping("/api/evaluator/evaluator")
	public Result<Map<String, Object>> createEvaluator(@RequestBody Map<String, Object> body) {
		return Result.success(service.createNamed("evaluator", body));
	}

	@PutMapping("/api/evaluator/evaluator")
	public Result<Map<String, Object>> updateEvaluator(@RequestBody Map<String, Object> body) {
		Long id = Long.valueOf(String.valueOf(body.get("id")));
		return Result.success(service.updateNamed("evaluator", id, body));
	}

	@DeleteMapping("/api/evaluator/evaluator")
	public Result<Boolean> deleteEvaluator(@RequestParam Long id) {
		service.deleteById("evaluator", id);
		return Result.success(true);
	}

	@GetMapping("/api/evaluator/evaluatorVersions")
	public Result<AdminPage<Map<String, Object>>> evaluatorVersions(
			@RequestParam(defaultValue = "1") long pageNumber, @RequestParam(defaultValue = "10") long pageSize) {
		return Result.success(service.list("evaluator_version", "update_time", pageNumber, pageSize));
	}

	@PostMapping("/api/evaluator/evaluatorVersion")
	public Result<Map<String, Object>> createEvaluatorVersion(@RequestBody Map<String, Object> body) {
		return Result.success(service.createEvaluatorVersion(body));
	}

	@GetMapping("/api/evaluator/templates")
	public Result<AdminPage<Map<String, Object>>> evaluatorTemplates() {
		return Result.success(service.list("evaluator_template", "id", 1, 100));
	}

	@GetMapping("/api/evaluator/template")
	public Result<Map<String, Object>> evaluatorTemplate(@RequestParam Long templateId) {
		return Result.success(service.getEvaluatorTemplate(templateId));
	}

	@PostMapping("/api/evaluator/debug")
	public Result<Map<String, Object>> debugEvaluator() {
		return Result.success(Map.of("score", 0, "reason", "No evaluator engine is configured", "evaluationTime", ""));
	}

	@GetMapping("/api/evaluator/experiments")
	public Result<AdminPage<Map<String, Object>>> evaluatorExperiments(
			@RequestParam(defaultValue = "1") long pageNumber, @RequestParam(defaultValue = "10") long pageSize) {
		return Result.success(AdminPage.of(0, pageNumber, pageSize, List.of()));
	}

	@GetMapping("/api/experiments")
	public Result<AdminPage<Map<String, Object>>> experiments(
			@RequestParam(defaultValue = "1") long pageNumber, @RequestParam(defaultValue = "10") long pageSize) {
		return Result.success(service.list("experiment", "update_time", pageNumber, pageSize));
	}

	@GetMapping("/api/experiment")
	public Result<Map<String, Object>> experiment(@RequestParam Long experimentId) {
		return Result.success(service.getById("experiment", experimentId));
	}

	@PostMapping("/api/experiment")
	public Result<Map<String, Object>> createExperiment(@RequestBody Map<String, Object> body) {
		return Result.success(service.createExperiment(body));
	}

	@PutMapping("/api/experiment/stop")
	public Result<Map<String, Object>> stopExperiment(@RequestParam Long experimentId) {
		return Result.success(service.getById("experiment", experimentId));
	}

	@PutMapping("/api/experiment/restart")
	public Result<Boolean> restartExperiment() {
		return Result.success(true);
	}

	@DeleteMapping("/api/experiment")
	public Result<Boolean> deleteExperiment(@RequestParam Long experimentId) {
		service.deleteById("experiment", experimentId);
		return Result.success(true);
	}

	@GetMapping("/api/experiment/results")
	public Result<List<Map<String, Object>>> experimentResults() {
		return Result.success(List.of());
	}

	@GetMapping("/api/experiment/result")
	public Result<AdminPage<Map<String, Object>>> experimentResult(
			@RequestParam(defaultValue = "1") long pageNumber, @RequestParam(defaultValue = "10") long pageSize) {
		return Result.success(AdminPage.of(0, pageNumber, pageSize, List.of()));
	}

}
