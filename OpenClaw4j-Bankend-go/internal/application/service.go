package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/seaskyland/openclaw4j-backend-go/internal/dao"
)

const (
	statusDeleted          int16 = 0
	statusDraft            int16 = 1
	statusPublished        int16 = 2
	statusPublishedEditing int16 = 3
)

var (
	ErrNameRequired          = errors.New("application name is required")
	ErrConfigRequired        = errors.New("application config is required")
	ErrNameExists            = errors.New("application name already exists")
	ErrNotFound              = errors.New("application not found")
	ErrVersionNotFound       = errors.New("application version not found")
	ErrModelProviderRequired = errors.New("model_provider is required")
	ErrModelRequired         = errors.New("model is required")
)

type Service struct {
	dao   dao.ApplicationDAO
	clock func() time.Time
}
type Input struct {
	Name        string          `json:"name"`
	Description *string         `json:"description"`
	Icon        *string         `json:"icon"`
	Type        string          `json:"type"`
	Config      json.RawMessage `json:"config"`
	Source      *string         `json:"source"`
}
type Application struct {
	AppID       string         `json:"app_id"`
	Name        string         `json:"name"`
	Description *string        `json:"description,omitempty"`
	Icon        *string        `json:"icon,omitempty"`
	Type        string         `json:"type"`
	Status      string         `json:"status"`
	Config      map[string]any `json:"config,omitempty"`
	PubConfig   map[string]any `json:"pub_config,omitempty"`
	Source      string         `json:"source"`
	GmtCreate   time.Time      `json:"gmt_create"`
	GmtModified time.Time      `json:"gmt_modified"`
}
type Version struct {
	WorkspaceID string    `json:"workspace_id"`
	AppID       string    `json:"app_id"`
	Status      string    `json:"status"`
	Config      string    `json:"config"`
	Version     string    `json:"version"`
	Description *string   `json:"description,omitempty"`
	GmtCreate   time.Time `json:"gmt_create"`
	GmtModified time.Time `json:"gmt_modified"`
	Creator     string    `json:"creator"`
	Modifier    string    `json:"modifier"`
}
type Page[T any] struct {
	Current int64 `json:"current"`
	Size    int64 `json:"size"`
	Total   int64 `json:"total"`
	Records []T   `json:"records"`
}

func NewService(data dao.ApplicationDAO, clock func() time.Time) *Service {
	if clock == nil {
		clock = time.Now
	}
	return &Service{dao: data, clock: clock}
}
func (s *Service) Create(ctx context.Context, workspaceID, accountID string, input Input) (string, error) {
	if strings.TrimSpace(input.Name) == "" {
		return "", ErrNameRequired
	}
	if len(input.Config) == 0 || string(input.Config) == "null" {
		return "", ErrConfigRequired
	}
	if _, e := s.dao.FindByName(ctx, input.Name, workspaceID); e == nil {
		return "", ErrNameExists
	} else if !errors.Is(e, dao.ErrNotFound) {
		return "", e
	}
	appID, e := newID()
	if e != nil {
		return "", e
	}
	now := s.clock()
	cfg := string(input.Config)
	source := "console"
	if input.Source != nil && strings.TrimSpace(*input.Source) != "" {
		source = *input.Source
	}
	typ := strings.TrimSpace(input.Type)
	if typ == "" {
		typ = "basic"
	}
	app := dao.Application{WorkspaceID: workspaceID, AppID: appID, Name: strings.TrimSpace(input.Name), Description: input.Description, Icon: input.Icon, Source: source, Type: typ, Status: statusDraft, GmtCreate: now, GmtModified: now, Creator: accountID, Modifier: accountID}
	if e = s.dao.Create(ctx, app); e != nil {
		return "", fmt.Errorf("create application: %w", e)
	}
	if e = s.dao.CreateVersion(ctx, dao.ApplicationVersion{AppID: appID, WorkspaceID: workspaceID, Config: &cfg, Status: statusDraft, Version: "1", Description: input.Description, GmtCreate: now, GmtModified: now, Creator: accountID, Modifier: accountID}); e != nil {
		return "", fmt.Errorf("create application version: %w", e)
	}
	return appID, nil
}
func (s *Service) Get(ctx context.Context, w, id string) (*Application, error) {
	a, e := s.find(ctx, w, id)
	if e != nil {
		return nil, e
	}
	latest, e := s.dao.FindLatestVersion(ctx, id, w)
	if e != nil {
		return nil, ErrVersionNotFound
	}
	out := toApplication(a, latest)
	if published, e := s.dao.FindLastPublishedVersion(ctx, id, w); e == nil {
		out.PubConfig = parseConfig(published.Config)
	}
	return &out, nil
}
func (s *Service) Update(ctx context.Context, w, actor, id string, input Input) error {
	a, e := s.find(ctx, w, id)
	if e != nil {
		return e
	}
	if strings.TrimSpace(input.Name) == "" {
		return ErrNameRequired
	}
	if other, e := s.dao.FindByName(ctx, input.Name, w); e == nil && other.ID != a.ID {
		return ErrNameExists
	} else if e != nil && !errors.Is(e, dao.ErrNotFound) {
		return e
	}
	latest, e := s.dao.FindLatestVersion(ctx, id, w)
	if e != nil {
		return ErrVersionNotFound
	}
	now := s.clock()
	if a.Status == statusPublished {
		next := nextVersion(latest.Version)
		cfg := latest.Config
		if len(input.Config) > 0 {
			v := string(input.Config)
			cfg = &v
		}
		if e = s.dao.CreateVersion(ctx, dao.ApplicationVersion{AppID: id, WorkspaceID: w, Config: cfg, Status: statusDraft, Version: next, Description: input.Description, GmtCreate: now, GmtModified: now, Creator: actor, Modifier: actor}); e != nil {
			return e
		}
		a.Status = statusPublishedEditing
	} else {
		if len(input.Config) > 0 {
			v := string(input.Config)
			latest.Config = &v
		}
		latest.Description = input.Description
		latest.GmtModified = now
		latest.Modifier = actor
		if e = s.dao.UpdateVersion(ctx, *latest); e != nil {
			return e
		}
	}
	a.Name = strings.TrimSpace(input.Name)
	a.Description = input.Description
	a.Icon = input.Icon
	if strings.TrimSpace(input.Type) != "" {
		a.Type = input.Type
	}
	a.GmtModified = now
	a.Modifier = actor
	return s.dao.Update(ctx, *a)
}
func (s *Service) Delete(ctx context.Context, w, actor, id string) error {
	if _, e := s.find(ctx, w, id); e != nil {
		if errors.Is(e, ErrNotFound) {
			return nil
		}
		return e
	}
	return s.dao.Delete(ctx, id, w, actor, s.clock())
}
func (s *Service) Publish(ctx context.Context, w, actor, id string) error {
	a, e := s.find(ctx, w, id)
	if e != nil {
		return e
	}
	v, e := s.dao.FindLatestVersion(ctx, id, w)
	if e != nil {
		return ErrVersionNotFound
	}
	if a.Type == "basic" {
		var cfg map[string]any
		if e := json.Unmarshal([]byte(value(v.Config)), &cfg); e != nil {
			return ErrConfigRequired
		}
		if strings.TrimSpace(stringValue(cfg["model_provider"])) == "" {
			return ErrModelProviderRequired
		}
		if strings.TrimSpace(modelID(cfg["model"])) == "" {
			return ErrModelRequired
		}
	}
	now := s.clock()
	v.Status = statusPublished
	v.GmtModified = now
	v.Modifier = actor
	if e = s.dao.UpdateVersion(ctx, *v); e != nil {
		return e
	}
	a.Status = statusPublished
	a.GmtModified = now
	a.Modifier = actor
	return s.dao.Update(ctx, *a)
}
func (s *Service) List(ctx context.Context, w, name, typ, status string, current, size int64) (*Page[Application], error) {
	current, size = page(current, size)
	code := statusCode(status)
	rows, e := s.dao.List(ctx, w, strings.TrimSpace(name), strings.TrimSpace(typ), code, int32(size), int32((current-1)*size))
	if e != nil {
		return nil, e
	}
	total, e := s.dao.Count(ctx, w, strings.TrimSpace(name), strings.TrimSpace(typ), code)
	if e != nil {
		return nil, e
	}
	out := make([]Application, 0, len(rows))
	for _, a := range rows {
		v, _ := s.dao.FindLatestVersion(ctx, a.AppID, w)
		out = append(out, toApplication(&a, v))
	}
	return &Page[Application]{current, size, total, out}, nil
}
func (s *Service) ListVersions(ctx context.Context, w, id, status string, current, size int64) (*Page[Version], error) {
	current, size = page(current, size)
	code := statusCode(status)
	rows, e := s.dao.ListVersions(ctx, id, w, code, int32(size), int32((current-1)*size))
	if e != nil {
		return nil, e
	}
	total, e := s.dao.CountVersions(ctx, id, w, code)
	if e != nil {
		return nil, e
	}
	out := make([]Version, 0, len(rows))
	for _, v := range rows {
		out = append(out, toVersion(&v))
	}
	return &Page[Version]{current, size, total, out}, nil
}
func (s *Service) GetVersion(ctx context.Context, w, id, version string) (*Version, error) {
	var v *dao.ApplicationVersion
	var e error
	if version == "latest" {
		v, e = s.dao.FindLatestVersion(ctx, id, w)
	} else if version == "lastPublished" {
		v, e = s.dao.FindLastPublishedVersion(ctx, id, w)
	} else {
		v, e = s.dao.FindVersion(ctx, id, w, version)
	}
	if errors.Is(e, dao.ErrNotFound) {
		return nil, ErrVersionNotFound
	}
	if e != nil {
		return nil, e
	}
	out := toVersion(v)
	return &out, nil
}
func (s *Service) Copy(ctx context.Context, w, actor, id string) (string, error) {
	a, e := s.find(ctx, w, id)
	if e != nil {
		return "", e
	}
	v, e := s.dao.FindLatestVersion(ctx, id, w)
	if e != nil {
		return "", ErrVersionNotFound
	}
	newID, e := newID()
	if e != nil {
		return "", e
	}
	now := s.clock()
	copy := *a
	copy.AppID = newID
	copy.Name = a.Name + "_copy_" + now.Format("20060102150405")
	copy.Status = statusDraft
	copy.GmtCreate = now
	copy.GmtModified = now
	copy.Creator = actor
	copy.Modifier = actor
	if e = s.dao.Create(ctx, copy); e != nil {
		return "", e
	}
	vcopy := *v
	vcopy.AppID = newID
	vcopy.Status = statusDraft
	vcopy.Version = "1"
	vcopy.GmtCreate = now
	vcopy.GmtModified = now
	vcopy.Creator = actor
	vcopy.Modifier = actor
	if e = s.dao.CreateVersion(ctx, vcopy); e != nil {
		return "", e
	}
	return newID, nil
}
func (s *Service) find(ctx context.Context, w, id string) (*dao.Application, error) {
	a, e := s.dao.FindByID(ctx, id, w)
	if errors.Is(e, dao.ErrNotFound) {
		return nil, ErrNotFound
	}
	return a, e
}
func toApplication(a *dao.Application, v *dao.ApplicationVersion) Application {
	return Application{AppID: a.AppID, Name: a.Name, Description: a.Description, Icon: a.Icon, Type: a.Type, Status: statusName(a.Status), Config: parseConfig(v.Config), Source: a.Source, GmtCreate: a.GmtCreate, GmtModified: a.GmtModified}
}
func toVersion(v *dao.ApplicationVersion) Version {
	return Version{WorkspaceID: v.WorkspaceID, AppID: v.AppID, Status: statusName(v.Status), Config: value(v.Config), Version: v.Version, Description: v.Description, GmtCreate: v.GmtCreate, GmtModified: v.GmtModified, Creator: v.Creator, Modifier: v.Modifier}
}
func parseConfig(raw *string) map[string]any {
	out := map[string]any{}
	if raw != nil {
		_ = json.Unmarshal([]byte(*raw), &out)
	}
	return out
}
func value(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
func stringValue(v any) string { s, _ := v.(string); return s }
func modelID(v any) string {
	if value := stringValue(v); value != "" {
		return value
	}
	model, ok := v.(map[string]any)
	if !ok {
		return ""
	}
	for _, key := range []string{"model_id", "id", "name"} {
		if value := stringValue(model[key]); value != "" {
			return value
		}
	}
	return ""
}
func page(c, s int64) (int64, int64) {
	if c < 1 {
		c = 1
	}
	if s < 1 {
		s = 10
	}
	if s > 100 {
		s = 100
	}
	return c, s
}
func statusCode(v string) int16 {
	switch v {
	case "draft":
		return statusDraft
	case "published":
		return statusPublished
	case "published_editing":
		return statusPublishedEditing
	default:
		return -1
	}
}
func statusName(v int16) string {
	switch v {
	case statusDraft:
		return "draft"
	case statusPublished:
		return "published"
	case statusPublishedEditing:
		return "published_editing"
	default:
		return "deleted"
	}
}
func nextVersion(v string) string {
	n, e := strconv.Atoi(v)
	if e != nil {
		return "1"
	}
	return strconv.Itoa(n + 1)
}
func newID() (string, error) {
	var b [12]byte
	if _, e := rand.Read(b[:]); e != nil {
		return "", e
	}
	return hex.EncodeToString(b[:]), nil
}
