package workspace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/seaskyland/openclaw4j-backend-go/internal/dao"
)

const maxWorkspacesPerAccount = 10

var (
	ErrNameRequired  = errors.New("workspace name is required")
	ErrNameExists    = errors.New("workspace name already exists")
	ErrLimitExceeded = errors.New("workspace limit exceeded")
	ErrNotFound      = errors.New("workspace not found")
)

type Service struct {
	dao   dao.WorkspaceDAO
	clock func() time.Time
}

type Workspace struct {
	WorkspaceID string    `json:"workspace_id"`
	AccountID   string    `json:"account_id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	Config      *string   `json:"config"`
	GmtCreate   time.Time `json:"gmt_create"`
}

type Page struct {
	Current int64       `json:"current"`
	Size    int64       `json:"size"`
	Total   int64       `json:"total"`
	Records []Workspace `json:"records"`
}

func NewService(dataAccess dao.WorkspaceDAO, clock func() time.Time) *Service {
	if clock == nil {
		clock = time.Now
	}
	return &Service{dao: dataAccess, clock: clock}
}

func (s *Service) Create(ctx context.Context, accountID, name string, description, config *string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrNameRequired
	}
	if s.dao == nil {
		return "", fmt.Errorf("workspace service dependencies are unavailable")
	}
	if existing, err := s.dao.FindActiveByNameAndAccountID(ctx, name, accountID); err == nil && existing != nil {
		return "", ErrNameExists
	} else if !errors.Is(err, dao.ErrNotFound) {
		return "", fmt.Errorf("find workspace by name: %w", err)
	}
	count, err := s.dao.CountActiveByAccountID(ctx, accountID)
	if err != nil {
		return "", fmt.Errorf("count workspaces: %w", err)
	}
	if count >= maxWorkspacesPerAccount {
		return "", ErrLimitExceeded
	}
	id, err := newWorkspaceID()
	if err != nil {
		return "", err
	}
	now := s.clock()
	err = s.dao.Create(ctx, dao.Workspace{WorkspaceID: id, AccountID: accountID, Status: 1, Name: name, Description: description, Config: config, GmtCreate: now, GmtModified: now, Creator: accountID, Modifier: accountID})
	if err != nil {
		return "", fmt.Errorf("create workspace: %w", err)
	}
	return id, nil
}

func (s *Service) Get(ctx context.Context, accountID, workspaceID string) (*Workspace, error) {
	item, err := s.findOwned(ctx, accountID, workspaceID)
	if err != nil {
		return nil, err
	}
	return toResponse(item), nil
}

func (s *Service) List(ctx context.Context, accountID string, current, size int64) (*Page, error) {
	if current < 1 {
		current = 1
	}
	if size < 1 {
		size = 10
	}
	if size > 100 {
		size = 100
	}
	items, err := s.dao.ListActiveByAccountID(ctx, accountID, int32(size), int32((current-1)*size))
	if err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}
	total, err := s.dao.CountActiveByAccountID(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("count workspaces: %w", err)
	}
	records := make([]Workspace, 0, len(items))
	for _, item := range items {
		records = append(records, *toResponse(&item))
	}
	return &Page{Current: current, Size: size, Total: total, Records: records}, nil
}

func (s *Service) Update(ctx context.Context, accountID, workspaceID, name string, description, config *string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrNameRequired
	}
	current, err := s.findOwned(ctx, accountID, workspaceID)
	if err != nil {
		return err
	}
	if existing, err := s.dao.FindActiveByNameAndAccountID(ctx, name, accountID); err == nil && existing != nil && existing.WorkspaceID != current.WorkspaceID {
		return ErrNameExists
	} else if !errors.Is(err, dao.ErrNotFound) {
		return fmt.Errorf("find workspace by name: %w", err)
	}
	current.Name = name
	current.Description = description
	current.Config = config
	current.Modifier = accountID
	current.GmtModified = s.clock()
	if err := s.dao.Update(ctx, *current); err != nil {
		return fmt.Errorf("update workspace: %w", err)
	}
	return nil
}

func (s *Service) Delete(ctx context.Context, accountID, workspaceID string) error {
	if _, err := s.findOwned(ctx, accountID, workspaceID); err != nil {
		return err
	}
	if err := s.dao.SoftDelete(ctx, workspaceID, accountID, accountID, s.clock()); err != nil {
		return fmt.Errorf("soft delete workspace: %w", err)
	}
	return nil
}

func (s *Service) findOwned(ctx context.Context, accountID, workspaceID string) (*dao.Workspace, error) {
	if s.dao == nil {
		return nil, fmt.Errorf("workspace service dependencies are unavailable")
	}
	item, err := s.dao.FindActiveByIDAndAccountID(ctx, workspaceID, accountID)
	if errors.Is(err, dao.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find workspace: %w", err)
	}
	return item, nil
}
func toResponse(item *dao.Workspace) *Workspace {
	return &Workspace{WorkspaceID: item.WorkspaceID, AccountID: item.AccountID, Name: item.Name, Description: item.Description, Config: item.Config, GmtCreate: item.GmtCreate}
}
func newWorkspaceID() (string, error) {
	var data [12]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", fmt.Errorf("generate workspace id: %w", err)
	}
	return hex.EncodeToString(data[:]), nil
}
