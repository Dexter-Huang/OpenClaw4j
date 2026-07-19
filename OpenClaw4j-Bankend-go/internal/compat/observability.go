package compat

import (
	"fmt"
	"sort"
	"strconv"
	"sync"
)

// ObservabilityService keeps the Java compatibility endpoint's process-local trace buffer.
// It deliberately is not persisted because the original Controller used CopyOnWriteArrayList.
type ObservabilityService struct {
	mu    sync.RWMutex
	spans []map[string]any
}

func NewObservabilityService() *ObservabilityService {
	return &ObservabilityService{spans: make([]map[string]any, 0)}
}

func (s *ObservabilityService) AddTrace(span map[string]any) map[string]any {
	normalized := map[string]any{
		"traceId":      stringValue(span, "traceId"),
		"spanId":       stringValue(span, "spanId"),
		"parentSpanId": stringValue(span, "parentSpanId"),
		"durationNs":   numberValue(span, "durationNs"),
		"spanKind":     valueOr(span, "spanKind", "SPAN_KIND_INTERNAL"),
		"service":      valueOr(span, "service", "openclaw4j"),
		"spanName":     valueOr(span, "spanName", "unknown"),
		"startTime":    valueOr(span, "startTime", ""),
		"endTime":      valueOr(span, "endTime", ""),
		"status":       valueOr(span, "status", "Ok"),
		"errorCount":   numberValue(span, "errorCount"),
		"attributes":   mapValue(span["attributes"]),
		"resources":    mapValue(span["resources"]),
		"spanLinks":    listValue(span["spanLinks"]),
		"spanEvents":   listValue(span["spanEvents"]),
	}
	s.mu.Lock()
	s.spans = append(s.spans, normalized)
	s.mu.Unlock()
	return copyMap(normalized)
}

func (s *ObservabilityService) Traces(pageNumber, pageSize int64, service, spanName string) *Page {
	spans := s.snapshot()
	filtered := make([]map[string]any, 0, len(spans))
	for _, span := range spans {
		if (service == "" || service == fmt.Sprint(span["service"])) && (spanName == "" || spanName == fmt.Sprint(span["spanName"])) {
			filtered = append(filtered, span)
		}
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		return fmt.Sprint(filtered[i]["startTime"]) > fmt.Sprint(filtered[j]["startTime"])
	})
	return pageOf(filtered, pageNumber, pageSize)
}

func (s *ObservabilityService) TraceDetail(traceID string) map[string]any {
	spans := s.snapshot()
	records := make([]map[string]any, 0)
	for _, span := range spans {
		if traceID == fmt.Sprint(span["traceId"]) {
			records = append(records, span)
		}
	}
	sort.SliceStable(records, func(i, j int) bool {
		return fmt.Sprint(records[i]["startTime"]) < fmt.Sprint(records[j]["startTime"])
	})
	return map[string]any{"records": records}
}

func (s *ObservabilityService) Services() map[string]any {
	grouped := make(map[string][]string)
	order := make([]string, 0)
	for _, span := range s.snapshot() {
		service := fmt.Sprint(span["service"])
		if _, ok := grouped[service]; !ok {
			order = append(order, service)
		}
		grouped[service] = append(grouped[service], fmt.Sprint(span["spanName"]))
	}
	services := make([]map[string]any, 0, len(order))
	for _, service := range order {
		seen := make(map[string]bool)
		operations := make([]string, 0)
		for _, operation := range grouped[service] {
			if !seen[operation] {
				seen[operation] = true
				operations = append(operations, operation)
			}
		}
		services = append(services, map[string]any{"name": service, "operations": operations})
	}
	return map[string]any{"services": services}
}

func (s *ObservabilityService) Overview(detail bool) map[string]any {
	spans := s.snapshot()
	return map[string]any{
		"span.count":      groupedMetric("spanName", spans, detail),
		"operation.count": groupedMetric("spanName", spans, detail),
		"usage.tokens":    tokenMetric(spans, detail),
	}
}

func (s *ObservabilityService) snapshot() []map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]map[string]any, len(s.spans))
	for index, span := range s.spans {
		result[index] = copyMap(span)
	}
	return result
}

func pageOf(items []map[string]any, pageNumber, pageSize int64) *Page {
	pageNumber, pageSize = normalizePage(pageNumber, pageSize)
	from := (pageNumber - 1) * pageSize
	if from > int64(len(items)) {
		from = int64(len(items))
	}
	to := from + pageSize
	if to > int64(len(items)) {
		to = int64(len(items))
	}
	total := int64(len(items))
	pages := int64(0)
	if total > 0 {
		pages = (total + pageSize - 1) / pageSize
	}
	return &Page{TotalCount: total, TotalPage: pages, PageNumber: pageNumber, PageSize: pageSize, PageItems: items[from:to]}
}

func groupedMetric(key string, spans []map[string]any, detail bool) map[string]any {
	metric := map[string]any{"total": len(spans), "detail": []map[string]any{}}
	if !detail {
		return metric
	}
	counts := make(map[string]int64)
	order := make([]string, 0)
	for _, span := range spans {
		value := fmt.Sprint(span[key])
		if _, ok := counts[value]; !ok {
			order = append(order, value)
		}
		counts[value]++
	}
	items := make([]map[string]any, 0, len(order))
	for _, value := range order {
		items = append(items, map[string]any{key: value, "total": counts[value]})
	}
	metric["detail"] = items
	return metric
}

func tokenMetric(spans []map[string]any, detail bool) map[string]any {
	var total int64
	counts := make(map[string]int64)
	order := make([]string, 0)
	for _, span := range spans {
		tokens := tokens(span)
		total += tokens
		if !detail {
			continue
		}
		model := modelName(span)
		if _, ok := counts[model]; !ok {
			order = append(order, model)
		}
		counts[model] += tokens
	}
	metric := map[string]any{"total": total, "detail": []map[string]any{}}
	if detail {
		items := make([]map[string]any, 0, len(order))
		for _, model := range order {
			items = append(items, map[string]any{"modelName": model, "total": counts[model]})
		}
		metric["detail"] = items
	}
	return metric
}

func stringValue(data map[string]any, key string) string {
	if data[key] == nil {
		return ""
	}
	return fmt.Sprint(data[key])
}

func valueOr(data map[string]any, key string, fallback any) any {
	if data[key] == nil {
		return fallback
	}
	return data[key]
}

func numberValue(data map[string]any, key string) int64 {
	value, err := strconv.ParseInt(fmt.Sprint(data[key]), 10, 64)
	if err != nil {
		return 0
	}
	return value
}

func mapValue(value any) map[string]any {
	if data, ok := value.(map[string]any); ok {
		return copyMap(data)
	}
	return map[string]any{}
}

func listValue(value any) []map[string]any {
	list, ok := value.([]any)
	if !ok {
		return []map[string]any{}
	}
	result := make([]map[string]any, 0, len(list))
	for _, item := range list {
		if data, ok := item.(map[string]any); ok {
			result = append(result, copyMap(data))
		}
	}
	return result
}

func tokens(span map[string]any) int64 {
	attributes := mapValue(span["attributes"])
	if attributes["usage.total_tokens"] != nil {
		return numberValue(attributes, "usage.total_tokens")
	}
	return numberValue(attributes, "usage.tokens")
}

func modelName(span map[string]any) string {
	attributes := mapValue(span["attributes"])
	if attributes["model.name"] != nil {
		return fmt.Sprint(attributes["model.name"])
	}
	return "unknown"
}

func copyMap(source map[string]any) map[string]any {
	result := make(map[string]any, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
