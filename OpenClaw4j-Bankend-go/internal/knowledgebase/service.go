package knowledgebase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"github.com/seaskyland/openclaw4j-backend-go/internal/dao"
	"strings"
	"time"
)

var (
	ErrNameRequired        = errors.New("knowledge base name is required")
	ErrIndexConfigRequired = errors.New("index_config is required")
	ErrNotFound            = errors.New("knowledge base not found")
)

type Service struct {
	dao   dao.KnowledgeBaseDAO
	clock func() time.Time
}
type Input struct {
	Name          string  `json:"name"`
	Description   *string `json:"description"`
	Type          string  `json:"type"`
	ProcessConfig *string `json:"process_config"`
	IndexConfig   *string `json:"index_config"`
	SearchConfig  *string `json:"search_config"`
}
type KnowledgeBase struct {
	KbID          string    `json:"kb_id"`
	Name          string    `json:"name"`
	Description   *string   `json:"description,omitempty"`
	Type          string    `json:"type"`
	ProcessConfig *string   `json:"process_config,omitempty"`
	IndexConfig   *string   `json:"index_config,omitempty"`
	SearchConfig  *string   `json:"search_config,omitempty"`
	TotalDocs     int64     `json:"total_docs"`
	GmtModified   time.Time `json:"gmt_modified"`
}
type Page struct {
	Current int64           `json:"current"`
	Size    int64           `json:"size"`
	Total   int64           `json:"total"`
	Records []KnowledgeBase `json:"records"`
}

func NewService(d dao.KnowledgeBaseDAO, c func() time.Time) *Service {
	if c == nil {
		c = time.Now
	}
	return &Service{dao: d, clock: c}
}
func (s *Service) Create(c context.Context, w, a string, in Input) (string, error) {
	if strings.TrimSpace(in.Name) == "" {
		return "", ErrNameRequired
	}
	if in.IndexConfig == nil || strings.TrimSpace(*in.IndexConfig) == "" {
		return "", ErrIndexConfigRequired
	}
	b := make([]byte, 12)
	if _, e := rand.Read(b); e != nil {
		return "", e
	}
	id := "kb_" + hex.EncodeToString(b)
	n := s.clock()
	e := s.dao.Create(c, dao.KnowledgeBase{WorkspaceID: w, KbID: id, Type: defaultType(in.Type), Status: 1, Name: strings.TrimSpace(in.Name), Description: in.Description, ProcessConfig: in.ProcessConfig, IndexConfig: in.IndexConfig, SearchConfig: in.SearchConfig, GmtCreate: n, GmtModified: n, Creator: a, Modifier: a})
	return id, e
}
func (s *Service) Get(c context.Context, w, id string) (*KnowledgeBase, error) {
	v, e := s.dao.Find(c, id, w)
	if errors.Is(e, dao.ErrNotFound) {
		return nil, ErrNotFound
	}
	if e != nil {
		return nil, e
	}
	return mapKB(*v), nil
}
func (s *Service) List(c context.Context, w, n string, cur, size int64) (*Page, error) {
	if cur < 1 {
		cur = 1
	}
	if size < 1 {
		size = 10
	}
	vs, e := s.dao.List(c, w, strings.TrimSpace(n))
	if e != nil {
		return nil, e
	}
	out := make([]KnowledgeBase, 0, len(vs))
	for _, v := range vs {
		out = append(out, *mapKB(v))
	}
	return &Page{Current: cur, Size: size, Total: int64(len(out)), Records: out}, nil
}
func (s *Service) Update(c context.Context, w, a, id string, in Input) error {
	v, e := s.dao.Find(c, id, w)
	if errors.Is(e, dao.ErrNotFound) {
		return ErrNotFound
	}
	if e != nil {
		return e
	}
	if strings.TrimSpace(in.Name) == "" {
		return ErrNameRequired
	}
	v.Name = strings.TrimSpace(in.Name)
	v.Type = defaultType(in.Type)
	v.Description = in.Description
	v.ProcessConfig = in.ProcessConfig
	v.IndexConfig = in.IndexConfig
	v.SearchConfig = in.SearchConfig
	v.GmtModified = s.clock()
	v.Modifier = a
	return s.dao.Update(c, *v)
}
func (s *Service) Delete(c context.Context, w, a, id string) error {
	return s.dao.Delete(c, id, w, a, s.clock())
}
func (s *Service) ListByCodes(c context.Context, workspaceID string, kbIDs []string) ([]KnowledgeBase, error) {
	values, err := s.dao.List(c, workspaceID, "")
	if err != nil {
		return nil, err
	}
	selected := make(map[string]struct{}, len(kbIDs))
	for _, kbID := range kbIDs {
		selected[strings.TrimSpace(kbID)] = struct{}{}
	}
	result := make([]KnowledgeBase, 0, len(kbIDs))
	for _, value := range values {
		if _, ok := selected[value.KbID]; ok {
			result = append(result, *mapKB(value))
		}
	}
	return result, nil
}
func mapKB(v dao.KnowledgeBase) *KnowledgeBase {
	return &KnowledgeBase{KbID: v.KbID, Name: v.Name, Description: v.Description, Type: v.Type, ProcessConfig: v.ProcessConfig, IndexConfig: v.IndexConfig, SearchConfig: v.SearchConfig, TotalDocs: v.TotalDocs, GmtModified: v.GmtModified}
}
func defaultType(v string) string {
	if strings.TrimSpace(v) == "" {
		return "unstructured"
	}
	return strings.TrimSpace(v)
}
