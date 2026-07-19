package rag

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestPGVectorStoreIntegration runs against an explicitly supplied pgvector
// database, so ordinary unit-test runs remain self-contained on developer hosts.
func TestPGVectorStoreIntegration(t *testing.T) {
	dsn := os.Getenv("OPENCLAW_TEST_PGVECTOR_DSN")
	if dsn == "" {
		t.Skip("OPENCLAW_TEST_PGVECTOR_DSN is not configured")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	defer func() { _, _ = pool.Exec(context.Background(), "DROP TABLE IF EXISTS kb_integration_rag") }()
	store := NewPGVectorStore(pool)
	if err := store.Upsert(context.Background(), "integration_rag", []VectorDocument{
		{ID: "a", WorkspaceID: "ws", KnowledgeBaseID: "kb-a", DocumentID: "doc-a", Content: "first", Enabled: true, Embedding: []float32{1, 0, 0}},
		{ID: "b", WorkspaceID: "ws", KnowledgeBaseID: "kb-b", DocumentID: "doc-b", Content: "second", Enabled: true, Embedding: []float32{0, 1, 0}},
	}); err != nil {
		t.Fatal(err)
	}
	matches, err := store.Search(context.Background(), "integration_rag", "ws", []string{"kb-a"}, []float32{1, 0, 0}, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0].ID != "a" {
		t.Fatalf("unexpected matches: %#v", matches)
	}
}
