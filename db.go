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

// GetCollection returns the collection with the given name.
// The collection is a copy of the original, except for its EmbeddingFunc.
// It's only copied one level deep though. So while you *can* manipulate the collection's
// map of documents, you must not manipulate the documents themselves.
// Regarding the EmbeddingFunc it's the original. So if it closes over some state,
// this state is shared. But usually an EmbeddingFunc just closes over an API key
// or HTTP client, which are safe to share.
func (c *DB) GetCollection(name string) *Collection {
	c.collectionsLock.RLock()
	defer c.collectionsLock.RUnlock()

	orig, ok := c.collections[name]
	if !ok {
		return nil
	}

	newMetadata := make(map[string]string, len(orig.Metadata))
	for k, v := range orig.Metadata {
		newMetadata[k] = v
	}

	orig.documentsLock.RLock()
	defer orig.documentsLock.RUnlock()
	newDocuments := make(map[string]*document, len(orig.documents))
	for k, v := range orig.documents {
		newDocuments[k] = v
	}

	return &Collection{
		Name:     orig.Name,
		Metadata: newMetadata,

		documents: make(map[string]*document, len(orig.documents)),

		embed: orig.embed,
	}
}

// DeleteCollection deletes the collection with the given name.
// If the collection doesn't exist, this is a no-op.
func (c *DB) DeleteCollection(name string) {
	c.collectionsLock.Lock()
	defer c.collectionsLock.Unlock()
	delete(c.collections, name)
}
