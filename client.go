package chromem

import (
	"context"
	"sync"
)

type EmbeddingFunc func(ctx context.Context, document string) ([]float32, error)

type Client struct {
	embed EmbeddingFunc

	collections     map[string]*Collection
	collectionsLock sync.RWMutex
}

func NewClient() *Client {
	return &Client{
		collections: make(map[string]*Collection),
	}
}

// CreateCollection creates a new collection with the given name and metadata.
//
//   - name: The name of the collection to create.
//   - metadata: Optional metadata to associate with the collection.
//   - embedding_function: Optional function to use to embed documents.
//     Uses the default embedding function if not provided.
func (c *Client) CreateCollection(name string, metadata map[string]string, embeddingFunc EmbeddingFunc) *Collection {
	if embeddingFunc == nil {
		embeddingFunc = createEmbeddings
	}
	collection := newCollection(name, metadata, embeddingFunc)

	c.collectionsLock.Lock()
	defer c.collectionsLock.Unlock()
	c.collections[name] = collection
	return collection
}
