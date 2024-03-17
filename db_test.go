package chromem

import (
	"context"
	"reflect"
	"slices"
	"testing"
)

func TestDB_CreateCollection(t *testing.T) {
	// Values in the collection
	name := "test"
	metadata := map[string]string{"foo": "bar"}
	vectors := []float32{-0.40824828, 0.40824828, 0.81649655} // normalized version of `{-0.1, 0.1, 0.2}`
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return vectors, nil
	}

	db := NewDB()

	t.Run("OK", func(t *testing.T) {
		c, err := db.CreateCollection(name, metadata, embeddingFunc)
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
		// Check the embedding function first, then the rest with DeepEqual
		gotVectors, err := c.embed(context.Background(), "test")
		if err != nil {
			t.Fatal("expected no error, got", err)
		}
		if !slices.Equal(gotVectors, vectors) {
			t.Fatal("expected vectors", vectors, "got", gotVectors)
		}
		c.embed, c2.embed = nil, nil
		if !reflect.DeepEqual(c, c2) {
			t.Fatalf("expected collection %+v, got %+v", c, c2)
		}
	})

	t.Run("NOK - Empty name", func(t *testing.T) {
		_, err := db.CreateCollection("", metadata, embeddingFunc)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestDB_ListCollections(t *testing.T) {
	// Values in the collection
	name := "test"
	metadata := map[string]string{"foo": "bar"}
	vectors := []float32{-0.40824828, 0.40824828, 0.81649655} // normalized version of `{-0.1, 0.1, 0.2}`
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return vectors, nil
	}

	// Create initial collection
	db := NewDB()
	orig, err := db.CreateCollection(name, metadata, embeddingFunc)
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
	// Check the embedding function first, then the rest with DeepEqual
	gotVectors, err := c.embed(context.Background(), "test")
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	if !slices.Equal(gotVectors, vectors) {
		t.Fatal("expected vectors", vectors, "got", gotVectors)
	}
	orig.embed, c.embed = nil, nil
	if !reflect.DeepEqual(orig, c) {
		t.Fatalf("expected collection %+v, got %+v", orig, c)
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
	vectors := []float32{-0.40824828, 0.40824828, 0.81649655} // normalized version of `{-0.1, 0.1, 0.2}`
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return vectors, nil
	}

	// Create initial collection
	db := NewDB()
	orig, err := db.CreateCollection(name, metadata, embeddingFunc)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}

	// Get collection
	c := db.GetCollection(name, nil)

	// Check the embedding function first, then the rest with DeepEqual
	gotVectors, err := c.embed(context.Background(), "test")
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	if !slices.Equal(gotVectors, vectors) {
		t.Fatal("expected vectors", vectors, "got", gotVectors)
	}
	orig.embed, c.embed = nil, nil
	if !reflect.DeepEqual(orig, c) {
		t.Fatalf("expected collection %+v, got %+v", orig, c)
	}
}

func TestDB_GetOrCreateCollection(t *testing.T) {
	// Values in the collection
	name := "test"
	metadata := map[string]string{"foo": "bar"}
	vectors := []float32{-0.40824828, 0.40824828, 0.81649655} // normalized version of `{-0.1, 0.1, 0.2}`
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return vectors, nil
	}

	t.Run("Get", func(t *testing.T) {
		// Create initial collection
		db := NewDB()
		// Create collection so that the GetOrCreateCollection() call below only
		// gets it.
		orig, err := db.CreateCollection(name, metadata, embeddingFunc)
		if err != nil {
			t.Fatal("expected no error, got", err)
		}

		// Call GetOrCreateCollection() with the same name to only get it. We pass
		// nil for the metadata and embeddingFunc so we can check that the returned
		// collection is the original one, and not a new one.
		c, err := db.GetOrCreateCollection(name, nil, nil)
		if err != nil {
			t.Fatal("expected no error, got", err)
		}
		if c == nil {
			t.Fatal("expected collection, got nil")
		}

		// Check the embedding function first, then the rest with DeepEqual
		gotVectors, err := c.embed(context.Background(), "test")
		if err != nil {
			t.Fatal("expected no error, got", err)
		}
		if !slices.Equal(gotVectors, vectors) {
			t.Fatal("expected vectors", vectors, "got", gotVectors)
		}
		orig.embed, c.embed = nil, nil
		if !reflect.DeepEqual(orig, c) {
			t.Fatalf("expected collection %+v, got %+v", orig, c)
		}
	})

	t.Run("Create", func(t *testing.T) {
		// Create initial collection
		db := NewDB()

		// Call GetOrCreateCollection()
		c, err := db.GetOrCreateCollection(name, metadata, embeddingFunc)
		if err != nil {
			t.Fatal("expected no error, got", err)
		}
		if c == nil {
			t.Fatal("expected collection, got nil")
		}

		// Check like we check CreateCollection()
		c2, ok := db.collections[name]
		if !ok {
			t.Fatal("expected collection", name, "not found")
		}
		gotVectors, err := c.embed(context.Background(), "test")
		if err != nil {
			t.Fatal("expected no error, got", err)
		}
		if !slices.Equal(gotVectors, vectors) {
			t.Fatal("expected vectors", vectors, "got", gotVectors)
		}
		c.embed, c2.embed = nil, nil
		if !reflect.DeepEqual(c, c2) {
			t.Fatalf("expected collection %+v, got %+v", c, c2)
		}
	})
}

func TestDB_DeleteCollection(t *testing.T) {
	// Values in the collection
	name := "test"
	metadata := map[string]string{"foo": "bar"}
	vectors := []float32{-0.40824828, 0.40824828, 0.81649655} // normalized version of `{-0.1, 0.1, 0.2}`
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return vectors, nil
	}

	// Create initial collection
	db := NewDB()
	// We ignore the return value. CreateCollection is tested elsewhere.
	_, err := db.CreateCollection(name, metadata, embeddingFunc)
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
	vectors := []float32{-0.40824828, 0.40824828, 0.81649655} // normalized version of `{-0.1, 0.1, 0.2}`
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return vectors, nil
	}

	// Create initial collection
	db := NewDB()
	// We ignore the return value. CreateCollection is tested elsewhere.
	_, err := db.CreateCollection(name, metadata, embeddingFunc)
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
