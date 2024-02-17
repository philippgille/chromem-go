package chromem_test

import (
	"context"
	"testing"

	"github.com/philippgille/chromem-go"
)

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
	_ = db.CreateCollection(name, metadata, embeddingFunc)

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
	if len(c.Metadata) != 1 {
		t.Error("expected 1 metadata, got", len(c.Metadata))
	}
	if c.Metadata["foo"] != "bar" {
		t.Error("expected metadata", metadata, "got", c.Metadata)
	}

	// And it should be a copy
	res["foo"] = &chromem.Collection{}
	if len(db.ListCollections()) != 1 {
		t.Error("expected 1 collection, got", len(db.ListCollections()))
	}
}
