package chromem

import (
	"context"
	"errors"
	"math/rand"
	"os"
	"slices"
	"strconv"
	"testing"
)

func TestCollection_Add(t *testing.T) {
	ctx := context.Background()
	name := "test"
	metadata := map[string]string{"foo": "bar"}
	vectors := []float32{-0.40824828, 0.40824828, 0.81649655} // normalized version of `{-0.1, 0.1, 0.2}`
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
	vectors := []float32{-0.40824828, 0.40824828, 0.81649655} // normalized version of `{-0.1, 0.1, 0.2}`
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
	vectors := []float32{-0.40824828, 0.40824828, 0.81649655} // normalized version of `{-0.1, 0.1, 0.2}`
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
	vectors := []float32{-0.40824828, 0.40824828, 0.81649655} // normalized version of `{-0.1, 0.1, 0.2}`
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

func TestCollection_QueryError(t *testing.T) {
	// Create collection
	db := NewDB()
	name := "test"
	metadata := map[string]string{"foo": "bar"}
	vectors := []float32{-0.40824828, 0.40824828, 0.81649655} // normalized version of `{-0.1, 0.1, 0.2}`
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return vectors, nil
	}
	c, err := db.CreateCollection(name, metadata, embeddingFunc)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	if c == nil {
		t.Fatal("expected collection, got nil")
	}
	// Add a document
	err = c.AddDocument(context.Background(), Document{ID: "1", Content: "hello world"})
	if err != nil {
		t.Fatal("expected nil, got", err)
	}

	tt := []struct {
		name   string
		query  func() error
		expErr string
	}{
		{
			name: "Empty query",
			query: func() error {
				_, err := c.Query(context.Background(), "", 1, nil, nil)
				return err
			},
			expErr: "queryText is empty",
		},
		{
			name: "Negative limit",
			query: func() error {
				_, err := c.Query(context.Background(), "foo", -1, nil, nil)
				return err
			},
			expErr: "nResults must be > 0",
		},
		{
			name: "Zero limit",
			query: func() error {
				_, err := c.Query(context.Background(), "foo", 0, nil, nil)
				return err
			},
			expErr: "nResults must be > 0",
		},
		{
			name: "Limit greater than number of documents",
			query: func() error {
				_, err := c.Query(context.Background(), "foo", 2, nil, nil)
				return err
			},
			expErr: "nResults must be <= the number of documents in the collection",
		},
		{
			name: "Bad content filter",
			query: func() error {
				_, err := c.Query(context.Background(), "foo", 1, nil, map[string]string{"invalid": "foo"})
				return err
			},
			expErr: "unsupported operator",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.query()
			if err == nil {
				t.Fatal("expected error, got nil")
			} else if err.Error() != tc.expErr {
				t.Fatal("expected", tc.expErr, "got", err)
			}
		})
	}
}

func TestCollection_ListIDs(t *testing.T) {
	ctx := context.Background()

	// Create collection
	db := NewDB()
	name := "test"
	metadata := map[string]string{"foo": "bar"}
	vectors := []float32{-0.40824828, 0.40824828, 0.81649655} // normalized version of `{-0.1, 0.1, 0.2}`
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return vectors, nil
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

	// List IDs
	foundIds := c.ListIDs(ctx)

	// Ensure IDs match
	// (slices are same length and all the items in the first slice exist in the second slice)
	if len(foundIds) != len(ids) {
		t.Fatal("expected", len(ids), "got", len(foundIds))
	}
	for _, id := range ids {
		found := false
		for _, foundID := range foundIds {
			if id == foundID {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("expected", id, "in", foundIds)
		}
	}
}

// TestCollection_ListDocuments verifies that ListDocuments returns all documents
// and that the returned documents are deep-copies (mutating them must not affect
// the collection’s internal state).
func TestCollection_ListDocuments(t *testing.T) {
	ctx := context.Background()

	// Fixed embedding so we can compare easily.
	embedVec := []float32{0.0, 1.0, 0.0}
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return embedVec, nil
	}

	// Create collection.
	db := NewDB()
	coll, err := db.CreateCollection("test", nil, embeddingFunc)
	if err != nil {
		t.Fatalf("unexpected error creating collection: %v", err)
	}

	// Add two documents (one with explicit embedding, one relying on embeddingFunc).
	docs := []Document{
		{ID: "1", Metadata: map[string]string{"foo": "bar"}, Embedding: embedVec, Content: "hello"},
		{ID: "2", Metadata: map[string]string{"baz": "qux"}, Content: "world"},
	}
	for _, d := range docs {
		if err := coll.AddDocument(ctx, d); err != nil {
			t.Fatalf("unexpected error adding document %q: %v", d.ID, err)
		}
	}

	got, err := coll.ListDocuments(ctx)
	if err != nil {
		t.Fatalf("unexpected error from ListDocuments: %v", err)
	}
	if len(got) != len(docs) {
		t.Fatalf("expected %d docs, got %d", len(docs), len(got))
	}

	// Map for convenient lookup.
	deepByID := make(map[string]Document, len(got))
	for _, d := range got {
		deepByID[d.ID] = d
	}

	for _, want := range docs {
		got, ok := deepByID[want.ID]
		if !ok {
			t.Fatalf("doc %q not found", want.ID)
		}
		if got.Content != want.Content {
			t.Fatalf("doc %q: expected content %q, got %q", want.ID, want.Content, got.Content)
		}
		if !slices.Equal(got.Embedding, embedVec) {
			t.Fatalf("doc %q: embeddings differ, expected %v got %v", want.ID, embedVec, got.Embedding)
		}
		for k, v := range want.Metadata {
			if got.Metadata[k] != v {
				t.Fatalf("doc %q: expected metadata %q=%q, got %q", want.ID, k, v, got.Metadata[k])
			}
		}
	}

	// Mutate deep copy and ensure collection is untouched.
	got[0].Metadata["foo"] = "mutated"
	orig, _ := coll.GetByID(ctx, "1")
	if orig.Metadata["foo"] != "bar" {
		t.Fatalf("mutation leaked into collection: expected \"bar\", got %q", orig.Metadata["foo"])
	}
}

// TestCollection_ListDocumentsShort verifies that ListDocumentsShort returns all documents
// and that the returned documents are deep-copies (mutating them must not affect
// the collection’s internal state).
func TestCollection_ListDocumentsShort(t *testing.T) {
	ctx := context.Background()

	// Fixed embedding so we can compare easily.
	embedVec := []float32{0.0, 1.0, 0.0}
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return embedVec, nil
	}

	// Create collection.
	db := NewDB()
	coll, err := db.CreateCollection("test", nil, embeddingFunc)
	if err != nil {
		t.Fatalf("unexpected error creating collection: %v", err)
	}

	// Add two documents (one with explicit embedding, one relying on embeddingFunc).
	docs := []Document{
		{ID: "1", Metadata: map[string]string{"foo": "bar"}, Embedding: embedVec, Content: "hello"},
		{ID: "2", Metadata: map[string]string{"baz": "qux"}, Content: "world"},
	}
	for _, d := range docs {
		if err := coll.AddDocument(ctx, d); err != nil {
			t.Fatalf("unexpected error adding document %q: %v", d.ID, err)
		}
	}

	got, err := coll.ListDocumentsShort(ctx)
	if err != nil {
		t.Fatalf("unexpected error from ListDocumentsShort: %v", err)
	}
	if len(got) != len(docs) {
		t.Fatalf("expected %d docs, got %d", len(docs), len(got))
	}
	for _, d := range got {
		if d.Metadata != nil {
			t.Fatalf("expected Metadata to be nil, got %#v", d.Metadata)
		}
		if d.Embedding != nil {
			t.Fatalf("expected Embedding to be nil, got %#v", d.Embedding)
		}
		// Content and ID must still be present.
		if d.Content == "" || d.ID == "" {
			t.Fatalf("expected ID and Content to be set, got %+v", d)
		}
	}

	// Mutate deep copy and ensure collection is untouched.
	got[0].Content = "mutated"
	orig, _ := coll.GetByID(ctx, "1")
	if orig.Content != "hello" {
		t.Fatalf("mutation leaked into collection: expected \"hello\", got %q", orig.Content)
	}
}

func TestCollection_GetByID(t *testing.T) {
	ctx := context.Background()

	// Create collection
	db := NewDB()
	name := "test"
	metadata := map[string]string{"foo": "bar"}
	vectors := []float32{-0.40824828, 0.40824828, 0.81649655} // normalized version of `{-0.1, 0.1, 0.2}`
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return vectors, nil
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

	// Get by ID
	doc, err := c.GetByID(ctx, ids[0])
	if err != nil {
		t.Fatal("expected nil, got", err)
	}
	// Check fields
	if doc.ID != ids[0] {
		t.Fatal("expected", ids[0], "got", doc.ID)
	}
	if len(doc.Metadata) != 1 {
		t.Fatal("expected 1, got", len(doc.Metadata))
	}
	if !slices.Equal(doc.Embedding, vectors) {
		t.Fatal("expected", vectors, "got", doc.Embedding)
	}
	if doc.Content != contents[0] {
		t.Fatal("expected", contents[0], "got", doc.Content)
	}

	// Check error
	_, err = c.GetByID(ctx, "3")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCollection_GetByMetadata(t *testing.T) {
	ctx := context.Background()

	// Create collection
	db := NewDB()
	name := "test"
	metadata := map[string]string{"foo": "bar"}
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return []float32{1.0, 2.0, 3.0}, nil
	}
	c, err := db.CreateCollection(name, metadata, embeddingFunc)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}

	// Add documents
	docs := []Document{
		{ID: "1", Metadata: map[string]string{"type": "article", "lang": "en"}, Content: "Hello World"},
		{ID: "2", Metadata: map[string]string{"type": "article", "lang": "fr"}, Content: "Bonjour le monde"},
		{ID: "3", Metadata: map[string]string{"type": "blog", "lang": "en"}, Content: "My blog post"},
	}
	for _, doc := range docs {
		err := c.AddDocument(ctx, doc)
		if err != nil {
			t.Fatal("expected no error, got", err)
		}
	}

	// Filter by metadata
	where := map[string]string{"type": "article", "lang": "en"}
	results, err := c.GetByMetadata(ctx, where)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ID != "1" {
		t.Fatalf("expected document ID '1', got '%s'", results[0].ID)
	}
}

func TestCollection_Count(t *testing.T) {
	// Create collection
	db := NewDB()
	name := "test"
	metadata := map[string]string{"foo": "bar"}
	vectors := []float32{-0.40824828, 0.40824828, 0.81649655} // normalized version of `{-0.1, 0.1, 0.2}`
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return vectors, nil
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

func TestCollection_Delete(t *testing.T) {
	// Create persistent collection
	tmpdir, err := os.MkdirTemp(os.TempDir(), "chromem-test-*")
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	db, err := NewPersistentDB(tmpdir, false)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	name := "test"
	metadata := map[string]string{"foo": "bar"}
	vectors := []float32{-0.40824828, 0.40824828, 0.81649655} // normalized version of `{-0.1, 0.1, 0.2}`
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return vectors, nil
	}
	c, err := db.CreateCollection(name, metadata, embeddingFunc)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	if c == nil {
		t.Fatal("expected collection, got nil")
	}

	// Add documents
	ids := []string{"1", "2", "3", "4"}
	metadatas := []map[string]string{{"foo": "bar"}, {"a": "b"}, {"foo": "bar"}, {"e": "f"}}
	contents := []string{"hello world", "hallo welt", "bonjour le monde", "hola mundo"}
	err = c.Add(context.Background(), ids, nil, metadatas, contents)
	if err != nil {
		t.Fatal("expected nil, got", err)
	}

	// Check count
	if c.Count() != 4 {
		t.Fatal("expected 4 documents, got", c.Count())
	}

	// Check number of files in the persist directory
	d, err := os.ReadDir(c.persistDirectory)
	if err != nil {
		t.Fatal("expected nil, got", err)
	}
	if len(d) != 5 { // 4 documents + 1 metadata file
		t.Fatal("expected 4 document files + 1 metadata file in persist_dir, got", len(d))
	}

	checkCount := func(expected int) {
		// Check count
		if c.Count() != expected {
			t.Fatalf("expected %d documents, got %d", expected, c.Count())
		}

		// Check number of files in the persist directory
		d, err = os.ReadDir(c.persistDirectory)
		if err != nil {
			t.Fatal("expected nil, got", err)
		}
		if len(d) != expected+1 { // 3 document + 1 metadata file
			t.Fatalf("expected %d document files + 1 metadata file in persist_dir, got %d", expected, len(d))
		}
	}

	// Test 1 - Remove document by ID: should delete one document
	err = c.Delete(context.Background(), nil, nil, "4")
	if err != nil {
		t.Fatal("expected nil, got", err)
	}
	checkCount(3)

	// Test 2 - Remove document by metadata
	err = c.Delete(context.Background(), map[string]string{"foo": "bar"}, nil)
	if err != nil {
		t.Fatal("expected nil, got", err)
	}

	checkCount(1)

	// Test 3 - Remove document by content
	err = c.Delete(context.Background(), nil, map[string]string{"$contains": "hallo welt"})
	if err != nil {
		t.Fatal("expected nil, got", err)
	}

	checkCount(0)
}

// TestCloneDocument verifies that cloneDocument creates a deep copy of the Document.
func TestCloneDocument(t *testing.T) {
	orig := &Document{
		ID:        "doc1",
		Metadata:  map[string]string{"foo": "bar"},
		Embedding: []float32{1.0, 2.0, 3.0},
		Content:   "hello",
	}
	clone := cloneDocument(orig)

	// Check values are equal
	if clone.ID != orig.ID {
		t.Fatalf("expected ID %q, got %q", orig.ID, clone.ID)
	}
	if clone.Content != orig.Content {
		t.Fatalf("expected Content %q, got %q", orig.Content, clone.Content)
	}
	if !slices.Equal(clone.Embedding, orig.Embedding) {
		t.Fatalf("expected Embedding %v, got %v", orig.Embedding, clone.Embedding)
	}
	for k, v := range orig.Metadata {
		if clone.Metadata[k] != v {
			t.Fatalf("expected Metadata[%q]=%q, got %q", k, v, clone.Metadata[k])
		}
	}

	// Mutate clone and ensure original is not affected
	clone.Metadata["foo"] = "baz"
	clone.Embedding[0] = 42.0
	if orig.Metadata["foo"] != "bar" {
		t.Fatalf("mutation leaked into original Metadata: expected \"bar\", got %q", orig.Metadata["foo"])
	}
	if orig.Embedding[0] != 1.0 {
		t.Fatalf("mutation leaked into original Embedding: expected 1.0, got %v", orig.Embedding[0])
	}
}

// TestCloneDocumentShort verifies that cloneDocumentShort creates a shallow copy with nil Metadata and Embedding.
func TestCloneDocumentShort(t *testing.T) {
	orig := &Document{
		ID:        "doc2",
		Metadata:  map[string]string{"foo": "bar"},
		Embedding: []float32{1.0, 2.0, 3.0},
		Content:   "world",
	}
	clone := cloneDocumentShort(orig)

	// Check ID and Content are copied
	if clone.ID != orig.ID {
		t.Fatalf("expected ID %q, got %q", orig.ID, clone.ID)
	}
	if clone.Content != orig.Content {
		t.Fatalf("expected Content %q, got %q", orig.Content, clone.Content)
	}
	// Metadata and Embedding must be nil
	if clone.Metadata != nil {
		t.Fatalf("expected Metadata to be nil, got %#v", clone.Metadata)
	}
	if clone.Embedding != nil {
		t.Fatalf("expected Embedding to be nil, got %#v", clone.Embedding)
	}
}

// Global var for assignment in the benchmark to avoid compiler optimizations.
var globalRes []Result

func BenchmarkCollection_Query_NoContent_100(b *testing.B) {
	benchmarkCollection_Query(b, 100, false)
}

func BenchmarkCollection_Query_NoContent_1000(b *testing.B) {
	benchmarkCollection_Query(b, 1000, false)
}

func BenchmarkCollection_Query_NoContent_5000(b *testing.B) {
	benchmarkCollection_Query(b, 5000, false)
}

func BenchmarkCollection_Query_NoContent_25000(b *testing.B) {
	benchmarkCollection_Query(b, 25000, false)
}

func BenchmarkCollection_Query_NoContent_100000(b *testing.B) {
	benchmarkCollection_Query(b, 100_000, false)
}

func BenchmarkCollection_Query_100(b *testing.B) {
	benchmarkCollection_Query(b, 100, true)
}

func BenchmarkCollection_Query_1000(b *testing.B) {
	benchmarkCollection_Query(b, 1000, true)
}

func BenchmarkCollection_Query_5000(b *testing.B) {
	benchmarkCollection_Query(b, 5000, true)
}

func BenchmarkCollection_Query_25000(b *testing.B) {
	benchmarkCollection_Query(b, 25000, true)
}

func BenchmarkCollection_Query_100000(b *testing.B) {
	benchmarkCollection_Query(b, 100_000, true)
}

// n is number of documents in the collection
func benchmarkCollection_Query(b *testing.B, n int, withContent bool) {
	ctx := context.Background()

	// Seed to make deterministic
	r := rand.New(rand.NewSource(42))

	d := 1536 // dimensions, same as text-embedding-3-small
	// Random query vector
	qv := make([]float32, d)
	for j := 0; j < d; j++ {
		qv[j] = r.Float32()
	}
	// The document embeddings are normalized, so the query must be normalized too.
	qv = normalizeVector(qv)

	// Create collection
	db := NewDB()
	name := "test"
	embeddingFunc := func(_ context.Context, text string) ([]float32, error) {
		return nil, errors.New("embedding func not expected to be called")
	}
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

		// Add document with some metadata and content depending on parameter.
		// When providing embeddings, the embedding func is not called.
		is := strconv.Itoa(i)
		doc := Document{
			ID:        is,
			Metadata:  map[string]string{"i": is, "foo": "bar" + is},
			Embedding: v,
		}
		if withContent {
			// Let's say we embed 500 tokens, that's ~375 words, ~1875 characters
			doc.Content = randomString(r, 1875)
		}

		if err := c.AddDocument(ctx, doc); err != nil {
			b.Fatal("expected nil, got", err)
		}
	}

	b.ResetTimer()

	// Query
	var res []Result
	for i := 0; i < b.N; i++ {
		res, err = c.QueryEmbedding(ctx, qv, 10, nil, nil)
	}
	if err != nil {
		b.Fatal("expected nil, got", err)
	}
	globalRes = res
}

// BenchmarkCloneDocument_1 performs 1 clone per iteration.
func BenchmarkCloneDocument_1(b *testing.B) {
	benchmarkCloneDocumentN(b, 1)
}

// BenchmarkCloneDocument_10 performs 10 clones per iteration.
func BenchmarkCloneDocument_10(b *testing.B) {
	benchmarkCloneDocumentN(b, 10)
}

// BenchmarkCloneDocument_100 performs 100 clones per iteration.
func BenchmarkCloneDocument_100(b *testing.B) {
	benchmarkCloneDocumentN(b, 100)
}

// BenchmarkCloneDocument_1000 performs 1000 clones per iteration.
func BenchmarkCloneDocument_1000(b *testing.B) {
	benchmarkCloneDocumentN(b, 1000)
}

// BenchmarkCloneDocumentShort_1 performs 1 shallow clone per iteration.
func BenchmarkCloneDocumentShort_1(b *testing.B) {
	benchmarkCloneDocumentShortN(b, 1)
}

// BenchmarkCloneDocumentShort_10 performs 10 shallow clones per iteration.
func BenchmarkCloneDocumentShort_10(b *testing.B) {
	benchmarkCloneDocumentShortN(b, 10)
}

// BenchmarkCloneDocumentShort_100 performs 100 shallow clones per iteration.
func BenchmarkCloneDocumentShort_100(b *testing.B) {
	benchmarkCloneDocumentShortN(b, 100)
}

// BenchmarkCloneDocumentShort_1000 performs 1000 shallow clones per iteration.
func BenchmarkCloneDocumentShort_1000(b *testing.B) {
	benchmarkCloneDocumentShortN(b, 1000)
}

// Helper for benchmarking cloneDocument with n clones per iteration.
func benchmarkCloneDocumentN(b *testing.B, n int) {
	doc := &Document{
		ID:        "bench",
		Metadata:  map[string]string{"foo": "bar", "baz": "qux"},
		Embedding: []float32{1.0, 2.0, 3.0},
		Content:   "benchmark content",
	}
	var res Document
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < n; j++ {
			res = cloneDocument(doc)
		}
	}
	_ = res // prevent compiler optimization
}

// Helper for benchmarking cloneDocumentShort with n clones per iteration.
func benchmarkCloneDocumentShortN(b *testing.B, n int) {
	doc := &Document{
		ID:        "bench",
		Metadata:  map[string]string{"foo": "bar", "baz": "qux"},
		Embedding: []float32{1.0, 2.0, 3.0},
		Content:   "benchmark content",
	}
	var res Document
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < n; j++ {
			res = cloneDocumentShort(doc)
		}
	}
	_ = res // prevent compiler optimization
}

// randomString returns a random string of length n using lowercase letters and space.
func randomString(r *rand.Rand, n int) string {
	// We add 5 spaces to get roughly one space every 5 characters
	characters := []rune("abcdefghijklmnopqrstuvwxyz     ")

	b := make([]rune, n)
	for i := range b {
		b[i] = characters[r.Intn(len(characters))]
	}
	return string(b)
}
