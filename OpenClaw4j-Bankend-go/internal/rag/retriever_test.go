package rag

import (
	"context"
	"testing"
)

type memorySource []Chunk

func (m memorySource) SearchChunks(context.Context, string, []string) ([]Chunk, error) { return m, nil }

func TestRetrieveRanksEnabledChunksAndCapsContext(t *testing.T) {
	retriever := New(memorySource{
		{ID: "weaker", Text: "Redis stores tokens", Enabled: true},
		{ID: "best", Text: "Redis workflow state persists task status and node output", Enabled: true},
		{ID: "disabled", Text: "Redis workflow state", Enabled: false},
	})
	results, err := retriever.Retrieve(context.Background(), Request{Query: "Redis workflow state", Limit: 2, MaxContextRunes: 12})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Chunk.ID != "best" {
		t.Fatalf("unexpected results: %#v", results)
	}
	if len([]rune(results[0].Context)) != 12 {
		t.Fatalf("context cap was not applied: %q", results[0].Context)
	}
}
