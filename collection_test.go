package chromem_test

import (
	"context"
	"testing"

	"github.com/philippgille/chromem-go"
)

func TestCollection_Add(t *testing.T) {
	// Create collection
	db := chromem.NewDB()
	name := "test"
	metadata := map[string]string{"foo": "bar"}
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return []float32{-0.1, 0.1, 0.2}, nil
	}
	c := db.CreateCollection(name, metadata, embeddingFunc)
	if c == nil {
		t.Error("expected collection, got nil")
	}

	// Add document
	ids := []string{"1", "2"}
	metadatas := []map[string]string{{"foo": "bar"}, {"a": "b"}}
	documents := []string{"hello world", "hallo welt"}
	err := c.Add(context.Background(), ids, nil, metadatas, documents)
	if err != nil {
		t.Error("expected nil, got", err)
	}

	// TODO: Check expectations when documents become accessible
}
