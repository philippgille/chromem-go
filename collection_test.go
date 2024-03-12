package chromem

import (
	"context"
	"errors"
	"math/rand"
	"slices"
	"strconv"
	"testing"
)

func TestCollection_Add(t *testing.T) {
	ctx := context.Background()
	name := "test"
	metadata := map[string]string{"foo": "bar"}
	vectors := []float32{-0.1, 0.1, 0.2}
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return vectors, nil
	}

	// Create collection
	db := NewDB()
	c, err := db.CreateCollection(name, metadata, embeddingFunc)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	if c == nil {
		t.Fatal("expected collection, got nil")
	}

	// Add documents

	ids := []string{"1", "2"}
	embeddings := [][]float32{vectors, vectors}
	metadatas := []map[string]string{{"foo": "bar"}, {"a": "b"}}
	contents := []string{"hello world", "hallo welt"}

	tt := []struct {
		name       string
		ids        []string
		embeddings [][]float32
		metadatas  []map[string]string
		contents   []string
	}{
		{
			name:       "No embeddings",
			ids:        ids,
			embeddings: nil,
			metadatas:  metadatas,
			contents:   contents,
		},
		{
			name:       "With embeddings",
			ids:        ids,
			embeddings: embeddings,
			metadatas:  metadatas,
			contents:   contents,
		},
		{
			name:       "With embeddings but no contents",
			ids:        ids,
			embeddings: embeddings,
			metadatas:  metadatas,
			contents:   nil,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			err = c.Add(ctx, ids, nil, metadatas, contents)
			if err != nil {
				t.Fatal("expected nil, got", err)
			}

			// Check documents
			if len(c.documents) != 2 {
				t.Fatal("expected 2, got", len(c.documents))
			}
			for i, id := range ids {
				doc, ok := c.documents[id]
				if !ok {
					t.Fatal("expected document, got nil")
				}
				if doc.ID != id {
					t.Fatal("expected", id, "got", doc.ID)
				}
				if len(doc.Metadata) != 1 {
					t.Fatal("expected 1, got", len(doc.Metadata))
				}
				if !slices.Equal(doc.Embedding, vectors) {
					t.Fatal("expected", vectors, "got", doc.Embedding)
				}
				if doc.Content != contents[i] {
					t.Fatal("expected", contents[i], "got", doc.Content)
				}
			}
			// Metadata can't be accessed with the loop's i
			if c.documents[ids[0]].Metadata["foo"] != "bar" {
				t.Fatal("expected bar, got", c.documents[ids[0]].Metadata["foo"])
			}
			if c.documents[ids[1]].Metadata["a"] != "b" {
				t.Fatal("expected b, got", c.documents[ids[1]].Metadata["a"])
			}
		})
	}
}

func TestCollection_Add_Error(t *testing.T) {
	ctx := context.Background()
	name := "test"
	metadata := map[string]string{"foo": "bar"}
	vectors := []float32{-0.1, 0.1, 0.2}
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return vectors, nil
	}

	// Create collection
	db := NewDB()
	c, err := db.CreateCollection(name, metadata, embeddingFunc)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	if c == nil {
		t.Fatal("expected collection, got nil")
	}

	// Add documents, provoking errors
	ids := []string{"1", "2"}
	embeddings := [][]float32{vectors, vectors}
	metadatas := []map[string]string{{"foo": "bar"}, {"a": "b"}}
	contents := []string{"hello world", "hallo welt"}
	// Empty IDs
	err = c.Add(ctx, []string{}, embeddings, metadatas, contents)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Empty embeddings and contents (both at the same time!)
	err = c.Add(ctx, ids, [][]float32{}, metadatas, []string{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Bad embeddings length
	err = c.Add(ctx, ids, [][]float32{vectors}, metadatas, contents)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Bad metadatas length
	err = c.Add(ctx, ids, embeddings, []map[string]string{{"foo": "bar"}}, contents)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Bad contents length
	err = c.Add(ctx, ids, embeddings, metadatas, []string{"hello world"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCollection_AddConcurrently(t *testing.T) {
	ctx := context.Background()
	name := "test"
	metadata := map[string]string{"foo": "bar"}
	vectors := []float32{-0.1, 0.1, 0.2}
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return vectors, nil
	}

	// Create collection
	db := NewDB()
	c, err := db.CreateCollection(name, metadata, embeddingFunc)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	if c == nil {
		t.Fatal("expected collection, got nil")
	}

	// Add documents

	ids := []string{"1", "2"}
	embeddings := [][]float32{vectors, vectors}
	metadatas := []map[string]string{{"foo": "bar"}, {"a": "b"}}
	contents := []string{"hello world", "hallo welt"}

	tt := []struct {
		name       string
		ids        []string
		embeddings [][]float32
		metadatas  []map[string]string
		contents   []string
	}{
		{
			name:       "No embeddings",
			ids:        ids,
			embeddings: nil,
			metadatas:  metadatas,
			contents:   contents,
		},
		{
			name:       "With embeddings",
			ids:        ids,
			embeddings: embeddings,
			metadatas:  metadatas,
			contents:   contents,
		},
		{
			name:       "With embeddings but no contents",
			ids:        ids,
			embeddings: embeddings,
			metadatas:  metadatas,
			contents:   nil,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			err = c.AddConcurrently(ctx, ids, nil, metadatas, contents, 2)
			if err != nil {
				t.Fatal("expected nil, got", err)
			}

			// Check documents
			if len(c.documents) != 2 {
				t.Fatal("expected 2, got", len(c.documents))
			}
			for i, id := range ids {
				doc, ok := c.documents[id]
				if !ok {
					t.Fatal("expected document, got nil")
				}
				if doc.ID != id {
					t.Fatal("expected", id, "got", doc.ID)
				}
				if len(doc.Metadata) != 1 {
					t.Fatal("expected 1, got", len(doc.Metadata))
				}
				if !slices.Equal(doc.Embedding, vectors) {
					t.Fatal("expected", vectors, "got", doc.Embedding)
				}
				if doc.Content != contents[i] {
					t.Fatal("expected", contents[i], "got", doc.Content)
				}
			}
			// Metadata can't be accessed with the loop's i
			if c.documents[ids[0]].Metadata["foo"] != "bar" {
				t.Fatal("expected bar, got", c.documents[ids[0]].Metadata["foo"])
			}
			if c.documents[ids[1]].Metadata["a"] != "b" {
				t.Fatal("expected b, got", c.documents[ids[1]].Metadata["a"])
			}
		})
	}
}

func TestCollection_AddConcurrently_Error(t *testing.T) {
	ctx := context.Background()
	name := "test"
	metadata := map[string]string{"foo": "bar"}
	vectors := []float32{-0.1, 0.1, 0.2}
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return vectors, nil
	}

	// Create collection
	db := NewDB()
	c, err := db.CreateCollection(name, metadata, embeddingFunc)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	if c == nil {
		t.Fatal("expected collection, got nil")
	}

	// Add documents, provoking errors
	ids := []string{"1", "2"}
	embeddings := [][]float32{vectors, vectors}
	metadatas := []map[string]string{{"foo": "bar"}, {"a": "b"}}
	contents := []string{"hello world", "hallo welt"}
	// Empty IDs
	err = c.AddConcurrently(ctx, []string{}, embeddings, metadatas, contents, 2)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Empty embeddings and contents (both at the same time!)
	err = c.AddConcurrently(ctx, ids, [][]float32{}, metadatas, []string{}, 2)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Bad embeddings length
	err = c.AddConcurrently(ctx, ids, [][]float32{vectors}, metadatas, contents, 2)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Bad metadatas length
	err = c.AddConcurrently(ctx, ids, embeddings, []map[string]string{{"foo": "bar"}}, contents, 2)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Bad contents length
	err = c.AddConcurrently(ctx, ids, embeddings, metadatas, []string{"hello world"}, 2)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Bad concurrency
	err = c.AddConcurrently(ctx, ids, embeddings, metadatas, contents, 0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCollection_Count(t *testing.T) {
	// Create collection
	db := NewDB()
	name := "test"
	metadata := map[string]string{"foo": "bar"}
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return []float32{-0.1, 0.1, 0.2}, nil
	}
	c, err := db.CreateCollection(name, metadata, embeddingFunc)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	if c == nil {
		t.Fatal("expected collection, got nil")
	}

	// Add documents
	ids := []string{"1", "2"}
	metadatas := []map[string]string{{"foo": "bar"}, {"a": "b"}}
	contents := []string{"hello world", "hallo welt"}
	err = c.Add(context.Background(), ids, nil, metadatas, contents)
	if err != nil {
		t.Fatal("expected nil, got", err)
	}

	// Check count
	if c.Count() != 2 {
		t.Fatal("expected 2, got", c.Count())
	}
}

// Global var for assignment in the benchmark to avoid compiler optimizations.
var globalRes []Result

func BenchmarkCollection_Query_100(b *testing.B) {
	benchmarkCollection_Query(b, 100)
}

func BenchmarkCollection_Query_1000(b *testing.B) {
	benchmarkCollection_Query(b, 1000)
}

func BenchmarkCollection_Query_5000(b *testing.B) {
	benchmarkCollection_Query(b, 5000)
}

func BenchmarkCollection_Query_25000(b *testing.B) {
	benchmarkCollection_Query(b, 25000)
}

func BenchmarkCollection_Query_100000(b *testing.B) {
	benchmarkCollection_Query(b, 100_000)
}

// n is number of documents in the collection
func benchmarkCollection_Query(b *testing.B, n int) {
	ctx := context.Background()

	// Seed to make deterministic
	r := rand.New(rand.NewSource(42))

	d := 1536 // dimensions, same as text-embedding-3-small
	// Random query vector
	qv := make([]float32, d)
	for j := 0; j < d; j++ {
		qv[j] = r.Float32()
	}
	// Most embeddings are normalized, so we normalize this one too
	qv = normalizeVector(qv)
	embeddingFunc := func(_ context.Context, text string) ([]float32, error) {
		if text != "foo" {
			return nil, errors.New("embedding func not expected to be called")
		}
		return qv, nil
	}

	// Create collection
	db := NewDB()
	name := "test"
	c, err := db.CreateCollection(name, nil, embeddingFunc)
	if err != nil {
		b.Fatal("expected no error, got", err)
	}
	if c == nil {
		b.Fatal("expected collection, got nil")
	}

	// Add documents
	for i := 0; i < n; i++ {
		// Random embedding
		v := make([]float32, d)
		for j := 0; j < d; j++ {
			v[j] = r.Float32()
		}
		v = normalizeVector(v)

		// Add document without metadata or content.
		// When providing embeddings, the embedding func is not called.
		c.AddDocument(ctx, Document{
			ID:        strconv.Itoa(i),
			Embedding: v,
		})
	}

	b.ResetTimer()

	// Query
	var res []Result
	for i := 0; i < b.N; i++ {
		res, err = c.Query(ctx, "foo", 10, nil, nil)
	}
	if err != nil {
		b.Fatal("expected nil, got", err)
	}
	globalRes = res
}
