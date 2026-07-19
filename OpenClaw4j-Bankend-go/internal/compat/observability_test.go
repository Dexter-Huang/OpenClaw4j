package compat

import "testing"

func TestObservabilityServiceKeepsJavaCompatibilityShape(t *testing.T) {
	service := NewObservabilityService()
	service.AddTrace(map[string]any{
		"traceId": "trace-1", "spanId": "parent", "service": "catalog", "spanName": "search",
		"startTime": "2026-07-19T10:00:00Z", "attributes": map[string]any{"usage.total_tokens": 3, "model.name": "qwen"},
	})
	service.AddTrace(map[string]any{
		"traceId": "trace-1", "spanId": "child", "parentSpanId": "parent", "service": "catalog", "spanName": "rank",
		"startTime": "2026-07-19T10:00:01Z", "attributes": map[string]any{"usage.tokens": "5", "model.name": "qwen"},
	})

	traces := service.Traces(1, 10, "catalog", "")
	if traces.TotalCount != 2 || traces.PageItems[0]["spanId"] != "child" {
		t.Fatalf("unexpected trace list: %#v", traces)
	}
	if records := service.TraceDetail("trace-1")["records"].([]map[string]any); len(records) != 2 || records[0]["spanId"] != "parent" {
		t.Fatalf("unexpected trace details: %#v", records)
	}

	overview := service.Overview(true)
	if got := overview["usage.tokens"].(map[string]any)["total"]; got != int64(8) {
		t.Fatalf("expected total token count 8, got %#v", got)
	}
	services := service.Services()["services"].([]map[string]any)
	if len(services) != 1 || services[0]["name"] != "catalog" {
		t.Fatalf("unexpected services: %#v", services)
	}
}
