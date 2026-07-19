package account

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/seaskyland/openclaw4j-backend-go/internal/auth"
	"github.com/seaskyland/openclaw4j-backend-go/internal/dao"
	"github.com/seaskyland/openclaw4j-backend-go/internal/workspace"
)

var (
	ErrPermissionDenied = errors.New("permission denied")
	ErrUsernameRequired = errors.New("username is required")
	ErrPasswordRequired = errors.New("password is required")
	ErrUsernameExists   = errors.New("account username already exists")
	ErrNotFound         = errors.New("account not found")
	ErrPasswordNotMatch = errors.New("account password does not match")
)

type Service struct {
	accounts   dao.AccountDAO
	workspaces *workspace.Service
	clock      func() time.Time
}
type Input struct {
	Username string  `json:"username"`
	Password string  `json:"password"`
	Email    *string `json:"email"`
	Mobile   *string `json:"mobile"`
	Nickname *string `json:"nickname"`
	Icon     *string `json:"icon"`
}
type Account struct {
	AccountID    string     `json:"account_id"`
	Username     string     `json:"username"`
	Email        *string    `json:"email"`
	Mobile       *string    `json:"mobile"`
	Nickname     *string    `json:"nickname"`
	Icon         *string    `json:"icon"`
	Type         string     `json:"type"`
	Status       int16      `json:"status"`
	GmtCreate    time.Time  `json:"gmt_create"`
	GmtModified  time.Time  `json:"gmt_modified"`
	GmtLastLogin *time.Time `json:"gmt_last_login"`
}
type Page struct {
	Current int64     `json:"current"`
	Size    int64     `json:"size"`
	Total   int64     `json:"total"`
	Records []Account `json:"records"`
}

func NewService(accounts dao.AccountDAO, workspaces *workspace.Service, clock func() time.Time) *Service {
	if clock == nil {
		clock = time.Now
	}
	return &Service{accounts: accounts, workspaces: workspaces, clock: clock}
}

func (s *Service) Create(ctx context.Context, operatorID string, input Input) (string, error) {
	if err := s.requireAdmin(ctx, operatorID); err != nil {
		return "", err
	}
	input.Username = strings.TrimSpace(input.Username)
	if input.Username == "" {
		return "", ErrUsernameRequired
	}
	if strings.TrimSpace(input.Password) == "" {
		return "", ErrPasswordRequired
	}
	if found, err := s.accounts.FindActiveByUsername(ctx, input.Username); err == nil && found != nil {
		return "", ErrUsernameExists
	} else if !errors.Is(err, dao.ErrNotFound) {
		return "", fmt.Errorf("find account by username: %w", err)
	}
	id, err := newAccountID()
	if err != nil {
		return "", err
	}
	password, err := auth.HashPassword(input.Password)
	if err != nil {
		return "", err
	}
	now := s.clock()
	if err := s.accounts.Create(ctx, dao.Account{AccountID: id, Username: input.Username, Email: input.Email, Mobile: input.Mobile, Password: password, Nickname: input.Nickname, Icon: input.Icon, Type: "user", Status: 1, GmtCreate: now, GmtModified: now, Creator: operatorID, Modifier: operatorID}); err != nil {
		return "", fmt.Errorf("create account: %w", err)
	}
	if s.workspaces == nil {
		return "", fmt.Errorf("workspace service is unavailable")
	}
	if _, err := s.workspaces.Create(ctx, id, "Default Workspace", stringPointer("Default workspace"), nil); err != nil {
		return "", fmt.Errorf("create default workspace: %w", err)
	}
	return id, nil
}

func (s *Service) Update(ctx context.Context, operatorID, accountID string, input Input) error {
	if err := s.requireAdmin(ctx, operatorID); err != nil {
		return err
	}
	existing, err := s.find(ctx, accountID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(input.Password) != "" {
		hash, err := auth.HashPassword(input.Password)
		if err != nil {
			return err
		}
		existing.Password = hash
	}
	existing.Email = input.Email
	existing.Mobile = input.Mobile
	existing.Nickname = input.Nickname
	existing.Icon = input.Icon
	existing.Modifier = operatorID
	existing.GmtModified = s.clock()
	if err := s.accounts.Update(ctx, *existing); err != nil {
		return fmt.Errorf("update account: %w", err)
	}
	return nil
}
func (s *Service) Delete(ctx context.Context, operatorID, accountID string) error {
	if err := s.requireAdmin(ctx, operatorID); err != nil {
		return err
	}
	if _, err := s.find(ctx, accountID); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		return err
	}
	if err := s.accounts.SoftDelete(ctx, accountID, operatorID, s.clock()); err != nil {
		return fmt.Errorf("soft delete account: %w", err)
	}
	return nil
}
func (s *Service) Get(ctx context.Context, accountID string) (*Account, error) {
	item, err := s.find(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return toResponse(item), nil
}
func (s *Service) List(ctx context.Context, operatorID, name string, current, size int64) (*Page, error) {
	if err := s.requireAdmin(ctx, operatorID); err != nil {
		return nil, err
	}
	if current < 1 {
		current = 1
	}
	if size < 1 {
		size = 10
	}
	if size > 100 {
		size = 100
	}
	items, err := s.accounts.ListActiveUsers(ctx, strings.TrimSpace(name), int32(size), int32((current-1)*size))
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	total, err := s.accounts.CountActiveUsers(ctx, strings.TrimSpace(name))
	if err != nil {
		return nil, fmt.Errorf("count accounts: %w", err)
	}
	records := make([]Account, 0, len(items))
	for _, item := range items {
		records = append(records, *toResponse(&item))
	}
	return &Page{Current: current, Size: size, Total: total, Records: records}, nil
}
func (s *Service) ChangePassword(ctx context.Context, accountID, password, newPassword string) error {
	if strings.TrimSpace(newPassword) == "" {
		return ErrPasswordRequired
	}
	existing, err := s.find(ctx, accountID)
	if err != nil {
		return err
	}
	if !auth.VerifyPassword(password, existing.Password) {
		return ErrPasswordNotMatch
	}
	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return err
	}
	existing.Password = hash
	existing.Modifier = accountID
	existing.GmtModified = s.clock()
	if err := s.accounts.Update(ctx, *existing); err != nil {
		return fmt.Errorf("change password: %w", err)
	}
	return nil
}
func (s *Service) requireAdmin(ctx context.Context, accountID string) error {
	operator, err := s.find(ctx, accountID)
	if err != nil {
		return err
	}
	if operator.Type != "admin" {
		return ErrPermissionDenied
	}
	return nil
}
func (s *Service) find(ctx context.Context, id string) (*dao.Account, error) {
	if s.accounts == nil {
		return nil, fmt.Errorf("account service dependencies are unavailable")
	}
	item, err := s.accounts.FindActiveByAccountID(ctx, id)
	if errors.Is(err, dao.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find account: %w", err)
	}
	return item, nil
}
func toResponse(item *dao.Account) *Account {
	return &Account{AccountID: item.AccountID, Username: item.Username, Email: item.Email, Mobile: item.Mobile, Nickname: item.Nickname, Icon: item.Icon, Type: item.Type, Status: item.Status, GmtCreate: item.GmtCreate, GmtModified: item.GmtModified, GmtLastLogin: item.GmtLastLogin}
}
func newAccountID() (string, error) {
	var data [12]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", fmt.Errorf("generate account id: %w", err)
	}
	return hex.EncodeToString(data[:]), nil
}
func stringPointer(value string) *string { return &value }
