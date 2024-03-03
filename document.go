package chromem

import (
	"context"
	"errors"
)

// Document represents a single document.
type Document struct {
	ID        string
	Metadata  map[string]string
	Embedding []float32
	Content   string
}

// NewDocument creates a new document, including its embeddings.
// Metadata is optional.
// If the embeddings are not provided, they are created using the embedding function.
// You can leave the content empty if you only want to store embeddings.
// If embeddingFunc is nil, the default embedding function is used.
//
// If you want to create a document without embeddings, for example to let [Collection.AddDocuments]
// create them concurrently, you can create a document with `chromem.Document{...}`
// instead of using this constructor.
func NewDocument(ctx context.Context, id string, metadata map[string]string, embedding []float32, content string, embeddingFunc EmbeddingFunc) (Document, error) {
	if id == "" {
		return Document{}, errors.New("id is empty")
	}
	if len(embedding) == 0 && content == "" {
		return Document{}, errors.New("either embedding or content must be filled")
	}
	if embeddingFunc == nil {
		embeddingFunc = NewEmbeddingFuncDefault()
	}

	if len(embedding) == 0 {
		var err error
		embedding, err = embeddingFunc(ctx, content)
		if err != nil {
			return Document{}, err
		}
	}

	// We copy the metadata to avoid data races in case the caller modifies the
	// map after creating the document while we range over it.
	m := make(map[string]string, len(metadata))
	for k, v := range metadata {
		m[k] = v
	}

	return Document{
		ID:        id,
		Metadata:  metadata,
		Embedding: embedding,
		Content:   content,
	}, nil
}
