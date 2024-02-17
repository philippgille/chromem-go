package chromem

import (
	"context"
	"sync"
)

// EmbeddingFunc is a function that creates embeddings for a given document.
// chromem-go will use OpenAI`s "text-embedding-3-small" model by default,
// but you can provide your own function, using any model you like.
type EmbeddingFunc func(ctx context.Context, document string) ([]float32, error)

// DB is the chromem-go database. It holds collections, which hold documents.
//
//	+----+    1-n    +------------+    n-n    +----------+
//	| DB |-----------| Collection |-----------| Document |
//	+----+           +------------+           +----------+
type DB struct {
	collections     map[string]*Collection
	collectionsLock sync.RWMutex
}

// NewDB creates a new chromem-go DB.
func NewDB() *DB {
	return &DB{
		collections: make(map[string]*Collection),
	}
}

// CreateCollection creates a new collection with the given name and metadata.
//
//   - name: The name of the collection to create.
//   - metadata: Optional metadata to associate with the collection.
//   - embeddingFunc: Optional function to use to embed documents.
//     Uses the default embedding function if not provided.
func (c *DB) CreateCollection(name string, metadata map[string]string, embeddingFunc EmbeddingFunc) *Collection {
	if embeddingFunc == nil {
		embeddingFunc = NewEmbeddingFuncDefault()
	}
	collection := newCollection(name, metadata, embeddingFunc)

	c.collectionsLock.Lock()
	defer c.collectionsLock.Unlock()
	c.collections[name] = collection
	return collection
}

// ListCollections returns a map of all collections in the DB.
// The returned map is a copy of the internal map, to it's safe to modify the map
// itself. But it's not an entirely deep clone, so the collections themselves are
// still the original ones.
// You must not use them concurrently with other chromem-go operations that modify
// the collections (like adding a document).
func (c *DB) ListCollections() map[string]*Collection {
	c.collectionsLock.RLock()
	defer c.collectionsLock.RUnlock()

	res := make(map[string]*Collection, len(c.collections))
	for k, v := range c.collections {
		res[k] = v
	}

	return res
}
