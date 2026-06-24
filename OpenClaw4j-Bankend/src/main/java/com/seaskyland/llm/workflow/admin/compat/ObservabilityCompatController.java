package com.seaskyland.llm.workflow.admin.compat;

import com.seaskyland.llm.workflow.runtime.domain.Result;
import org.springframework.web.bind.annotation.*;

import java.util.ArrayList;
import java.util.Comparator;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.CopyOnWriteArrayList;
import java.util.stream.Collectors;

@RestController
@RequestMapping("/api/observability")
public class ObservabilityCompatController {

	private final CopyOnWriteArrayList<Map<String, Object>> spans = new CopyOnWriteArrayList<>();

	@GetMapping("/traces")
	public Result<AdminPage<Map<String, Object>>> traces(
			@RequestParam(defaultValue = "1") long pageNumber, @RequestParam(defaultValue = "10") long pageSize,
			@RequestParam(required = false) String service, @RequestParam(required = false) String spanName) {
		List<Map<String, Object>> filtered = spans.stream()
			.filter(span -> service == null || service.equals(span.get("service")))
			.filter(span -> spanName == null || spanName.equals(span.get("spanName")))
			.sorted(Comparator.comparing(span -> String.valueOf(span.getOrDefault("startTime", "")),
					Comparator.reverseOrder()))
			.toList();
		int from = (int) Math.min(Math.max(pageNumber - 1, 0) * pageSize, filtered.size());
		int to = (int) Math.min(from + pageSize, filtered.size());
		return Result.success(AdminPage.of(filtered.size(), pageNumber, pageSize,
				new ArrayList<>(filtered.subList(from, to))));
	}

	@PostMapping("/traces")
	public Result<Map<String, Object>> addTrace(@RequestBody Map<String, Object> span) {
		Map<String, Object> normalized = new LinkedHashMap<>();
		normalized.put("traceId", string(span, "traceId"));
		normalized.put("spanId", string(span, "spanId"));
		normalized.put("parentSpanId", string(span, "parentSpanId"));
		normalized.put("durationNs", number(span, "durationNs"));
		normalized.put("spanKind", valueOr(span, "spanKind", "SPAN_KIND_INTERNAL"));
		normalized.put("service", valueOr(span, "service", "openclaw4j"));
		normalized.put("spanName", valueOr(span, "spanName", "unknown"));
		normalized.put("startTime", valueOr(span, "startTime", ""));
		normalized.put("endTime", valueOr(span, "endTime", ""));
		normalized.put("status", valueOr(span, "status", "Ok"));
		normalized.put("errorCount", number(span, "errorCount"));
		normalized.put("attributes", map(span.get("attributes")));
		normalized.put("resources", map(span.get("resources")));
		normalized.put("spanLinks", list(span.get("spanLinks")));
		normalized.put("spanEvents", list(span.get("spanEvents")));
		spans.add(normalized);
		return Result.success(normalized);
	}

	@GetMapping("/traces/{traceId}")
	public Result<Map<String, Object>> traceDetail(@PathVariable String traceId) {
		List<Map<String, Object>> records = spans.stream()
			.filter(span -> traceId.equals(span.get("traceId")))
			.sorted(Comparator.comparing(span -> String.valueOf(span.getOrDefault("startTime", ""))))
			.toList();
		return Result.success(Map.of("records", records));
	}

	@GetMapping("/services")
	public Result<Map<String, Object>> services() {
		List<Map<String, Object>> services = spans.stream()
			.collect(Collectors.groupingBy(span -> String.valueOf(span.get("service")), LinkedHashMap::new,
					Collectors.mapping(span -> String.valueOf(span.get("spanName")), Collectors.toCollection(ArrayList::new))))
			.entrySet()
			.stream()
			.map(entry -> Map.<String, Object>of("name", entry.getKey(), "operations",
					entry.getValue().stream().distinct().toList()))
			.toList();
		return Result.success(Map.of("services", services));
	}

	@GetMapping("/overview")
	public Result<Map<String, Object>> overview(@RequestParam(defaultValue = "false") boolean detail) {
		Map<String, Object> data = new LinkedHashMap<>();
		data.put("span.count", metric("spanName", spans, detail));
		data.put("operation.count", metric("spanName", spans, detail));
		data.put("usage.tokens", tokenMetric(detail));
		return Result.success(data);
	}

	private Map<String, Object> metric(String key, List<Map<String, Object>> data, boolean detail) {
		Map<String, Object> metric = new LinkedHashMap<>();
		metric.put("total", data.size());
		metric.put("detail", detail ? groupedCount(key, data) : List.of());
		return metric;
	}

	private Map<String, Object> tokenMetric(boolean detail) {
		long total = spans.stream().mapToLong(this::tokens).sum();
		Map<String, Object> metric = new LinkedHashMap<>();
		metric.put("total", total);
		metric.put("detail", detail ? spans.stream()
			.collect(Collectors.groupingBy(this::modelName, LinkedHashMap::new, Collectors.summingLong(this::tokens)))
			.entrySet()
			.stream()
			.map(entry -> {
				Map<String, Object> item = new LinkedHashMap<>();
				item.put("modelName", entry.getKey());
				item.put("total", entry.getValue());
				return item;
			})
			.toList() : List.of());
		return metric;
	}

	private List<Map<String, Object>> groupedCount(String key, List<Map<String, Object>> data) {
		return data.stream()
			.collect(Collectors.groupingBy(span -> String.valueOf(span.getOrDefault(key, "")), LinkedHashMap::new,
					Collectors.counting()))
			.entrySet()
			.stream()
			.map(entry -> {
				Map<String, Object> item = new LinkedHashMap<>();
				item.put(key, entry.getKey());
				item.put("total", entry.getValue());
				return item;
			})
			.toList();
	}

	private long tokens(Map<String, Object> span) {
		Object attributes = span.get("attributes");
		if (!(attributes instanceof Map<?, ?> map)) {
			return 0;
		}
		Object value = map.get("usage.total_tokens");
		if (value == null) {
			value = map.get("usage.tokens");
		}
		try {
			return value == null ? 0 : Long.parseLong(String.valueOf(value));
		}
		catch (NumberFormatException ignored) {
			return 0;
		}
	}

	private String modelName(Map<String, Object> span) {
		Object attributes = span.get("attributes");
		if (attributes instanceof Map<?, ?> map && map.get("model.name") != null) {
			return String.valueOf(map.get("model.name"));
		}
		return "unknown";
	}

	private String string(Map<String, Object> map, String key) {
		Object value = map.get(key);
		return value == null ? "" : String.valueOf(value);
	}

	private Object valueOr(Map<String, Object> map, String key, Object fallback) {
		Object value = map.get(key);
		return value == null ? fallback : value;
	}

	private long number(Map<String, Object> map, String key) {
		Object value = map.get(key);
		try {
			return value == null ? 0 : Long.parseLong(String.valueOf(value));
		}
		catch (NumberFormatException ignored) {
			return 0;
		}
	}

	@SuppressWarnings("unchecked")
	private Map<String, Object> map(Object value) {
		return value instanceof Map<?, ?> ? (Map<String, Object>) value : Map.of();
	}

	@SuppressWarnings("unchecked")
	private List<Map<String, Object>> list(Object value) {
		return value instanceof List<?> ? (List<Map<String, Object>>) value : List.of();
	}

}
