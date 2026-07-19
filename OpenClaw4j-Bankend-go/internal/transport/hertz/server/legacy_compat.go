package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/seaskyland/openclaw4j-backend-go/internal/compat"
)

// registerLegacyCompatibilityRoutes retains the unversioned endpoints served by the Java admin
// module. New console functionality remains under /console/v1; these paths exist for older UI
// pages and integrations that have not yet switched to the versioned API.
func registerLegacyCompatibilityRoutes(h *server.Hertz, options Options) {
	admin := options.LegacyAdminManager
	observability := options.ObservabilityManager

	h.GET("/api/prompts", legacyListHandler(admin, "prompt", "update_time", "pageNo", "pageSize"))
	h.GET("/api/prompt", legacyPromptGetHandler(admin))
	h.POST("/api/prompt", legacyPromptWriteHandler(admin))
	h.PUT("/api/prompt", legacyPromptWriteHandler(admin))
	h.DELETE("/api/prompt", legacyPromptDeleteHandler(admin))
	h.GET("/api/prompt/versions", legacyListHandler(admin, "prompt_version", "update_time", "pageNo", "pageSize"))
	h.GET("/api/prompt/version", legacyPromptVersionGetHandler(admin))
	h.POST("/api/prompt/version", legacyPromptVersionCreateHandler(admin))
	h.GET("/api/prompt/templates", legacyListHandler(admin, "prompt_build_template", "id", "pageNo", "pageSize"))
	h.GET("/api/prompt/template", legacyPromptTemplateHandler(admin))
	h.POST("/api/prompt/run", legacyStaticBodyHandler(func(body map[string]any) any {
		sessionID := strings.TrimSpace(textValue(body, "sessionId"))
		if sessionID == "" {
			sessionID = newRequestID()
		}
		return map[string]any{"sessionId": sessionID, "type": "message", "content": "", "messageCount": 0}
	}))
	h.GET("/api/prompt/session", legacySessionGetHandler)
	h.DELETE("/api/prompt/session", legacySuccessHandler(true))

	h.GET("/api/dataset/datasets", legacyListHandler(admin, "dataset", "update_time", "pageNumber", "pageSize"))
	h.GET("/api/dataset/dataset", legacyGetByIDHandler(admin, "dataset", "datasetId"))
	h.POST("/api/dataset/dataset", legacyNamedCreateHandler(admin, "dataset"))
	h.PUT("/api/dataset/dataset", legacyNamedUpdateHandler(admin, "dataset", "datasetId"))
	h.DELETE("/api/dataset/dataset", legacyDeleteByIDHandler(admin, "dataset", "datasetId"))
	h.GET("/api/dataset/datasetVersions", legacyListHandler(admin, "dataset_version", "update_time", "pageNumber", "pageSize"))
	h.POST("/api/dataset/datasetVersion", legacyDatasetVersionCreateHandler(admin))
	h.PUT("/api/dataset/datasetVersion", legacyBodyGetByIDHandler(admin, "dataset_version", "id"))
	h.GET("/api/dataset/dataItems", legacyListHandler(admin, "dataset_item", "update_time", "pageNumber", "pageSize"))
	h.POST("/api/dataset/dataItem", legacySuccessHandler([]map[string]any{}))
	h.GET("/api/dataset/dataItem", legacyGetByIDHandler(admin, "dataset_item", "id"))
	h.PUT("/api/dataset/dataItem", legacyBodyGetByIDHandler(admin, "dataset_item", "id"))
	h.DELETE("/api/dataset/dataItem", legacySuccessHandler(true))
	h.DELETE("/api/dataset/dataItems", legacySuccessHandler(true))
	h.POST("/api/dataset/dataItemFromTrace", legacySuccessHandler([]map[string]any{}))
	h.GET("/api/dataset/experiments", legacyEmptyPageHandler("pageNumber", "pageSize"))

	h.GET("/api/evaluator/evaluators", legacyListHandler(admin, "evaluator", "update_time", "pageNumber", "pageSize"))
	h.GET("/api/evaluator/evaluator", legacyGetByIDHandler(admin, "evaluator", "id"))
	h.POST("/api/evaluator/evaluator", legacyNamedCreateHandler(admin, "evaluator"))
	h.PUT("/api/evaluator/evaluator", legacyNamedUpdateHandler(admin, "evaluator", "id"))
	h.DELETE("/api/evaluator/evaluator", legacyDeleteByIDHandler(admin, "evaluator", "id"))
	h.GET("/api/evaluator/evaluatorVersions", legacyListHandler(admin, "evaluator_version", "update_time", "pageNumber", "pageSize"))
	h.POST("/api/evaluator/evaluatorVersion", legacyEvaluatorVersionCreateHandler(admin))
	h.GET("/api/evaluator/templates", legacyListHandler(admin, "evaluator_template", "id", "pageNumber", "pageSize"))
	h.GET("/api/evaluator/template", legacyGetByIDHandler(admin, "evaluator_template", "templateId"))
	h.POST("/api/evaluator/debug", legacySuccessHandler(map[string]any{"score": 0, "reason": "No evaluator engine is configured", "evaluationTime": ""}))
	h.GET("/api/evaluator/experiments", legacyEmptyPageHandler("pageNumber", "pageSize"))

	h.GET("/api/experiments", legacyListHandler(admin, "experiment", "update_time", "pageNumber", "pageSize"))
	h.GET("/api/experiment", legacyGetByIDHandler(admin, "experiment", "experimentId"))
	h.POST("/api/experiment", legacyExperimentCreateHandler(admin))
	h.PUT("/api/experiment/stop", legacyGetByIDHandler(admin, "experiment", "experimentId"))
	h.PUT("/api/experiment/restart", legacySuccessHandler(true))
	h.DELETE("/api/experiment", legacyDeleteByIDHandler(admin, "experiment", "experimentId"))
	h.GET("/api/experiment/results", legacySuccessHandler([]map[string]any{}))
	h.GET("/api/experiment/result", legacyEmptyPageHandler("pageNumber", "pageSize"))

	h.GET("/api/observability/traces", observabilityTracesHandler(observability))
	h.POST("/api/observability/traces", observabilityAddTraceHandler(observability))
	h.GET("/api/observability/traces/:traceId", observabilityTraceDetailHandler(observability))
	h.GET("/api/observability/services", observabilityServicesHandler(observability))
	h.GET("/api/observability/overview", observabilityOverviewHandler(observability))
}

func legacyListHandler(manager *compat.AdminService, table, order, pageKey, sizeKey string) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			legacyUnavailable(c)
			return
		}
		value, err := manager.List(ctx, table, order, legacyPage(c, pageKey), legacyPage(c, sizeKey))
		legacyResult(c, value, err)
	}
}

func legacyPromptGetHandler(manager *compat.AdminService) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		value, err := callPrompt(manager, ctx, c.Query("promptKey"))
		legacyResult(c, value, err)
	}
}

func legacyPromptVersionGetHandler(manager *compat.AdminService) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			legacyUnavailable(c)
			return
		}
		value, err := manager.GetPromptVersion(ctx, c.Query("promptKey"), c.Query("version"))
		legacyResult(c, value, err)
	}
}

func legacyPromptTemplateHandler(manager *compat.AdminService) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			legacyUnavailable(c)
			return
		}
		value, err := manager.GetPromptTemplate(ctx, c.Query("promptTemplateKey"))
		legacyResult(c, value, err)
	}
}

func legacyPromptWriteHandler(manager *compat.AdminService) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		body, ok := legacyBody(c)
		if !ok {
			return
		}
		if manager == nil {
			legacyUnavailable(c)
			return
		}
		value, err := manager.CreateOrUpdatePrompt(ctx, body)
		legacyResult(c, value, err)
	}
}

func legacyPromptVersionCreateHandler(manager *compat.AdminService) app.HandlerFunc {
	return legacyBodyCall(manager, func(ctx context.Context, body map[string]any) (any, error) {
		return manager.CreatePromptVersion(ctx, body)
	})
}
func legacyDatasetVersionCreateHandler(manager *compat.AdminService) app.HandlerFunc {
	return legacyBodyCall(manager, func(ctx context.Context, body map[string]any) (any, error) {
		return manager.CreateDatasetVersion(ctx, body)
	})
}
func legacyEvaluatorVersionCreateHandler(manager *compat.AdminService) app.HandlerFunc {
	return legacyBodyCall(manager, func(ctx context.Context, body map[string]any) (any, error) {
		return manager.CreateEvaluatorVersion(ctx, body)
	})
}
func legacyExperimentCreateHandler(manager *compat.AdminService) app.HandlerFunc {
	return legacyBodyCall(manager, func(ctx context.Context, body map[string]any) (any, error) {
		return manager.CreateExperiment(ctx, body)
	})
}

func legacyBodyCall(manager *compat.AdminService, call func(context.Context, map[string]any) (any, error)) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		body, ok := legacyBody(c)
		if !ok {
			return
		}
		if manager == nil {
			legacyUnavailable(c)
			return
		}
		value, err := call(ctx, body)
		legacyResult(c, value, err)
	}
}

func legacyPromptDeleteHandler(manager *compat.AdminService) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			legacyUnavailable(c)
			return
		}
		legacyResult(c, true, manager.DeletePrompt(ctx, c.Query("promptKey")))
	}
}

func legacyNamedCreateHandler(manager *compat.AdminService, table string) app.HandlerFunc {
	return legacyBodyCall(manager, func(ctx context.Context, body map[string]any) (any, error) {
		return manager.CreateNamed(ctx, table, body)
	})
}

func legacyNamedUpdateHandler(manager *compat.AdminService, table, idKey string) app.HandlerFunc {
	return legacyBodyCall(manager, func(ctx context.Context, body map[string]any) (any, error) {
		id, err := legacyMapID(body, idKey)
		if err != nil {
			return nil, err
		}
		return manager.UpdateNamed(ctx, table, id, body)
	})
}

func legacyGetByIDHandler(manager *compat.AdminService, table, idKey string) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			legacyUnavailable(c)
			return
		}
		id, err := legacyID(c.Query(idKey))
		if err != nil {
			legacyResult(c, nil, err)
			return
		}
		value, err := manager.GetByID(ctx, table, id)
		legacyResult(c, value, err)
	}
}

func legacyBodyGetByIDHandler(manager *compat.AdminService, table, idKey string) app.HandlerFunc {
	return legacyBodyCall(manager, func(ctx context.Context, body map[string]any) (any, error) {
		id, err := legacyMapID(body, idKey)
		if err != nil {
			return nil, err
		}
		return manager.GetByID(ctx, table, id)
	})
}

func legacyDeleteByIDHandler(manager *compat.AdminService, table, idKey string) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if manager == nil {
			legacyUnavailable(c)
			return
		}
		id, err := legacyID(c.Query(idKey))
		if err != nil {
			legacyResult(c, nil, err)
			return
		}
		legacyResult(c, true, manager.DeleteByID(ctx, table, id))
	}
}

func legacyStaticBodyHandler(result func(map[string]any) any) app.HandlerFunc {
	return func(_ context.Context, c *app.RequestContext) {
		body, ok := legacyBody(c)
		if ok {
			c.JSON(consts.StatusOK, successEnvelope(newRequestID(), result(body)))
		}
	}
}
func legacySuccessHandler(value any) app.HandlerFunc {
	return func(_ context.Context, c *app.RequestContext) {
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
	}
}
func legacySessionGetHandler(_ context.Context, c *app.RequestContext) {
	c.JSON(consts.StatusOK, successEnvelope(newRequestID(), map[string]any{"sessionId": c.Query("sessionId"), "messages": []any{}, "messageCount": 0}))
}
func legacyEmptyPageHandler(pageKey, sizeKey string) app.HandlerFunc {
	return func(_ context.Context, c *app.RequestContext) {
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), emptyLegacyPage(legacyPage(c, pageKey), legacyPage(c, sizeKey))))
	}
}

func observabilityTracesHandler(manager *compat.ObservabilityService) app.HandlerFunc {
	return func(_ context.Context, c *app.RequestContext) {
		if manager == nil {
			legacyUnavailable(c)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), manager.Traces(legacyPage(c, "pageNumber"), legacyPage(c, "pageSize"), c.Query("service"), c.Query("spanName"))))
	}
}
func observabilityAddTraceHandler(manager *compat.ObservabilityService) app.HandlerFunc {
	return func(_ context.Context, c *app.RequestContext) {
		body, ok := legacyBody(c)
		if !ok {
			return
		}
		if manager == nil {
			legacyUnavailable(c)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), manager.AddTrace(body)))
	}
}
func observabilityTraceDetailHandler(manager *compat.ObservabilityService) app.HandlerFunc {
	return func(_ context.Context, c *app.RequestContext) {
		if manager == nil {
			legacyUnavailable(c)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), manager.TraceDetail(c.Param("traceId"))))
	}
}
func observabilityServicesHandler(manager *compat.ObservabilityService) app.HandlerFunc {
	return func(_ context.Context, c *app.RequestContext) {
		if manager == nil {
			legacyUnavailable(c)
			return
		}
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), manager.Services()))
	}
}
func observabilityOverviewHandler(manager *compat.ObservabilityService) app.HandlerFunc {
	return func(_ context.Context, c *app.RequestContext) {
		if manager == nil {
			legacyUnavailable(c)
			return
		}
		detail, _ := parseBool(c.Query("detail"))
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), manager.Overview(detail)))
	}
}

func callPrompt(manager *compat.AdminService, ctx context.Context, key string) (any, error) {
	if manager == nil {
		return nil, compat.ErrNotConfigured
	}
	return manager.GetPrompt(ctx, key)
}
func legacyBody(c *app.RequestContext) (map[string]any, bool) {
	body := map[string]any{}
	if json.Unmarshal(c.Request.Body(), &body) != nil {
		writeInvalidJSON(c)
		return nil, false
	}
	return body, true
}
func legacyPage(c *app.RequestContext, key string) int64 {
	return int64(parsePositiveInt(c.Query(key), map[string]int{"pageNo": 1, "pageNumber": 1, "page": 1, "pageSize": 10, "size": 10}[key]))
}
func legacyID(value string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || id < 1 {
		return 0, compat.ErrInvalidID
	}
	return id, nil
}
func legacyMapID(body map[string]any, key string) (int64, error) {
	return legacyID(textValue(body, key))
}
func textValue(body map[string]any, key string) string {
	if body[key] == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(body[key]))
}
func emptyLegacyPage(page, size int64) *compat.Page {
	page, size = normalizedLegacyPage(page, size)
	return &compat.Page{PageNumber: page, PageSize: size, PageItems: []map[string]any{}}
}
func normalizedLegacyPage(page, size int64) (int64, int64) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}
	return page, size
}
func legacyUnavailable(c *app.RequestContext) {
	c.JSON(consts.StatusServiceUnavailable, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "legacy admin service is unavailable"))
}
func legacyResult(c *app.RequestContext, value any, err error) {
	if err == nil {
		c.JSON(consts.StatusOK, successEnvelope(newRequestID(), value))
		return
	}
	if errors.Is(err, compat.ErrInvalidID) || errors.Is(err, compat.ErrInvalidTable) {
		c.JSON(consts.StatusBadRequest, errorEnvelope(newRequestID(), "INVALID_PARAMS", err.Error()))
		return
	}
	if errors.Is(err, compat.ErrNotConfigured) {
		legacyUnavailable(c)
		return
	}
	c.JSON(consts.StatusInternalServerError, errorEnvelope(newRequestID(), "SYSTEM_ERROR", "legacy admin request failed"))
}
