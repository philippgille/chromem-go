package chromem_test

import (
	"context"
	"testing"

	"github.com/philippgille/chromem-go"
)

func TestDB_CreateCollection(t *testing.T) {
	// Values in the collection
	name := "test"
	metadata := map[string]string{"foo": "bar"}
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return []float32{-0.1, 0.1, 0.2}, nil
	}

	// Create collection
	db := chromem.NewDB()
	c, err := db.CreateCollection(name, metadata, embeddingFunc)
	if err != nil {
		t.Error("expected no error, got", err)
	}
	if c == nil {
		t.Error("expected collection, got nil")
	}

	// Check expectations
	if c.Name != name {
		t.Error("expected name", name, "got", c.Name)
	}
	// TODO: Check metadata etc when they become accessible
}

func TestDB_ListCollections(t *testing.T) {
	// Values in the collection
	name := "test"
	metadata := map[string]string{"foo": "bar"}
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return []float32{-0.1, 0.1, 0.2}, nil
	}

	// Create initial collection
	db := chromem.NewDB()
	// We ignore the return value. CreateCollection is tested elsewhere.
	_, err := db.CreateCollection(name, metadata, embeddingFunc)
	if err != nil {
		t.Error("expected no error, got", err)
	}

	// List collections
	res := db.ListCollections()

	// Check expectations
	if len(res) != 1 {
		t.Error("expected 1 collection, got", len(res))
	}
	c, ok := res[name]
	if !ok {
		t.Error("expected collection", name, "not found")
	}
	if c.Name != name {
		t.Error("expected name", name, "got", c.Name)
	}
	// TODO: Check metadata when it's accessible (e.g. with GetMetadata())
	// if len(c.Metadata) != 1 {
	// 	t.Error("expected 1 metadata, got", len(c.Metadata))
	// }
	// if c.Metadata["foo"] != "bar" {
	// 	t.Error("expected metadata", metadata, "got", c.Metadata)
	// }
	// TODO: Same for documents and EmbeddingFunc

	// And it should be a copy. Adding a value here should not reflect on the DB's
	// collection.
	res["foo"] = &chromem.Collection{}
	if len(db.ListCollections()) != 1 {
		t.Error("expected 1 collection, got", len(db.ListCollections()))
	}
}

func TestDB_GetCollection(t *testing.T) {
	// Values in the collection
	name := "test"
	metadata := map[string]string{"foo": "bar"}
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return []float32{-0.1, 0.1, 0.2}, nil
	}

	// Create initial collection
	db := chromem.NewDB()
	// We ignore the return value. CreateCollection is tested elsewhere.
	_, err := db.CreateCollection(name, metadata, embeddingFunc)
	if err != nil {
		t.Error("expected no error, got", err)
	}

	// Get collection
	c := db.GetCollection(name)

	// Check expectations
	if c.Name != name {
		t.Error("expected name", name, "got", c.Name)
	}
	// TODO: Check metadata when it's accessible (e.g. with GetMetadata())
	// if len(c.Metadata) != 1 {
	// 	t.Error("expected 1 metadata, got", len(c.Metadata))
	// }
	// if c.Metadata["foo"] != "bar" {
	// 	t.Error("expected metadata", metadata, "got", c.Metadata)
	// }
	// TODO: Check documents content as soon as we have access to them
	// TODO: Same for the EmbeddingFunc
	// TODO: Check documents map being a copy as soon as we have access to it
}

func TestDB_GetOrCreateCollection(t *testing.T) {
	// Values in the collection
	name := "test"
	metadata := map[string]string{"foo": "bar"}
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return []float32{-0.1, 0.1, 0.2}, nil
	}

	t.Run("Get", func(t *testing.T) {
		// Create initial collection
		db := chromem.NewDB()
		// Create collection so that the GetOrCreateCollection() call below only
		// gets it.
		// We ignore the return value. CreateCollection is tested elsewhere.
		_, err := db.CreateCollection(name, metadata, embeddingFunc)
		if err != nil {
			t.Error("expected no error, got", err)
		}

		// Call GetOrCreateCollection() with the same name to only get it. We pass
		// nil for the metadata and embeddingFunc so we can check that the returned
		// collection is the original one, and not a new one.
		c, err := db.GetOrCreateCollection(name, nil, nil)
		if err != nil {
			t.Error("expected no error, got", err)
		}
		if c == nil {
			t.Error("expected collection, got nil")
		}

		// Check expectations
		if c.Name != name {
			t.Error("expected name", name, "got", c.Name)
		}
		// TODO: Check metadata when it's accessible (e.g. with GetMetadata())
	})

	t.Run("Create", func(t *testing.T) {
		// Create initial collection
		db := chromem.NewDB()

		// Call GetOrCreateCollection()
		c, err := db.GetOrCreateCollection(name, metadata, embeddingFunc)
		if err != nil {
			t.Error("expected no error, got", err)
		}
		if c == nil {
			t.Error("expected collection, got nil")
		}

		// Check like we check CreateCollection()
		if c.Name != name {
			t.Error("expected name", name, "got", c.Name)
		}
		// TODO: Check metadata when it's accessible (e.g. with GetMetadata())
	})
}

func TestDB_DeleteCollection(t *testing.T) {
	// Values in the collection
	name := "test"
	metadata := map[string]string{"foo": "bar"}
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return []float32{-0.1, 0.1, 0.2}, nil
	}

	// Create initial collection
	db := chromem.NewDB()
	// We ignore the return value. CreateCollection is tested elsewhere.
	_, err := db.CreateCollection(name, metadata, embeddingFunc)
	if err != nil {
		t.Error("expected no error, got", err)
	}

	// Delete collection
	db.DeleteCollection(name)

	// Check expectations
	// We don't have access to the documents field, but we can rely on DB.ListCollections()
	// because it's tested elsewhere.
	if len(db.ListCollections()) != 0 {
		t.Error("expected 0 collections, got", len(db.ListCollections()))
	}
}

func TestDB_Reset(t *testing.T) {
	// Values in the collection
	name := "test"
	metadata := map[string]string{"foo": "bar"}
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return []float32{-0.1, 0.1, 0.2}, nil
	}

	// Create initial collection
	db := chromem.NewDB()
	// We ignore the return value. CreateCollection is tested elsewhere.
	_, err := db.CreateCollection(name, metadata, embeddingFunc)
	if err != nil {
		t.Error("expected no error, got", err)
	}

	// Reset DB
	db.Reset()

	// Check expectations
	// We don't have access to the documents field, but we can rely on DB.ListCollections()
	// because it's tested elsewhere.
	if len(db.ListCollections()) != 0 {
		t.Error("expected 0 collections, got", len(db.ListCollections()))
	}
}
