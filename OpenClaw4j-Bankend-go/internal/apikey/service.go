package apikey

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

const maxKeysPerAccount = 20

var (
	ErrDescriptionRequired = errors.New("api key description is required")
	ErrLimitExceeded       = errors.New("api key limit exceeded")
	ErrNotFound            = errors.New("api key not found")
)

type Cipher interface {
	Encrypt(plain string) (string, error)
	Decrypt(encrypted string) (string, error)
}

type Service struct {
	dao    dao.APIKeyDAO
	cipher Cipher
	clock  func() time.Time
}

type APIKey struct {
	ID          int64     `json:"id"`
	APIKey      string    `json:"api_key"`
	Description *string   `json:"description"`
	GmtCreate   time.Time `json:"gmt_create"`
}

type Page struct {
	Current int64    `json:"current"`
	Size    int64    `json:"size"`
	Total   int64    `json:"total"`
	Records []APIKey `json:"records"`
}

func NewService(dataAccess dao.APIKeyDAO, cipher Cipher, clock func() time.Time) *Service {
	if clock == nil {
		clock = time.Now
	}
	return &Service{dao: dataAccess, cipher: cipher, clock: clock}
}

func (s *Service) Create(ctx context.Context, accountID string, description string) (int64, error) {
	description = strings.TrimSpace(description)
	if description == "" {
		return 0, ErrDescriptionRequired
	}
	if s.dao == nil || s.cipher == nil {
		return 0, fmt.Errorf("api key service dependencies are unavailable")
	}
	count, err := s.dao.CountActiveByAccountID(ctx, accountID)
	if err != nil {
		return 0, fmt.Errorf("count active api keys: %w", err)
	}
	if count >= maxKeysPerAccount {
		return 0, ErrLimitExceeded
	}
	plain, err := newAPIKey()
	if err != nil {
		return 0, err
	}
	encrypted, err := s.cipher.Encrypt(plain)
	if err != nil {
		return 0, fmt.Errorf("encrypt api key: %w", err)
	}
	now := s.clock()
	return s.dao.Create(ctx, dao.APIKey{
		AccountID:   accountID,
		APIKey:      encrypted,
		Status:      1,
		Description: &description,
		GmtCreate:   now,
		GmtModified: now,
		Creator:     accountID,
		Modifier:    accountID,
	})
}

func (s *Service) Get(ctx context.Context, accountID string, id int64) (*APIKey, error) {
	key, err := s.findOwned(ctx, accountID, id)
	if err != nil {
		return nil, err
	}
	return s.toResponse(key, false)
}

func (s *Service) List(ctx context.Context, accountID string, current int64, size int64) (*Page, error) {
	if current < 1 {
		current = 1
	}
	if size < 1 {
		size = 10
	}
	if size > 100 {
		size = 100
	}
	keys, err := s.dao.ListActiveByAccountID(ctx, accountID, int32(size), int32((current-1)*size))
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	total, err := s.dao.CountActiveByAccountID(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("count api keys: %w", err)
	}
	items := make([]APIKey, 0, len(keys))
	for _, key := range keys {
		item, err := s.toResponse(&key, true)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return &Page{Current: current, Size: size, Total: total, Records: items}, nil
}

func (s *Service) Delete(ctx context.Context, accountID string, id int64) error {
	if _, err := s.findOwned(ctx, accountID, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		return err
	}
	if err := s.dao.SoftDelete(ctx, id, accountID, accountID, s.clock()); err != nil {
		return fmt.Errorf("soft delete api key: %w", err)
	}
	return nil
}

func (s *Service) UpdateDescription(ctx context.Context, accountID string, id int64, description string) error {
	description = strings.TrimSpace(description)
	if description == "" {
		return ErrDescriptionRequired
	}
	if _, err := s.findOwned(ctx, accountID, id); err != nil {
		return err
	}
	if err := s.dao.UpdateDescription(ctx, id, accountID, description, accountID, s.clock()); err != nil {
		return fmt.Errorf("update api key description: %w", err)
	}
	return nil
}

func (s *Service) findOwned(ctx context.Context, accountID string, id int64) (*dao.APIKey, error) {
	if s.dao == nil || s.cipher == nil {
		return nil, fmt.Errorf("api key service dependencies are unavailable")
	}
	key, err := s.dao.FindActiveByIDAndAccountID(ctx, id, accountID)
	if errors.Is(err, dao.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find api key: %w", err)
	}
	return key, nil
}

func (s *Service) toResponse(key *dao.APIKey, masked bool) (*APIKey, error) {
	plain, err := s.cipher.Decrypt(key.APIKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt api key: %w", err)
	}
	if masked {
		plain = mask(plain)
	}
	return &APIKey{ID: key.ID, APIKey: plain, Description: key.Description, GmtCreate: key.GmtCreate}, nil
}

func newAPIKey() (string, error) {
	var data [16]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", fmt.Errorf("generate api key: %w", err)
	}
	return "sk-" + hex.EncodeToString(data[:]), nil
}

func mask(value string) string {
	if len(strings.TrimSpace(value)) == 0 || len(value) <= 12 {
		return strings.Repeat("*", 16)
	}
	return value[:4] + strings.Repeat("*", len(value)-8) + value[len(value)-4:]
}
