package chromem

import (
	"context"
	"slices"
	"testing"
)

func TestDB_CreateCollection(t *testing.T) {
	// Values in the collection
	name := "test"
	metadata := map[string]string{"foo": "bar"}
	vectors := []float32{-0.1, 0.1, 0.2}
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return vectors, nil
	}

	db := NewDB()

	t.Run("OK", func(t *testing.T) {
		c, err := db.CreateCollection(name, metadata, embeddingFunc, nil)
		if err != nil {
			t.Fatal("expected no error, got", err)
		}
		if c == nil {
			t.Fatal("expected collection, got nil")
		}

		// Check expectations

		// DB should have one collection now
		if len(db.collections) != 1 {
			t.Fatal("expected 1 collection, got", len(db.collections))
		}
		// The collection should be the one we just created
		c2, ok := db.collections[name]
		if !ok {
			t.Fatal("expected collection", name, "not found")
		}
		if c2.Name != name {
			t.Fatal("expected name", name, "got", c2.Name)
		}
		// The returned collection should also be the same
		if c.Name != name {
			t.Fatal("expected name", name, "got", c.Name)
		}
		// The collection's persistent dir should be empty
		if c.persistDirectory != "" {
			t.Fatal("expected empty persistent directory, got", c.persistDirectory)
		}
		// It's metadata should match
		if len(c.metadata) != 1 || c.metadata["foo"] != "bar" {
			t.Fatal("expected metadata", metadata, "got", c.metadata)
		}
		// Documents should be empty, but not nil
		if c.documents == nil {
			t.Fatal("expected non-nil documents, got nil")
		}
		if len(c.documents) != 0 {
			t.Fatal("expected empty documents, got", len(c.documents))
		}
		// The embedding function should be the one we passed
		gotVectors, err := c.embed(context.Background(), "test")
		if err != nil {
			t.Fatal("expected no error, got", err)
		}
		if !slices.Equal(gotVectors, vectors) {
			t.Fatal("expected vectors", vectors, "got", gotVectors)
		}
	})

	t.Run("NOK - Empty name", func(t *testing.T) {
		_, err := db.CreateCollection("", metadata, embeddingFunc, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestDB_ListCollections(t *testing.T) {
	// Values in the collection
	name := "test"
	metadata := map[string]string{"foo": "bar"}
	vectors := []float32{-0.1, 0.1, 0.2}
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return vectors, nil
	}

	// Create initial collection
	db := NewDB()
	// We ignore the return value. CreateCollection is tested elsewhere.
	_, err := db.CreateCollection(name, metadata, embeddingFunc, nil)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}

	// List collections
	res := db.ListCollections()

	// Check expectations

	// Should've returned a map with one collection
	if len(res) != 1 {
		t.Fatal("expected 1 collection, got", len(res))
	}
	// The collection should be the one we just created
	c, ok := res[name]
	if !ok {
		t.Fatal("expected collection", name, "not found")
	}
	if c.Name != name {
		t.Fatal("expected name", name, "got", c.Name)
	}
	// The collection's persistent dir should be empty
	if c.persistDirectory != "" {
		t.Fatal("expected empty persistent directory, got", c.persistDirectory)
	}
	// It's metadata should match
	if len(c.metadata) != 1 || c.metadata["foo"] != "bar" {
		t.Fatal("expected metadata", metadata, "got", c.metadata)
	}
	// Documents should be empty, but not nil
	if c.documents == nil {
		t.Fatal("expected non-nil documents, got nil")
	}
	if len(c.documents) != 0 {
		t.Fatal("expected empty documents, got", len(c.documents))
	}
	// The embedding function should be the one we passed
	gotVectors, err := c.embed(context.Background(), "test")
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	if !slices.Equal(gotVectors, vectors) {
		t.Fatal("expected vectors", vectors, "got", gotVectors)
	}

	// And it should be a copy. Adding a value here should not reflect on the DB's
	// collection.
	res["foo"] = &Collection{}
	if len(db.ListCollections()) != 1 {
		t.Fatal("expected 1 collection, got", len(db.ListCollections()))
	}
}

func TestDB_GetCollection(t *testing.T) {
	// Values in the collection
	name := "test"
	metadata := map[string]string{"foo": "bar"}
	vectors := []float32{-0.1, 0.1, 0.2}
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return vectors, nil
	}

	// Create initial collection
	db := NewDB()
	// We ignore the return value. CreateCollection is tested elsewhere.
	_, err := db.CreateCollection(name, metadata, embeddingFunc, nil)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}

	// Get collection
	c := db.GetCollection(name, nil)

	// Check expectations
	if c.Name != name {
		t.Fatal("expected name", name, "got", c.Name)
	}
	// The collection's persistent dir should be empty
	if c.persistDirectory != "" {
		t.Fatal("expected empty persistent directory, got", c.persistDirectory)
	}
	// It's metadata should match
	if len(c.metadata) != 1 || c.metadata["foo"] != "bar" {
		t.Fatal("expected metadata", metadata, "got", c.metadata)
	}
	// Documents should be empty, but not nil
	if c.documents == nil {
		t.Fatal("expected non-nil documents, got nil")
	}
	if len(c.documents) != 0 {
		t.Fatal("expected empty documents, got", len(c.documents))
	}
	// The embedding function should be the one we passed
	gotVectors, err := c.embed(context.Background(), "test")
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	if !slices.Equal(gotVectors, vectors) {
		t.Fatal("expected vectors", vectors, "got", gotVectors)
	}
}

func TestDB_GetOrCreateCollection(t *testing.T) {
	// Values in the collection
	name := "test"
	metadata := map[string]string{"foo": "bar"}
	vectors := []float32{-0.1, 0.1, 0.2}
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return vectors, nil
	}

	t.Run("Get", func(t *testing.T) {
		// Create initial collection
		db := NewDB()
		// Create collection so that the GetOrCreateCollection() call below only
		// gets it.
		// We ignore the return value. CreateCollection is tested elsewhere.
		_, err := db.CreateCollection(name, metadata, embeddingFunc, nil)
		if err != nil {
			t.Fatal("expected no error, got", err)
		}

		// Call GetOrCreateCollection() with the same name to only get it. We pass
		// nil for the metadata and embeddingFunc so we can check that the returned
		// collection is the original one, and not a new one.
		c, err := db.GetOrCreateCollection(name, nil, embeddingFunc, nil)
		if err != nil {
			t.Fatal("expected no error, got", err)
		}
		if c == nil {
			t.Fatal("expected collection, got nil")
		}

		// Check expectations
		if c.Name != name {
			t.Fatal("expected name", name, "got", c.Name)
		}
		// The collection's persistent dir should be empty
		if c.persistDirectory != "" {
			t.Fatal("expected empty persistent directory, got", c.persistDirectory)
		}
		// It's metadata should match
		if len(c.metadata) != 1 || c.metadata["foo"] != "bar" {
			t.Fatal("expected metadata", metadata, "got", c.metadata)
		}
		// Documents should be empty, but not nil
		if c.documents == nil {
			t.Fatal("expected non-nil documents, got nil")
		}
		if len(c.documents) != 0 {
			t.Fatal("expected empty documents, got", len(c.documents))
		}
		// The embedding function should be the one we passed
		gotVectors, err := c.embed(context.Background(), "test")
		if err != nil {
			t.Fatal("expected no error, got", err)
		}
		if !slices.Equal(gotVectors, vectors) {
			t.Fatal("expected vectors", vectors, "got", gotVectors)
		}
	})

	t.Run("Create", func(t *testing.T) {
		// Create initial collection
		db := NewDB()

		// Call GetOrCreateCollection()
		c, err := db.GetOrCreateCollection(name, metadata, embeddingFunc, nil)
		if err != nil {
			t.Fatal("expected no error, got", err)
		}
		if c == nil {
			t.Fatal("expected collection, got nil")
		}

		// Check like we check CreateCollection()
		if c.Name != name {
			t.Fatal("expected name", name, "got", c.Name)
		}
		// The collection's persistent dir should be empty
		if c.persistDirectory != "" {
			t.Fatal("expected empty persistent directory, got", c.persistDirectory)
		}
		// It's metadata should match
		if len(c.metadata) != 1 || c.metadata["foo"] != "bar" {
			t.Fatal("expected metadata", metadata, "got", c.metadata)
		}
		// Documents should be empty, but not nil
		if c.documents == nil {
			t.Fatal("expected non-nil documents, got nil")
		}
		if len(c.documents) != 0 {
			t.Fatal("expected empty documents, got", len(c.documents))
		}
		// The embedding function should be the one we passed
		gotVectors, err := c.embed(context.Background(), "test")
		if err != nil {
			t.Fatal("expected no error, got", err)
		}
		if !slices.Equal(gotVectors, vectors) {
			t.Fatal("expected vectors", vectors, "got", gotVectors)
		}
	})
}

func TestDB_DeleteCollection(t *testing.T) {
	// Values in the collection
	name := "test"
	metadata := map[string]string{"foo": "bar"}
	vectors := []float32{-0.1, 0.1, 0.2}
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return vectors, nil
	}

	// Create initial collection
	db := NewDB()
	// We ignore the return value. CreateCollection is tested elsewhere.
	_, err := db.CreateCollection(name, metadata, embeddingFunc, nil)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}

	// Delete collection
	db.DeleteCollection(name)

	// Check expectations
	// We don't have access to the documents field, but we can rely on DB.ListCollections()
	// because it's tested elsewhere.
	if len(db.ListCollections()) != 0 {
		t.Fatal("expected 0 collections, got", len(db.ListCollections()))
	}
	// Also check internally
	if len(db.collections) != 0 {
		t.Fatal("expected 0 collections, got", len(db.collections))
	}
}

func TestDB_Reset(t *testing.T) {
	// Values in the collection
	name := "test"
	metadata := map[string]string{"foo": "bar"}
	vectors := []float32{-0.1, 0.1, 0.2}
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return vectors, nil
	}

	// Create initial collection
	db := NewDB()
	// We ignore the return value. CreateCollection is tested elsewhere.
	_, err := db.CreateCollection(name, metadata, embeddingFunc, nil)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}

	// Reset DB
	db.Reset()

	// Check expectations
	// We don't have access to the documents field, but we can rely on DB.ListCollections()
	// because it's tested elsewhere.
	if len(db.ListCollections()) != 0 {
		t.Fatal("expected 0 collections, got", len(db.ListCollections()))
	}
	// Also check internally
	if len(db.collections) != 0 {
		t.Fatal("expected 0 collections, got", len(db.collections))
	}
}
