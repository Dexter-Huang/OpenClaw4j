package tool

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/seaskyland/openclaw4j-backend-go/internal/dao"
)

const (
	statusPublished int16 = 2
	testNotTested   int16 = 1
)

var (
	ErrNameRequired   = errors.New("tool name is required")
	ErrConfigRequired = errors.New("tool config is required")
	ErrSchemaRequired = errors.New("tool api_schema is required")
	ErrNotFound       = errors.New("tool not found")
)

type Service struct {
	dao   dao.ToolDAO
	clock func() time.Time
}
type Input struct {
	PluginID    *string `json:"pluginId"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Config      *string `json:"config"`
	APISchema   *string `json:"apiSchema"`
	Status      *string `json:"status"`
	TestStatus  *string `json:"testStatus"`
	Enabled     *bool   `json:"enabled"`
}
type Tool struct {
	ID          int64     `json:"id"`
	ToolID      string    `json:"toolId"`
	PluginID    string    `json:"pluginId"`
	WorkspaceID string    `json:"workspaceId"`
	Status      string    `json:"status"`
	TestStatus  string    `json:"testStatus"`
	Enabled     bool      `json:"enabled"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	Config      string    `json:"config"`
	APISchema   string    `json:"apiSchema"`
	GmtCreate   time.Time `json:"gmtCreate"`
	GmtModified time.Time `json:"gmtModified"`
	Creator     string    `json:"creator"`
	Modifier    string    `json:"modifier"`
}
type Page struct {
	Current int64  `json:"current"`
	Size    int64  `json:"size"`
	Total   int64  `json:"total"`
	Records []Tool `json:"records"`
}

func NewService(data dao.ToolDAO, clock func() time.Time) *Service {
	if clock == nil {
		clock = time.Now
	}
	return &Service{dao: data, clock: clock}
}
func (s *Service) Create(ctx context.Context, w, a string, in Input) (*Tool, error) {
	name, config, schema := required(in.Name), required(in.Config), required(in.APISchema)
	if name == "" {
		return nil, ErrNameRequired
	}
	if config == "" {
		return nil, ErrConfigRequired
	}
	if schema == "" {
		return nil, ErrSchemaRequired
	}
	id, e := newID()
	if e != nil {
		return nil, e
	}
	now := s.clock()
	v := dao.Tool{ToolID: id, PluginID: required(in.PluginID), WorkspaceID: w, Status: statusPublished, Enabled: true, TestStatus: testNotTested, Name: name, Description: clean(in.Description), Config: config, APISchema: schema, GmtCreate: now, GmtModified: now, Creator: a, Modifier: a}
	v.ID, e = s.dao.Create(ctx, v)
	if e != nil {
		return nil, e
	}
	return mapTool(v), nil
}
func (s *Service) Get(ctx context.Context, w string, id int64) (*Tool, error) {
	v, e := s.dao.Find(ctx, id, w)
	if errors.Is(e, dao.ErrNotFound) {
		return nil, ErrNotFound
	}
	if e != nil {
		return nil, e
	}
	return mapTool(*v), nil
}
func (s *Service) List(ctx context.Context, w, name, plugin string) ([]Tool, error) {
	values, e := s.dao.List(ctx, w, strings.TrimSpace(name), strings.TrimSpace(plugin))
	if e != nil {
		return nil, e
	}
	out := make([]Tool, 0, len(values))
	for _, v := range values {
		out = append(out, *mapTool(v))
	}
	return out, nil
}
func (s *Service) Page(ctx context.Context, w, name, plugin string, current, size int64) (*Page, error) {
	if current < 1 {
		current = 1
	}
	if size < 1 {
		size = 10
	}
	if size > 100 {
		size = 100
	}
	values, e := s.List(ctx, w, name, plugin)
	if e != nil {
		return nil, e
	}
	total := int64(len(values))
	start := (current - 1) * size
	if start >= total {
		return &Page{Current: current, Size: size, Total: total, Records: []Tool{}}, nil
	}
	end := start + size
	if end > total {
		end = total
	}
	return &Page{Current: current, Size: size, Total: total, Records: values[start:end]}, nil
}
func (s *Service) Update(ctx context.Context, w, a string, id int64, in Input) (*Tool, error) {
	v, e := s.dao.Find(ctx, id, w)
	if errors.Is(e, dao.ErrNotFound) {
		return nil, ErrNotFound
	}
	if e != nil {
		return nil, e
	}
	if in.Name != nil {
		if v.Name = required(in.Name); v.Name == "" {
			return nil, ErrNameRequired
		}
	}
	if in.Config != nil {
		if v.Config = required(in.Config); v.Config == "" {
			return nil, ErrConfigRequired
		}
	}
	if in.APISchema != nil {
		if v.APISchema = required(in.APISchema); v.APISchema == "" {
			return nil, ErrSchemaRequired
		}
	}
	if in.PluginID != nil {
		v.PluginID = required(in.PluginID)
	}
	if in.Description != nil {
		v.Description = clean(in.Description)
	}
	if in.Enabled != nil {
		v.Enabled = *in.Enabled
	}
	if in.Status != nil {
		v.Status = statusCode(*in.Status, v.Status)
	}
	if in.TestStatus != nil {
		v.TestStatus = testStatusCode(*in.TestStatus, v.TestStatus)
	}
	v.GmtModified = s.clock()
	v.Modifier = a
	if e = s.dao.Update(ctx, *v); e != nil {
		return nil, e
	}
	return mapTool(*v), nil
}
func (s *Service) SetEnabled(ctx context.Context, w, a string, id int64, enabled bool) error {
	_, e := s.Update(ctx, w, a, id, Input{Enabled: &enabled})
	return e
}
func (s *Service) Delete(ctx context.Context, w string, id int64) error {
	if _, e := s.Get(ctx, w, id); e != nil {
		if errors.Is(e, ErrNotFound) {
			return nil
		}
		return e
	}
	return s.dao.Delete(ctx, id, w)
}
func required(v *string) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(*v)
}
func clean(v *string) *string {
	if s := required(v); s != "" {
		return &s
	}
	return nil
}
func mapTool(v dao.Tool) *Tool {
	return &Tool{ID: v.ID, ToolID: v.ToolID, PluginID: v.PluginID, WorkspaceID: v.WorkspaceID, Status: statusName(v.Status), TestStatus: testStatusName(v.TestStatus), Enabled: v.Enabled, Name: v.Name, Description: v.Description, Config: v.Config, APISchema: v.APISchema, GmtCreate: v.GmtCreate, GmtModified: v.GmtModified, Creator: v.Creator, Modifier: v.Modifier}
}
func statusName(v int16) string {
	switch v {
	case 0:
		return "deleted"
	case 1:
		return "draft"
	case 2:
		return "published"
	case 3:
		return "published_editing"
	default:
		return "draft"
	}
}
func statusCode(v string, f int16) int16 {
	switch strings.TrimSpace(v) {
	case "deleted":
		return 0
	case "draft":
		return 1
	case "published":
		return 2
	case "published_editing":
		return 3
	default:
		return f
	}
}
func testStatusName(v int16) string {
	switch v {
	case 2:
		return "passed"
	case 3:
		return "failed"
	default:
		return "not_test"
	}
}
func testStatusCode(v string, f int16) int16 {
	switch strings.TrimSpace(v) {
	case "not_test", "not_tested":
		return 1
	case "passed":
		return 2
	case "failed":
		return 3
	default:
		return f
	}
}
func newID() (string, error) {
	var b [12]byte
	if _, e := rand.Read(b[:]); e != nil {
		return "", e
	}
	return "tool_" + hex.EncodeToString(b[:]), nil
}
