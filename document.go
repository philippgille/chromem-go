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

	// ⚠️ When adding unexported fields here, consider adding a persistence struct
	// version of this in [DB.Export] and [DB.Import].
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

	return Document{
		ID:        id,
		Metadata:  metadata,
		Embedding: embedding,
		Content:   content,
	}, nil
}
