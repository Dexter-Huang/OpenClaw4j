// Package rag provides deterministic retrieval and prompt-context construction.
package rag

import (
	"context"
	"math"
	"sort"
	"strings"
	"unicode"
)

// Chunk is the storage-independent representation used by retrieval.
type Chunk struct {
	ID       string
	Document string
	Title    string
	Text     string
	Enabled  bool
}

// Source keeps persistence concerns outside the retriever. PostgreSQL, pgvector and
// test doubles can all implement the same narrow contract.
type Source interface {
	SearchChunks(context.Context, string, []string) ([]Chunk, error)
}

type Request struct {
	WorkspaceID      string
	KnowledgeBaseIDs []string
	Query            string
	Limit            int
	MaxContextRunes  int
}

type Result struct {
	Chunk   Chunk
	Score   float64
	Context string
}

type Retriever struct{ source Source }

func New(source Source) *Retriever { return &Retriever{source: source} }

// Retrieve uses BM25-style lexical ranking. It is deliberately deterministic so it
// remains a resilient fallback when an embedding provider or pgvector index is down.
// A vector-backed Source can pre-filter candidates without changing this API.
func (r *Retriever) Retrieve(ctx context.Context, request Request) ([]Result, error) {
	if r == nil || r.source == nil || strings.TrimSpace(request.Query) == "" {
		return []Result{}, nil
	}
	if request.Limit <= 0 {
		request.Limit = 5
	}
	if request.MaxContextRunes <= 0 {
		request.MaxContextRunes = 6000
	}
	candidates, err := r.source.SearchChunks(ctx, request.WorkspaceID, request.KnowledgeBaseIDs)
	if err != nil {
		return nil, err
	}
	queryTerms := termSet(terms(request.Query))
	if len(queryTerms) == 0 {
		return []Result{}, nil
	}
	documents := make([][]string, 0, len(candidates))
	averageLength := 0.0
	for _, candidate := range candidates {
		value := terms(candidate.Title + " " + candidate.Text)
		documents = append(documents, value)
		averageLength += float64(len(value))
	}
	if len(documents) == 0 {
		return []Result{}, nil
	}
	averageLength /= float64(len(documents))
	documentFrequency := make(map[string]int, len(queryTerms))
	for _, document := range documents {
		seen := map[string]struct{}{}
		for _, term := range document {
			if _, wanted := queryTerms[term]; wanted {
				seen[term] = struct{}{}
			}
		}
		for term := range seen {
			documentFrequency[term]++
		}
	}
	results := make([]Result, 0, len(candidates))
	for index, candidate := range candidates {
		if !candidate.Enabled {
			continue
		}
		frequency := map[string]int{}
		for _, term := range documents[index] {
			frequency[term]++
		}
		score := 0.0
		for term := range queryTerms {
			count := float64(frequency[term])
			if count == 0 {
				continue
			}
			idf := math.Log(1 + (float64(len(documents)-documentFrequency[term])+0.5)/(float64(documentFrequency[term])+0.5))
			const k1, b = 1.2, 0.75
			denominator := count + k1*(1-b+b*float64(len(documents[index]))/math.Max(averageLength, 1))
			score += idf * count * (k1 + 1) / denominator
		}
		if score > 0 {
			results = append(results, Result{Chunk: candidate, Score: score})
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].Chunk.ID < results[j].Chunk.ID
		}
		return results[i].Score > results[j].Score
	})
	if len(results) > request.Limit {
		results = results[:request.Limit]
	}
	remaining := request.MaxContextRunes
	for index := range results {
		text := strings.TrimSpace(results[index].Chunk.Text)
		if len([]rune(text)) > remaining {
			text = string([]rune(text)[:remaining])
		}
		results[index].Context = text
		remaining -= len([]rune(text))
		if remaining <= 0 {
			return results[:index+1], nil
		}
	}
	return results, nil
}

func terms(value string) []string {
	return strings.FieldsFunc(strings.ToLower(value), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) })
}

func termSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}
