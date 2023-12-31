package chromem

import (
	"context"
)

type document struct {
	ID       string
	Metadata map[string]string
	Document string

	Vectors []float32
}

// newDocument creates a new document, including its embeddings.
// If the embeddings are not provided, they are created using the embedding function.
func newDocument(ctx context.Context, id string, embeddings []float32, metadata map[string]string, doc string, embed EmbeddingFunc) (*document, error) {
	if len(embeddings) == 0 {
		vectors, err := embed(ctx, doc)
		if err != nil {
			return nil, err
		}
		embeddings = vectors
	}

	return &document{
		ID:       id,
		Metadata: metadata,
		Document: doc,

		Vectors: embeddings,
	}, nil
}
