package document

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/seaskyland/openclaw4j-backend-go/internal/dao"
	"github.com/seaskyland/openclaw4j-backend-go/internal/rag"
)

const (
	statusNormal          int16 = 1
	indexStatusUploaded   int16 = 1
	indexStatusProcessing int16 = 2
	indexStatusProcessed  int16 = 3
	indexStatusFailed     int16 = 4
)

var (
	ErrFilesRequired      = errors.New("files is required")
	ErrDocumentNotFound   = errors.New("document not found")
	ErrNameRequired       = errors.New("document name is required")
	ErrTextRequired       = errors.New("text is required")
	ErrChunkNotFound      = errors.New("document chunk not found")
	ErrIndexNotConfigured = errors.New("knowledge base vector index is not configured")
)

type FileInput struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Extension   string `json:"extension"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}

type CreateInput struct {
	Type          string          `json:"type"`
	Files         []FileInput     `json:"files"`
	ProcessConfig json.RawMessage `json:"process_config"`
}

type UpdateInput struct {
	Name          *string         `json:"name"`
	Format        *string         `json:"format"`
	Size          *int64          `json:"size"`
	Metadata      json.RawMessage `json:"metadata"`
	Enabled       *bool           `json:"enabled"`
	Path          *string         `json:"path"`
	ProcessConfig json.RawMessage `json:"process_config"`
	Source        *string         `json:"source"`
}

type ReIndexInput struct {
	ProcessConfig json.RawMessage `json:"process_config"`
}

type ChunkInput struct {
	DocName    *string  `json:"doc_name"`
	Title      *string  `json:"title"`
	Text       string   `json:"text"`
	Score      *float64 `json:"score"`
	PageNumber *int32   `json:"page_number"`
	Enabled    *bool    `json:"enabled"`
}

type ChunkStatusInput struct {
	ChunkIDs []string `json:"chunk_ids"`
	Enabled  *bool    `json:"enabled"`
}

type Document struct {
	DocID         string    `json:"doc_id"`
	KbID          string    `json:"kb_id"`
	Type          string    `json:"type"`
	Enabled       bool      `json:"enabled"`
	Name          string    `json:"name"`
	Format        string    `json:"format"`
	Size          int64     `json:"size"`
	Metadata      *string   `json:"metadata,omitempty"`
	IndexStatus   int16     `json:"index_status"`
	Path          string    `json:"path"`
	ParsedPath    *string   `json:"parsed_path,omitempty"`
	ProcessConfig *string   `json:"process_config,omitempty"`
	Source        *string   `json:"source,omitempty"`
	Error         *string   `json:"error,omitempty"`
	GmtModified   time.Time `json:"gmt_modified"`
}

type Page struct {
	Current int64      `json:"current"`
	Size    int64      `json:"size"`
	Total   int64      `json:"total"`
	Records []Document `json:"records"`
}

type Chunk struct {
	ChunkID    string   `json:"chunk_id"`
	DocID      string   `json:"doc_id"`
	DocName    string   `json:"doc_name"`
	Title      string   `json:"title"`
	Text       string   `json:"text"`
	Score      *float64 `json:"score,omitempty"`
	PageNumber *int32   `json:"page_number,omitempty"`
	Enabled    bool     `json:"enabled"`
}

type Service struct {
	dao            dao.DocumentDAO
	knowledgeBases dao.KnowledgeBaseDAO
	embedder       rag.Embedder
	vectors        rag.VectorStore
	fileStorageDir string
	clock          func() time.Time
}

func NewService(data dao.DocumentDAO, clock func() time.Time) *Service {
	return NewServiceWithRAG(data, nil, nil, nil, "", clock)
}

// NewServiceWithRAG keeps document metadata ownership in this service while the
// embedding provider and vector database remain independently replaceable.
func NewServiceWithRAG(data dao.DocumentDAO, knowledgeBases dao.KnowledgeBaseDAO, embedder rag.Embedder, vectors rag.VectorStore, fileStorageDir string, clock func() time.Time) *Service {
	if clock == nil {
		clock = time.Now
	}
	return &Service{dao: data, knowledgeBases: knowledgeBases, embedder: embedder, vectors: vectors, fileStorageDir: strings.TrimSpace(fileStorageDir), clock: clock}
}

func (s *Service) Create(ctx context.Context, workspaceID, accountID, kbID string, input CreateInput) ([]string, error) {
	if len(input.Files) == 0 {
		return nil, ErrFilesRequired
	}
	created := make([]string, 0, len(input.Files))
	for _, file := range input.Files {
		if strings.TrimSpace(file.Name) == "" {
			return nil, ErrNameRequired
		}
		id, err := newDocumentID()
		if err != nil {
			return nil, err
		}
		now := s.clock()
		format := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(file.Extension)), ".")
		if format == "" {
			format = strings.TrimPrefix(strings.ToLower(filepath.Ext(file.Name)), ".")
		}
		metadata := jsonText(map[string]string{"content_type": strings.TrimSpace(file.ContentType)})
		processConfig := rawJSONText(input.ProcessConfig)
		path := strings.TrimSpace(file.Path)
		value := dao.Document{WorkspaceID: workspaceID, KbID: kbID, DocID: id, Type: defaultType(input.Type), Status: statusNormal, Enabled: true, Name: strings.TrimSpace(file.Name), Format: format, Size: file.Size, Metadata: metadata, IndexStatus: indexStatusUploaded, Path: path, ProcessConfig: processConfig, Source: stringPointer(path), GmtCreate: now, GmtModified: now, Creator: accountID, Modifier: accountID}
		if err := s.dao.Create(ctx, value); err != nil {
			return nil, err
		}
		created = append(created, id)
	}
	return created, nil
}

func (s *Service) Get(ctx context.Context, workspaceID, kbID, docID string) (*Document, error) {
	value, err := s.dao.Find(ctx, workspaceID, kbID, docID)
	if errors.Is(err, dao.ErrNotFound) {
		return nil, ErrDocumentNotFound
	}
	if err != nil {
		return nil, err
	}
	return mapDocument(*value), nil
}

func (s *Service) List(ctx context.Context, workspaceID, kbID, name string, indexStatus, current, size int64) (*Page, error) {
	if current < 1 {
		current = 1
	}
	if size < 1 {
		size = 10
	}
	if size > 100 {
		size = 100
	}
	values, err := s.dao.List(ctx, workspaceID, kbID, strings.TrimSpace(name), int16(indexStatus))
	if err != nil {
		return nil, err
	}
	items := make([]Document, 0, len(values))
	for _, value := range values {
		items = append(items, *mapDocument(value))
	}
	start := (current - 1) * size
	if start >= int64(len(items)) {
		return &Page{Current: current, Size: size, Total: int64(len(items)), Records: []Document{}}, nil
	}
	end := start + size
	if end > int64(len(items)) {
		end = int64(len(items))
	}
	return &Page{Current: current, Size: size, Total: int64(len(items)), Records: items[start:end]}, nil
}

func (s *Service) Update(ctx context.Context, workspaceID, accountID, kbID, docID string, input UpdateInput) error {
	value, err := s.find(ctx, workspaceID, kbID, docID)
	if err != nil {
		return err
	}
	if input.Name != nil {
		if strings.TrimSpace(*input.Name) == "" {
			return ErrNameRequired
		}
		value.Name = strings.TrimSpace(*input.Name)
	}
	if input.Format != nil {
		value.Format = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(*input.Format)), ".")
	}
	if input.Size != nil {
		value.Size = *input.Size
	}
	if input.Enabled != nil {
		value.Enabled = *input.Enabled
	}
	if input.Path != nil {
		value.Path = strings.TrimSpace(*input.Path)
	}
	if input.Source != nil {
		value.Source = trimmedPointer(input.Source)
	}
	if len(input.Metadata) > 0 {
		value.Metadata = rawJSONText(input.Metadata)
	}
	if len(input.ProcessConfig) > 0 {
		value.ProcessConfig = rawJSONText(input.ProcessConfig)
	}
	value.Modifier, value.GmtModified = accountID, s.clock()
	return s.dao.Update(ctx, *value)
}

func (s *Service) ReIndex(ctx context.Context, workspaceID, accountID, kbID, docID string, input ReIndexInput) error {
	value, err := s.find(ctx, workspaceID, kbID, docID)
	if err != nil {
		return err
	}
	if len(input.ProcessConfig) > 0 {
		value.ProcessConfig = rawJSONText(input.ProcessConfig)
	}
	value.IndexStatus, value.Error, value.Modifier, value.GmtModified = indexStatusProcessing, nil, accountID, s.clock()
	if err := s.dao.Update(ctx, *value); err != nil {
		return err
	}
	if err := s.indexDocument(ctx, value, accountID); err != nil {
		message := err.Error()
		value.IndexStatus, value.Error, value.GmtModified = indexStatusFailed, &message, s.clock()
		_ = s.dao.Update(ctx, *value)
		return err
	}
	value.IndexStatus, value.Error, value.GmtModified = indexStatusProcessed, nil, s.clock()
	return s.dao.Update(ctx, *value)
}

func (s *Service) Delete(ctx context.Context, workspaceID, accountID, kbID, docID string) error {
	if _, err := s.find(ctx, workspaceID, kbID, docID); err != nil {
		return err
	}
	return s.dao.Delete(ctx, workspaceID, kbID, docID, accountID, s.clock())
}

func (s *Service) DeleteBatch(ctx context.Context, workspaceID, accountID, kbID string, docIDs []string) error {
	for _, docID := range docIDs {
		if err := s.Delete(ctx, workspaceID, accountID, kbID, strings.TrimSpace(docID)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) CreateChunk(ctx context.Context, workspaceID, accountID, docID string, input ChunkInput) (string, error) {
	if strings.TrimSpace(input.Text) == "" {
		return "", ErrTextRequired
	}
	doc, err := s.documentByID(ctx, workspaceID, docID)
	if err != nil {
		return "", err
	}
	chunkID, err := newChunkID()
	if err != nil {
		return "", err
	}
	now := s.clock()
	docName := doc.Name
	if input.DocName != nil && strings.TrimSpace(*input.DocName) != "" {
		docName = strings.TrimSpace(*input.DocName)
	}
	title := ""
	if input.Title != nil {
		title = strings.TrimSpace(*input.Title)
	}
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	value := dao.DocumentChunk{WorkspaceID: workspaceID, KbID: doc.KbID, DocID: docID, ChunkID: chunkID, DocName: docName, Title: title, Text: input.Text, Score: input.Score, PageNumber: input.PageNumber, Enabled: enabled, GmtCreate: now, GmtModified: now, Creator: accountID, Modifier: accountID}
	if err := s.dao.CreateChunk(ctx, value); err != nil {
		return "", err
	}
	return chunkID, nil
}

func (s *Service) UpdateChunk(ctx context.Context, workspaceID, accountID, docID, chunkID string, input ChunkInput) error {
	if strings.TrimSpace(input.Text) == "" {
		return ErrTextRequired
	}
	value, err := s.dao.FindChunk(ctx, workspaceID, docID, chunkID)
	if errors.Is(err, dao.ErrNotFound) {
		return ErrChunkNotFound
	}
	if err != nil {
		return err
	}
	if input.Title != nil {
		value.Title = strings.TrimSpace(*input.Title)
	}
	if input.Score != nil {
		value.Score = input.Score
	}
	if input.PageNumber != nil {
		value.PageNumber = input.PageNumber
	}
	if input.Enabled != nil {
		value.Enabled = *input.Enabled
	}
	value.Text, value.Modifier, value.GmtModified = input.Text, accountID, s.clock()
	return s.dao.UpdateChunk(ctx, *value)
}

func (s *Service) DeleteChunks(ctx context.Context, workspaceID, accountID, docID string, chunkIDs []string) error {
	if len(chunkIDs) == 0 {
		return ErrChunkNotFound
	}
	if _, err := s.documentByID(ctx, workspaceID, docID); err != nil {
		return err
	}
	return s.dao.DeleteChunks(ctx, workspaceID, docID, chunkIDs, accountID, s.clock())
}

func (s *Service) ListChunks(ctx context.Context, workspaceID, docID string, current, size int64) (*PageChunks, error) {
	if current < 1 {
		current = 1
	}
	if size < 1 {
		size = 10
	}
	if size > 100 {
		size = 100
	}
	if _, err := s.documentByID(ctx, workspaceID, docID); err != nil {
		return nil, err
	}
	values, err := s.dao.ListChunks(ctx, workspaceID, docID)
	if err != nil {
		return nil, err
	}
	items := make([]Chunk, 0, len(values))
	for _, value := range values {
		items = append(items, mapChunk(value))
	}
	start := (current - 1) * size
	if start >= int64(len(items)) {
		return &PageChunks{Current: current, Size: size, Total: int64(len(items)), Records: []Chunk{}}, nil
	}
	end := start + size
	if end > int64(len(items)) {
		end = int64(len(items))
	}
	return &PageChunks{Current: current, Size: size, Total: int64(len(items)), Records: items[start:end]}, nil
}

type PageChunks struct {
	Current int64   `json:"current"`
	Size    int64   `json:"size"`
	Total   int64   `json:"total"`
	Records []Chunk `json:"records"`
}

func (s *Service) PreviewChunks(ctx context.Context, workspaceID, docID string) ([]Chunk, error) {
	page, err := s.ListChunks(ctx, workspaceID, docID, 1, 100)
	if err != nil {
		return nil, err
	}
	return page.Records, nil
}

// Search 是工作流与 Agent 共享的检索入口。Retriever 当前提供带 BM25 评分的可解释
// fallback；后续接入 pgvector 时只需替换 Source，不必把向量库细节泄漏给节点执行器。
func (s *Service) Search(ctx context.Context, workspaceID string, knowledgeBaseIDs []string, query string, limit int) ([]Chunk, error) {
	if s.embedder != nil && s.vectors != nil && s.knowledgeBases != nil {
		if matches, used, err := s.vectorSearch(ctx, workspaceID, knowledgeBaseIDs, query, limit); err != nil {
			return nil, err
		} else if used {
			return matches, nil
		}
	}
	result, err := rag.New(documentChunkSource{service: s}).Retrieve(ctx, rag.Request{WorkspaceID: workspaceID, KnowledgeBaseIDs: knowledgeBaseIDs, Query: query, Limit: limit})
	if err != nil {
		return nil, err
	}
	chunks := make([]Chunk, 0, len(result))
	for _, item := range result {
		chunk := Chunk{ChunkID: item.Chunk.ID, DocID: item.Chunk.Document, Title: item.Chunk.Title, Text: item.Context, Enabled: item.Chunk.Enabled}
		chunks = append(chunks, chunk)
	}
	return chunks, nil
}

type indexConfig struct {
	Name              string `json:"name"`
	EmbeddingProvider string `json:"embedding_provider"`
	EmbeddingModel    string `json:"embedding_model"`
}
type processConfig struct {
	ChunkSize    int `json:"chunk_size"`
	ChunkOverlap int `json:"chunk_overlap"`
}

func (s *Service) indexDocument(ctx context.Context, document *dao.Document, accountID string) error {
	if s.embedder == nil || s.vectors == nil || s.knowledgeBases == nil {
		return errors.New("RAG indexer is unavailable")
	}
	config, err := s.loadIndexConfig(ctx, document.WorkspaceID, document.KbID)
	if err != nil {
		return err
	}
	chunks, err := s.PreviewChunks(ctx, document.WorkspaceID, document.DocID)
	if err != nil {
		return err
	}
	if len(chunks) == 0 {
		texts, err := s.loadAndSplit(document)
		if err != nil {
			return err
		}
		for _, text := range texts {
			chunkID, err := s.CreateChunk(ctx, document.WorkspaceID, accountID, document.DocID, ChunkInput{Text: text})
			if err != nil {
				return err
			}
			chunks = append(chunks, Chunk{ChunkID: chunkID, DocID: document.DocID, DocName: document.Name, Text: text, Enabled: true})
		}
	}
	texts := make([]string, 0, len(chunks))
	active := make([]Chunk, 0, len(chunks))
	for _, chunk := range chunks {
		if chunk.Enabled && strings.TrimSpace(chunk.Text) != "" {
			active = append(active, chunk)
			texts = append(texts, chunk.Text)
		}
	}
	if len(active) == 0 {
		return nil
	}
	embeddings, err := s.embedder.Embed(ctx, document.WorkspaceID, config.EmbeddingProvider, config.EmbeddingModel, texts)
	if err != nil {
		return err
	}
	vectors := make([]rag.VectorDocument, 0, len(active))
	for index, chunk := range active {
		vectors = append(vectors, rag.VectorDocument{ID: chunk.ChunkID, WorkspaceID: document.WorkspaceID, KnowledgeBaseID: document.KbID, DocumentID: document.DocID, Content: chunk.Text, Enabled: true, Metadata: map[string]any{"doc_name": chunk.DocName, "title": chunk.Title, "page_number": chunk.PageNumber}, Embedding: embeddings[index]})
	}
	return s.vectors.Upsert(ctx, config.Name, vectors)
}

func (s *Service) vectorSearch(ctx context.Context, workspaceID string, knowledgeBaseIDs []string, query string, limit int) ([]Chunk, bool, error) {
	if strings.TrimSpace(query) == "" {
		return nil, false, nil
	}
	items := make([]Chunk, 0)
	used := false
	for _, kbID := range knowledgeBaseIDs {
		config, err := s.loadIndexConfig(ctx, workspaceID, kbID)
		if errors.Is(err, ErrIndexNotConfigured) {
			continue
		}
		if err != nil {
			if errors.Is(err, dao.ErrNotFound) {
				continue
			}
			return nil, false, err
		}
		used = true
		embedding, err := s.embedder.Embed(ctx, workspaceID, config.EmbeddingProvider, config.EmbeddingModel, []string{query})
		if err != nil {
			return nil, true, err
		}
		matches, err := s.vectors.Search(ctx, config.Name, workspaceID, []string{kbID}, embedding[0], limit)
		if err != nil {
			return nil, true, err
		}
		for _, match := range matches {
			score := match.Score
			item := Chunk{ChunkID: match.ID, DocID: match.DocumentID, Text: match.Content, Enabled: match.Enabled, Score: &score}
			if name, ok := match.Metadata["doc_name"].(string); ok {
				item.DocName = name
			}
			if title, ok := match.Metadata["title"].(string); ok {
				item.Title = title
			}
			items = append(items, item)
		}
	}
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, used, nil
}

func (s *Service) loadIndexConfig(ctx context.Context, workspaceID, kbID string) (indexConfig, error) {
	value, err := s.knowledgeBases.Find(ctx, kbID, workspaceID)
	if err != nil {
		return indexConfig{}, err
	}
	if value.IndexConfig == nil {
		return indexConfig{}, ErrIndexNotConfigured
	}
	var config indexConfig
	if err := json.Unmarshal([]byte(*value.IndexConfig), &config); err != nil {
		return indexConfig{}, fmt.Errorf("decode knowledge base index_config: %w", err)
	}
	if strings.TrimSpace(config.Name) == "" || strings.TrimSpace(config.EmbeddingProvider) == "" || strings.TrimSpace(config.EmbeddingModel) == "" {
		return indexConfig{}, fmt.Errorf("%w: index_config requires name, embedding_provider and embedding_model", ErrIndexNotConfigured)
	}
	return config, nil
}

func (s *Service) loadAndSplit(document *dao.Document) ([]string, error) {
	if document.Format != "txt" && document.Format != "md" && document.Format != "markdown" {
		return nil, fmt.Errorf("RAG indexing currently supports existing chunks, txt and markdown documents; received %s", document.Format)
	}
	path := document.Path
	if !filepath.IsAbs(path) {
		path = filepath.Join(s.fileStorageDir, path)
	}
	if s.fileStorageDir != "" {
		root, err := filepath.Abs(s.fileStorageDir)
		if err != nil {
			return nil, err
		}
		resolved, err := filepath.Abs(path)
		if err != nil || !strings.HasPrefix(resolved, root+string(os.PathSeparator)) {
			return nil, errors.New("document path escapes file storage directory")
		}
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read document source: %w", err)
	}
	var config processConfig
	if document.ProcessConfig != nil {
		_ = json.Unmarshal([]byte(*document.ProcessConfig), &config)
	}
	return rag.SplitText(string(contents), config.ChunkSize, config.ChunkOverlap), nil
}

type documentChunkSource struct{ service *Service }

func (s documentChunkSource) SearchChunks(ctx context.Context, workspaceID string, knowledgeBaseIDs []string) ([]rag.Chunk, error) {
	result := make([]rag.Chunk, 0)
	for _, knowledgeBaseID := range knowledgeBaseIDs {
		page, err := s.service.List(ctx, workspaceID, knowledgeBaseID, "", -1, 1, 100)
		if err != nil {
			return nil, err
		}
		for _, document := range page.Records {
			chunks, err := s.service.PreviewChunks(ctx, workspaceID, document.DocID)
			if err != nil {
				return nil, err
			}
			for _, chunk := range chunks {
				result = append(result, rag.Chunk{ID: chunk.ChunkID, Document: chunk.DocID, Title: chunk.Title, Text: chunk.Text, Enabled: chunk.Enabled})
			}
		}
	}
	return result, nil
}

func (s *Service) SetChunksEnabled(ctx context.Context, workspaceID, accountID, docID string, input ChunkStatusInput) error {
	if len(input.ChunkIDs) == 0 {
		return ErrChunkNotFound
	}
	if _, err := s.documentByID(ctx, workspaceID, docID); err != nil {
		return err
	}
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	return s.dao.SetChunksEnabled(ctx, workspaceID, docID, input.ChunkIDs, enabled, accountID, s.clock())
}

func (s *Service) find(ctx context.Context, workspaceID, kbID, docID string) (*dao.Document, error) {
	value, err := s.dao.Find(ctx, workspaceID, kbID, docID)
	if errors.Is(err, dao.ErrNotFound) {
		return nil, ErrDocumentNotFound
	}
	return value, err
}

func (s *Service) documentByID(ctx context.Context, workspaceID, docID string) (*dao.Document, error) {
	value, err := s.dao.FindByDocID(ctx, workspaceID, docID)
	if errors.Is(err, dao.ErrNotFound) {
		return nil, ErrDocumentNotFound
	}
	return value, err
}

func mapDocument(value dao.Document) *Document {
	return &Document{DocID: value.DocID, KbID: value.KbID, Type: value.Type, Enabled: value.Enabled, Name: value.Name, Format: value.Format, Size: value.Size, Metadata: value.Metadata, IndexStatus: value.IndexStatus, Path: value.Path, ParsedPath: value.ParsedPath, ProcessConfig: value.ProcessConfig, Source: value.Source, Error: value.Error, GmtModified: value.GmtModified}
}

func mapChunk(value dao.DocumentChunk) Chunk {
	return Chunk{ChunkID: value.ChunkID, DocID: value.DocID, DocName: value.DocName, Title: value.Title, Text: value.Text, Score: value.Score, PageNumber: value.PageNumber, Enabled: value.Enabled}
}

func defaultType(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "file"
	}
	return value
}
func newDocumentID() (string, error) {
	data := make([]byte, 12)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return "doc_" + hex.EncodeToString(data), nil
}
func newChunkID() (string, error) {
	data := make([]byte, 12)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return "chunk_" + hex.EncodeToString(data), nil
}
func stringPointer(value string) *string { return &value }
func trimmedPointer(value *string) *string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	value = stringPointer(strings.TrimSpace(*value))
	return value
}
func rawJSONText(value json.RawMessage) *string {
	if len(value) == 0 || string(value) == "null" {
		return nil
	}
	var target any
	if json.Unmarshal(value, &target) != nil {
		return nil
	}
	text, err := json.Marshal(target)
	if err != nil {
		return nil
	}
	result := string(text)
	return &result
}
func jsonText(value any) *string {
	text, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	result := string(text)
	return &result
}

var _ = []int16{indexStatusProcessing, indexStatusProcessed, indexStatusFailed}
