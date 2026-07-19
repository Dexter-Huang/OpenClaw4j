// Package compat preserves the Java legacy admin API contracts during the Go migration.
package compat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/seaskyland/openclaw4j-backend-go/internal/db"
)

var (
	ErrNotConfigured = errors.New("legacy admin database is unavailable")
	ErrInvalidTable  = errors.New("unsupported legacy admin table")
	ErrInvalidID     = errors.New("legacy admin id is required")
)

type Page struct {
	TotalCount int64            `json:"totalCount"`
	TotalPage  int64            `json:"totalPage"`
	PageNumber int64            `json:"pageNumber"`
	PageSize   int64            `json:"pageSize"`
	PageItems  []map[string]any `json:"pageItems"`
}

type AdminService struct{ db db.DBTX }

func NewAdminService(database db.DBTX) *AdminService {
	if database == nil {
		return nil
	}
	return &AdminService{db: database}
}

// List matches AdminCompatService.list: only known legacy tables/order columns are interpolated;
// all values remain positional pgx parameters.
func (s *AdminService) List(ctx context.Context, table, order string, pageNumber, pageSize int64) (*Page, error) {
	if s == nil || s.db == nil {
		return nil, ErrNotConfigured
	}
	where, ok := legacyTableFilter(table)
	if !ok || !legacyOrderAllowed(table, order) {
		return nil, ErrInvalidTable
	}
	pageNumber, pageSize = normalizePage(pageNumber, pageSize)
	var total int64
	if err := s.db.QueryRow(ctx, "SELECT COUNT(1) FROM "+table+where).Scan(&total); err != nil {
		return nil, fmt.Errorf("count %s: %w", table, err)
	}
	rows, err := s.db.Query(ctx, "SELECT * FROM "+table+where+" ORDER BY "+order+" DESC LIMIT $1 OFFSET $2", pageSize, (pageNumber-1)*pageSize)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", table, err)
	}
	items, err := rowsToMaps(rows)
	if err != nil {
		return nil, err
	}
	pages := int64(0)
	if total > 0 {
		pages = (total + pageSize - 1) / pageSize
	}
	return &Page{TotalCount: total, TotalPage: pages, PageNumber: pageNumber, PageSize: pageSize, PageItems: items}, nil
}

func (s *AdminService) GetByID(ctx context.Context, table string, id int64) (map[string]any, error) {
	if s == nil || s.db == nil {
		return nil, ErrNotConfigured
	}
	where, ok := legacyTableFilter(table)
	if !ok {
		return nil, ErrInvalidTable
	}
	rows, err := s.db.Query(ctx, "SELECT * FROM "+table+where+" AND id = $1", id)
	if err != nil {
		return nil, fmt.Errorf("get %s: %w", table, err)
	}
	items, err := rowsToMaps(rows)
	if err != nil || len(items) == 0 {
		return nil, err
	}
	return items[0], nil
}

func (s *AdminService) GetPrompt(ctx context.Context, promptKey string) (map[string]any, error) {
	return s.one(ctx, "SELECT * FROM prompt WHERE prompt_key = $1", promptKey)
}

func (s *AdminService) GetPromptVersion(ctx context.Context, promptKey, version string) (map[string]any, error) {
	return s.one(ctx, "SELECT * FROM prompt_version WHERE prompt_key = $1 AND version = $2", promptKey, version)
}

func (s *AdminService) GetPromptTemplate(ctx context.Context, key string) (map[string]any, error) {
	return s.one(ctx, "SELECT * FROM prompt_build_template WHERE prompt_template_key = $1", key)
}

func (s *AdminService) CreateOrUpdatePrompt(ctx context.Context, body map[string]any) (map[string]any, error) {
	key := text(body, "promptKey")
	if key == "" {
		return nil, ErrInvalidID
	}
	result, err := s.db.Exec(ctx, "UPDATE prompt SET prompt_desc=$1,prompt_description=$1,tags=$2,update_time=CURRENT_TIMESTAMP WHERE prompt_key=$3", text(body, "promptDescription"), text(body, "tags"), key)
	if err != nil {
		return nil, err
	}
	if result.RowsAffected() == 0 {
		_, err = s.db.Exec(ctx, "INSERT INTO prompt(prompt_key,prompt_desc,prompt_description,latest_version,tags,create_time,update_time) VALUES($1,$2,$2,'1.0.0',$3,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)", key, text(body, "promptDescription"), text(body, "tags"))
		if err != nil {
			return nil, err
		}
	}
	return s.GetPrompt(ctx, key)
}

func (s *AdminService) CreatePromptVersion(ctx context.Context, body map[string]any) (map[string]any, error) {
	key := text(body, "promptKey")
	if key == "" {
		return nil, ErrInvalidID
	}
	version := text(body, "version")
	if version == "" {
		var count int64
		if err := s.db.QueryRow(ctx, "SELECT COUNT(1) FROM prompt_version WHERE prompt_key=$1", key).Scan(&count); err != nil {
			return nil, err
		}
		version = "1.0." + strconv.FormatInt(count+1, 10)
	}
	_, err := s.db.Exec(ctx, `INSERT INTO prompt_version(prompt_key,version,version_description,template,variables,model_config,previous_version,status,create_time,update_time)
        VALUES($1,$2,$3,$4,$5,$6,$7,$8,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)
        ON CONFLICT(prompt_key,version) DO UPDATE SET version_description=EXCLUDED.version_description,template=EXCLUDED.template,variables=EXCLUDED.variables,model_config=EXCLUDED.model_config,previous_version=EXCLUDED.previous_version,status=EXCLUDED.status,update_time=CURRENT_TIMESTAMP`,
		key, version, text(body, "versionDescription"), text(body, "template"), text(body, "variables"), text(body, "modelConfig"), text(body, "previousVersion"), text(body, "status"))
	if err != nil {
		return nil, err
	}
	_, err = s.db.Exec(ctx, "UPDATE prompt SET latest_version=$1,latest_version_status=$2,update_time=CURRENT_TIMESTAMP WHERE prompt_key=$3", version, text(body, "status"), key)
	if err != nil {
		return nil, err
	}
	return s.GetPromptVersion(ctx, key, version)
}

func (s *AdminService) CreateNamed(ctx context.Context, table string, body map[string]any) (map[string]any, error) {
	if table != "dataset" && table != "evaluator" {
		return nil, ErrInvalidTable
	}
	var id int64
	err := s.db.QueryRow(ctx, "INSERT INTO "+table+"(name,description,create_time,update_time) VALUES($1,$2,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP) RETURNING id", text(body, "name"), text(body, "description")).Scan(&id)
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, table, id)
}

func (s *AdminService) UpdateNamed(ctx context.Context, table string, id int64, body map[string]any) (map[string]any, error) {
	if table != "dataset" && table != "evaluator" {
		return nil, ErrInvalidTable
	}
	if _, err := s.db.Exec(ctx, "UPDATE "+table+" SET name=$1,description=$2,update_time=CURRENT_TIMESTAMP WHERE id=$3", text(body, "name"), text(body, "description"), id); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, table, id)
}

func (s *AdminService) CreateDatasetVersion(ctx context.Context, body map[string]any) (map[string]any, error) {
	id, err := integer(body, "datasetId")
	if err != nil {
		return nil, err
	}
	version, err := s.nextVersion(ctx, "dataset_version", "dataset_id", id)
	if err != nil {
		return nil, err
	}
	var resultID int64
	err = s.db.QueryRow(ctx, `INSERT INTO dataset_version(dataset_id,version,description,data_count,status,dataset_items,columns_config,create_time,update_time)
        VALUES($1,$2,$3,0,$4,$5,$6,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP) RETURNING id`, id, version, text(body, "description"), text(body, "status"), jsonText(body["datasetItems"]), jsonText(body["columnsConfig"])).Scan(&resultID)
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, "dataset_version", resultID)
}

func (s *AdminService) CreateEvaluatorVersion(ctx context.Context, body map[string]any) (map[string]any, error) {
	evaluatorID, err := integer(body, "evaluatorId")
	if err != nil {
		return nil, err
	}
	version := text(body, "version")
	if version == "" {
		version, err = s.nextVersion(ctx, "evaluator_version", "evaluator_id", evaluatorID)
		if err != nil {
			return nil, err
		}
	}
	var id int64
	err = s.db.QueryRow(ctx, `INSERT INTO evaluator_version(evaluator_id,description,version,model_config,prompt,variables,status,create_time,update_time)
        VALUES($1,$2,$3,$4,$5,$6,'draft',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP) RETURNING id`, evaluatorID, text(body, "description"), version, text(body, "modelConfig"), text(body, "prompt"), text(body, "variables")).Scan(&id)
	if err != nil {
		return nil, err
	}
	_, _ = s.db.Exec(ctx, "UPDATE evaluator SET latest_version=$1,prompt=$2,model_config=$3,variables=$4,update_time=CURRENT_TIMESTAMP WHERE id=$5", version, text(body, "prompt"), text(body, "modelConfig"), text(body, "variables"), evaluatorID)
	return s.GetByID(ctx, "evaluator_version", id)
}

func (s *AdminService) CreateExperiment(ctx context.Context, body map[string]any) (map[string]any, error) {
	datasetID, err := optionalInteger(body, "datasetId")
	if err != nil {
		return nil, err
	}
	datasetVersionID, err := optionalInteger(body, "datasetVersionId")
	if err != nil {
		return nil, err
	}
	var id int64
	err = s.db.QueryRow(ctx, `INSERT INTO experiment(name,description,dataset_id,dataset_version_id,dataset_version,evaluation_object_config,evaluator_config,status,progress,create_time,update_time)
        VALUES($1,$2,$3,$4,$5,$6,$7,'created',0,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP) RETURNING id`, text(body, "name"), text(body, "description"), datasetID, datasetVersionID, text(body, "datasetVersion"), text(body, "evaluationObjectConfig"), text(body, "evaluatorConfig")).Scan(&id)
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, "experiment", id)
}

func (s *AdminService) DeleteByID(ctx context.Context, table string, id int64) error {
	where, ok := legacyTableFilter(table)
	if !ok {
		return ErrInvalidTable
	}
	if strings.Contains(where, "deleted") {
		_, err := s.db.Exec(ctx, "UPDATE "+table+" SET deleted=1,update_time=CURRENT_TIMESTAMP WHERE id=$1", id)
		return err
	}
	_, err := s.db.Exec(ctx, "DELETE FROM "+table+" WHERE id=$1", id)
	return err
}

func (s *AdminService) DeletePrompt(ctx context.Context, key string) error {
	_, err := s.db.Exec(ctx, "DELETE FROM prompt WHERE prompt_key=$1", key)
	return err
}

func (s *AdminService) one(ctx context.Context, query string, args ...any) (map[string]any, error) {
	if s == nil || s.db == nil {
		return nil, ErrNotConfigured
	}
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	items, err := rowsToMaps(rows)
	if err != nil || len(items) == 0 {
		return nil, err
	}
	return items[0], nil
}

func (s *AdminService) nextVersion(ctx context.Context, table, column string, id int64) (string, error) {
	var count int64
	if err := s.db.QueryRow(ctx, "SELECT COUNT(1) FROM "+table+" WHERE "+column+"=$1", id).Scan(&count); err != nil {
		return "", err
	}
	return "1.0." + strconv.FormatInt(count+1, 10), nil
}

func legacyTableFilter(table string) (string, bool) {
	switch table {
	case "prompt", "prompt_version", "prompt_build_template", "dataset_version", "evaluator_version", "evaluator_template", "experiment", "experiment_result":
		return " WHERE TRUE", true
	case "dataset", "dataset_item", "evaluator", "model_config":
		return " WHERE deleted = 0", true
	case "model":
		return " WHERE enable = 1", true
	default:
		return "", false
	}
}

func legacyOrderAllowed(table, order string) bool {
	return map[string]map[string]bool{
		"prompt": {"update_time": true}, "prompt_version": {"update_time": true}, "prompt_build_template": {"id": true},
		"dataset": {"update_time": true}, "dataset_version": {"update_time": true}, "dataset_item": {"update_time": true},
		"evaluator": {"update_time": true}, "evaluator_version": {"update_time": true}, "evaluator_template": {"id": true},
		"experiment": {"update_time": true}, "experiment_result": {"update_time": true}, "model_config": {"update_time": true}, "model": {"gmt_modified": true},
	}[table][order]
}

func rowsToMaps(rows pgx.Rows) ([]map[string]any, error) {
	defer rows.Close()
	fields := rows.FieldDescriptions()
	items := make([]map[string]any, 0)
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, err
		}
		item := make(map[string]any, len(values)+2)
		for index, field := range fields {
			item[toCamel(string(field.Name))] = values[index]
		}
		alias(item, "promptDesc", "promptDescription")
		alias(item, "templateDesc", "templateDescription")
		items = append(items, item)
	}
	return items, rows.Err()
}

func normalizePage(page, size int64) (int64, int64) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}
	if size > 100 {
		size = 100
	}
	return page, size
}

func toCamel(value string) string {
	parts := strings.Split(strings.ToLower(value), "_")
	for index := 1; index < len(parts); index++ {
		if len(parts[index]) > 0 {
			parts[index] = strings.ToUpper(parts[index][:1]) + parts[index][1:]
		}
	}
	return strings.Join(parts, "")
}

func alias(data map[string]any, from, to string) {
	if value, ok := data[from]; ok && data[to] == nil {
		data[to] = value
	}
}

func text(data map[string]any, key string) string {
	if value := data[key]; value != nil {
		return fmt.Sprint(value)
	}
	return ""
}

func integer(data map[string]any, key string) (int64, error) {
	if data[key] == nil {
		return 0, ErrInvalidID
	}
	return strconv.ParseInt(fmt.Sprint(data[key]), 10, 64)
}

func optionalInteger(data map[string]any, key string) (any, error) {
	if data[key] == nil || strings.TrimSpace(fmt.Sprint(data[key])) == "" {
		return nil, nil
	}
	return strconv.ParseInt(fmt.Sprint(data[key]), 10, 64)
}

func jsonText(value any) string {
	if value == nil {
		return "[]"
	}
	if text, ok := value.(string); ok {
		return text
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return "[]"
	}
	return string(encoded)
}
