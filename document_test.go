package chromem

import (
	"context"
	"reflect"
	"testing"
)

func TestDocument_New(t *testing.T) {
	ctx := context.Background()
	id := "test"
	metadata := map[string]string{"foo": "bar"}
	vectors := []float32{-0.40824828, 0.40824828, 0.81649655} // normalized version of `{-0.1, 0.1, 0.2}`
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
				t.Fatal("expected no error, got", err)
			}
			// We can compare with DeepEqual after removing the embedding function
			d.Embedding = nil
			exp := Document{
				ID:       id,
				Metadata: metadata,
				Content:  content,
			}
			if !reflect.DeepEqual(exp, d) {
				t.Fatalf("expected %+v, got %+v", exp, d)
			}
		})
	}
}
