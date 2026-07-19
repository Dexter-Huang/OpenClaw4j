package agentschema

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/seaskyland/openclaw4j-backend-go/internal/dao"
)

var (
	ErrNameRequired = errors.New("agent schema name is required")
	ErrTypeRequired = errors.New("agent schema type is required")
	ErrNameExists   = errors.New("agent schema name already exists")
	ErrNotFound     = errors.New("agent schema not found")
)

type Service struct {
	dao   dao.AgentSchemaDAO
	clock func() time.Time
}

type Input struct {
	AgentID     *string `json:"agentId"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Type        *string `json:"type"`
	Instruction *string `json:"instruction"`
	InputKeys   *string `json:"inputKeys"`
	OutputKey   *string `json:"outputKey"`
	Handle      *string `json:"handle"`
	SubAgents   *string `json:"subAgents"`
	YamlSchema  *string `json:"yamlSchema"`
	Status      *string `json:"status"`
	Enabled     *bool   `json:"enabled"`
}

type Schema struct {
	ID          int64     `json:"id"`
	AgentID     *string   `json:"agentId,omitempty"`
	WorkspaceID string    `json:"workspaceId"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	Type        string    `json:"type"`
	Instruction *string   `json:"instruction,omitempty"`
	InputKeys   *string   `json:"inputKeys,omitempty"`
	OutputKey   *string   `json:"outputKey,omitempty"`
	Handle      *string   `json:"handle,omitempty"`
	SubAgents   *string   `json:"subAgents,omitempty"`
	YamlSchema  *string   `json:"yamlSchema,omitempty"`
	Status      string    `json:"status"`
	Enabled     bool      `json:"enabled"`
	GmtCreate   time.Time `json:"gmtCreate"`
	GmtModified time.Time `json:"gmtModified"`
	Creator     string    `json:"creator"`
	Modifier    string    `json:"modifier"`
}

type Page struct {
	Current int64    `json:"current"`
	Size    int64    `json:"size"`
	Total   int64    `json:"total"`
	Records []Schema `json:"records"`
}

func NewService(data dao.AgentSchemaDAO, clock func() time.Time) *Service {
	if clock == nil {
		clock = time.Now
	}
	return &Service{dao: data, clock: clock}
}

func (s *Service) Create(ctx context.Context, workspaceID, accountID string, input Input) (*Schema, error) {
	name, err := required(input.Name, ErrNameRequired)
	if err != nil {
		return nil, err
	}
	typ, err := required(input.Type, ErrTypeRequired)
	if err != nil {
		return nil, err
	}
	if _, err = s.dao.FindByName(ctx, name, workspaceID); err == nil {
		return nil, ErrNameExists
	} else if !errors.Is(err, dao.ErrNotFound) {
		return nil, err
	}
	agentID := text(input.AgentID)
	if agentID == "" {
		agentID, err = newAgentID()
		if err != nil {
			return nil, err
		}
	}
	now := s.clock()
	value := dao.AgentSchema{AgentID: &agentID, WorkspaceID: workspaceID, Name: name, Description: clean(input.Description), Type: typ, Instruction: clean(input.Instruction), InputKeys: clean(input.InputKeys), OutputKey: clean(input.OutputKey), Handle: clean(input.Handle), SubAgents: clean(input.SubAgents), YamlSchema: clean(input.YamlSchema), Status: "active", Enabled: true, GmtCreate: now, GmtModified: now, Creator: accountID, Modifier: accountID}
	id, err := s.dao.Create(ctx, value)
	if err != nil {
		return nil, err
	}
	value.ID = id
	return toSchema(value), nil
}

func (s *Service) Get(ctx context.Context, workspaceID string, id int64) (*Schema, error) {
	value, err := s.dao.Find(ctx, id, workspaceID)
	if errors.Is(err, dao.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return toSchema(*value), nil
}

func (s *Service) List(ctx context.Context, workspaceID, name string) ([]Schema, error) {
	values, err := s.dao.List(ctx, workspaceID, strings.TrimSpace(name))
	if err != nil {
		return nil, err
	}
	result := make([]Schema, 0, len(values))
	for _, value := range values {
		result = append(result, *toSchema(value))
	}
	return result, nil
}

func (s *Service) Page(ctx context.Context, workspaceID, name string, current, size int64) (*Page, error) {
	if current < 1 {
		current = 1
	}
	if size < 1 {
		size = 10
	}
	if size > 100 {
		size = 100
	}
	values, err := s.List(ctx, workspaceID, name)
	if err != nil {
		return nil, err
	}
	total := int64(len(values))
	start := (current - 1) * size
	if start >= total {
		return &Page{Current: current, Size: size, Total: total, Records: []Schema{}}, nil
	}
	end := start + size
	if end > total {
		end = total
	}
	return &Page{Current: current, Size: size, Total: total, Records: values[start:end]}, nil
}

func (s *Service) Update(ctx context.Context, workspaceID, accountID string, id int64, input Input) (*Schema, error) {
	value, err := s.dao.Find(ctx, id, workspaceID)
	if errors.Is(err, dao.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if input.Name != nil {
		name, err := required(input.Name, ErrNameRequired)
		if err != nil {
			return nil, err
		}
		if other, err := s.dao.FindByName(ctx, name, workspaceID); err == nil && other.ID != id {
			return nil, ErrNameExists
		} else if err != nil && !errors.Is(err, dao.ErrNotFound) {
			return nil, err
		}
		value.Name = name
	}
	if input.Type != nil {
		typ, err := required(input.Type, ErrTypeRequired)
		if err != nil {
			return nil, err
		}
		value.Type = typ
	}
	merge(&value.Description, input.Description)
	merge(&value.Instruction, input.Instruction)
	merge(&value.InputKeys, input.InputKeys)
	merge(&value.OutputKey, input.OutputKey)
	merge(&value.Handle, input.Handle)
	merge(&value.SubAgents, input.SubAgents)
	merge(&value.YamlSchema, input.YamlSchema)
	if status := clean(input.Status); status != nil {
		value.Status = *status
	}
	if input.Enabled != nil {
		value.Enabled = *input.Enabled
	}
	value.GmtModified = s.clock()
	value.Modifier = accountID
	if err = s.dao.Update(ctx, *value); err != nil {
		return nil, err
	}
	return toSchema(*value), nil
}

func (s *Service) SetEnabled(ctx context.Context, workspaceID, accountID string, id int64, enabled bool) error {
	_, err := s.Update(ctx, workspaceID, accountID, id, Input{Enabled: &enabled})
	return err
}

func (s *Service) Delete(ctx context.Context, workspaceID string, id int64) error {
	if _, err := s.Get(ctx, workspaceID, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		return err
	}
	return s.dao.Delete(ctx, id, workspaceID)
}

func required(value *string, missing error) (string, error) {
	value = clean(value)
	if value == nil {
		return "", missing
	}
	return *value, nil
}

func clean(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func text(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func merge(target **string, source *string) {
	if source != nil {
		*target = clean(source)
	}
}

func toSchema(value dao.AgentSchema) *Schema {
	return &Schema{ID: value.ID, AgentID: value.AgentID, WorkspaceID: value.WorkspaceID, Name: value.Name, Description: value.Description, Type: value.Type, Instruction: value.Instruction, InputKeys: value.InputKeys, OutputKey: value.OutputKey, Handle: value.Handle, SubAgents: value.SubAgents, YamlSchema: value.YamlSchema, Status: value.Status, Enabled: value.Enabled, GmtCreate: value.GmtCreate, GmtModified: value.GmtModified, Creator: value.Creator, Modifier: value.Modifier}
}

func newAgentID() (string, error) {
	var value [12]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return "agent_" + hex.EncodeToString(value[:]), nil
}
