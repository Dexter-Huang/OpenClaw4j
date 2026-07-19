package rag

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/seaskyland/openclaw4j-backend-go/internal/db"
)

var ErrInvalidIndexName = errors.New("RAG index name is invalid")

// VectorDocument holds an indexed chunk and its authorization metadata. Metadata
// is duplicated into columns so retrieval never needs an untrusted JSON expression.
type VectorDocument struct {
	ID, WorkspaceID, KnowledgeBaseID, DocumentID, Content string
	Enabled                                               bool
	Metadata                                              map[string]any
	Embedding                                             []float32
}

type VectorMatch struct {
	VectorDocument
	Score float64
}

type VectorStore interface {
	Upsert(context.Context, string, []VectorDocument) error
	Search(context.Context, string, string, []string, []float32, int) ([]VectorMatch, error)
	DeleteDocument(context.Context, string, string, string) error
}

type PGVectorStore struct{ database db.DBTX }

func NewPGVectorStore(database db.DBTX) *PGVectorStore { return &PGVectorStore{database: database} }

func (s *PGVectorStore) Upsert(ctx context.Context, indexName string, documents []VectorDocument) error {
	if len(documents) == 0 {
		return nil
	}
	if s == nil || s.database == nil {
		return errors.New("pgvector database is unavailable")
	}
	table, err := vectorTable(indexName)
	if err != nil {
		return err
	}
	dimension := len(documents[0].Embedding)
	if dimension == 0 {
		return errors.New("embedding vector is empty")
	}
	for _, document := range documents {
		if len(document.Embedding) != dimension {
			return errors.New("embedding dimensions are inconsistent")
		}
	}
	if _, err := s.database.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS vector"); err != nil {
		return fmt.Errorf("enable pgvector extension: %w", err)
	}
	create := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
id text PRIMARY KEY, workspace_id varchar(64) NOT NULL, kb_id varchar(64) NOT NULL,
doc_id varchar(64) NOT NULL, content text NOT NULL, metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
enabled boolean NOT NULL DEFAULT true, embedding vector(%d) NOT NULL, gmt_modified timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP)`, table, dimension)
	if _, err := s.database.Exec(ctx, create); err != nil {
		return fmt.Errorf("create pgvector table: %w", err)
	}
	index := "hnsw_" + strings.TrimPrefix(table, "kb_")
	if _, err := s.database.Exec(ctx, fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s USING hnsw (embedding vector_cosine_ops)", index, table)); err != nil {
		return fmt.Errorf("create pgvector index: %w", err)
	}
	statement := fmt.Sprintf(`INSERT INTO %s (id,workspace_id,kb_id,doc_id,content,metadata,enabled,embedding,gmt_modified)
VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7,$8::vector,CURRENT_TIMESTAMP)
ON CONFLICT (id) DO UPDATE SET content=EXCLUDED.content,metadata=EXCLUDED.metadata,enabled=EXCLUDED.enabled,embedding=EXCLUDED.embedding,gmt_modified=CURRENT_TIMESTAMP`, table)
	for _, document := range documents {
		metadata, err := json.Marshal(document.Metadata)
		if err != nil {
			return err
		}
		if _, err := s.database.Exec(ctx, statement, document.ID, document.WorkspaceID, document.KnowledgeBaseID, document.DocumentID, document.Content, string(metadata), document.Enabled, vectorLiteral(document.Embedding)); err != nil {
			return fmt.Errorf("upsert vector %s: %w", document.ID, err)
		}
	}
	return nil
}

func (s *PGVectorStore) Search(ctx context.Context, indexName, workspaceID string, knowledgeBaseIDs []string, embedding []float32, limit int) ([]VectorMatch, error) {
	if s == nil || s.database == nil {
		return nil, errors.New("pgvector database is unavailable")
	}
	if len(embedding) == 0 {
		return []VectorMatch{}, nil
	}
	if limit <= 0 {
		limit = 5
	}
	table, err := vectorTable(indexName)
	if err != nil {
		return nil, err
	}
	arguments := []any{workspaceID, vectorLiteral(embedding), limit}
	filter := "workspace_id=$1 AND enabled=true"
	if len(knowledgeBaseIDs) > 0 {
		values := make([]string, 0, len(knowledgeBaseIDs))
		for _, id := range knowledgeBaseIDs {
			if id = strings.TrimSpace(id); id != "" {
				values = append(values, id)
			}
		}
		if len(values) > 0 {
			arguments = append(arguments, values)
			filter += fmt.Sprintf(" AND kb_id = ANY($%d)", len(arguments))
		}
	}
	query := fmt.Sprintf("SELECT id,workspace_id,kb_id,doc_id,content,metadata,enabled,1-(embedding <=> $2::vector) AS score FROM %s WHERE %s ORDER BY embedding <=> $2::vector LIMIT $3", table, filter)
	rows, err := s.database.Query(ctx, query, arguments...)
	if err != nil {
		return nil, fmt.Errorf("search pgvector: %w", err)
	}
	defer rows.Close()
	result := []VectorMatch{}
	for rows.Next() {
		var item VectorMatch
		var raw string
		if err := rows.Scan(&item.ID, &item.WorkspaceID, &item.KnowledgeBaseID, &item.DocumentID, &item.Content, &raw, &item.Enabled, &item.Score); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(raw), &item.Metadata); err != nil {
			return nil, fmt.Errorf("decode vector metadata: %w", err)
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *PGVectorStore) DeleteDocument(ctx context.Context, indexName, workspaceID, documentID string) error {
	if s == nil || s.database == nil {
		return errors.New("pgvector database is unavailable")
	}
	table, err := vectorTable(indexName)
	if err != nil {
		return err
	}
	_, err = s.database.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE workspace_id=$1 AND doc_id=$2", table), workspaceID, documentID)
	return err
}

var indexNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,52}$`)

func vectorTable(indexName string) (string, error) {
	name := strings.ToLower(strings.TrimSpace(indexName))
	if !indexNamePattern.MatchString(name) {
		return "", ErrInvalidIndexName
	}
	return "kb_" + name, nil
}
func vectorLiteral(values []float32) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = strconv.FormatFloat(float64(value), 'g', -1, 32)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

var _ VectorStore = (*PGVectorStore)(nil)
