package chromem

import (
	"context"
	"slices"
	"testing"
)

func TestDocument_New(t *testing.T) {
	ctx := context.Background()
	id := "test"
	metadata := map[string]string{"foo": "bar"}
	vectors := []float32{-0.1, 0.1, 0.2}
	content := "hello world"
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return vectors, nil
	}

	tt := []struct {
		name          string
		id            string
		metadata      map[string]string
		vectors       []float32
		content       string
		embeddingFunc EmbeddingFunc
	}{
		{
			name:          "No embedding",
			id:            id,
			metadata:      metadata,
			vectors:       nil,
			content:       content,
			embeddingFunc: embeddingFunc,
		},
		{
			name:          "With embedding",
			id:            id,
			metadata:      metadata,
			vectors:       vectors,
			content:       content,
			embeddingFunc: embeddingFunc,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			// Create document
			d, err := NewDocument(ctx, id, metadata, vectors, content, embeddingFunc)
			if err != nil {
				t.Error("expected no error, got", err)
			}
			if d.ID != id {
				t.Error("expected id", id, "got", d.ID)
			}
			if d.Metadata["foo"] != metadata["foo"] {
				t.Error("expected metadata", metadata, "got", d.Metadata)
			}
			if !slices.Equal(d.Embedding, vectors) {
				t.Error("expected vectors", vectors, "got", d.Embedding)
			}
			if d.Content != content {
				t.Error("expected content", content, "got", d.Content)
			}
		})
	}
}
