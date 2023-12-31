package chromem

import (
	"context"
	"sync"
)

// EmbeddingFunc is a function that creates embeddings for a given document.
// chromem-go will use OpenAI`s ada v2 model by default, but you can provide your
// own function, using any model you like.
type EmbeddingFunc func(ctx context.Context, document string) ([]float32, error)

// Client is the chromem-go client. It holds collections, which hold documents.
//
//	+--------+    1-n    +------------+    n-n    +----------+
//	| Client |-----------| Collection |-----------| Document |
//	+--------+           +------------+           +----------+
type Client struct {
	collections     map[string]*Collection
	collectionsLock sync.RWMutex
}

// NewClient creates a new chromem-go client.
func NewClient() *Client {
	return &Client{
		collections: make(map[string]*Collection),
	}
}

// CreateCollection creates a new collection with the given name and metadata.
//
//   - name: The name of the collection to create.
//   - metadata: Optional metadata to associate with the collection.
//   - embeddingFunc: Optional function to use to embed documents.
//     Uses the default embedding function if not provided.
func (c *Client) CreateCollection(name string, metadata map[string]string, embeddingFunc EmbeddingFunc) *Collection {
	if embeddingFunc == nil {
		embeddingFunc = CreateEmbeddingsDefault()
	}
	collection := newCollection(name, metadata, embeddingFunc)

	c.collectionsLock.Lock()
	defer c.collectionsLock.Unlock()
	c.collections[name] = collection
	return collection
}
