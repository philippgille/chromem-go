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

// ListCollections returns all collections in the DB, mapping name->Collection.
// The returned map is a copy of the internal map, so it's safe to directly modify
// the map itself. Direct modifications of the map won't reflect on the DB's map.
// To do that use the DB's methods like CreateCollection() and DeleteCollection().
// The map is not an entirely deep clone, so the collections themselves are still
// the original ones. Any methods on the collections like Add() for adding documents
// will be reflected on the DB's collections and are concurrency-safe.
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
// The returned value is a reference to the original collection, so any methods
// on the collection like Add() will be reflected on the DB's collection. Those
// operations are concurrency-safe.
// If the collection doesn't exist, this returns nil.
func (c *DB) GetCollection(name string) *Collection {
	c.collectionsLock.RLock()
	defer c.collectionsLock.RUnlock()
	return c.collections[name]
}

// GetOrCreateCollection returns the collection with the given name if it exists
// in the DB, or otherwise creates it. When creating:
//
//   - name: The name of the collection to create.
//   - metadata: Optional metadata to associate with the collection.
//   - embeddingFunc: Optional function to use to embed documents.
//     Uses the default embedding function if not provided.
func (c *DB) GetOrCreateCollection(name string, metadata map[string]string, embeddingFunc EmbeddingFunc) *Collection {
	// No need to lock here, because the methods we call do that.
	collection := c.GetCollection(name)
	if collection == nil {
		collection = c.CreateCollection(name, metadata, embeddingFunc)
	}
	return collection
}

// DeleteCollection deletes the collection with the given name.
// If the collection doesn't exist, this is a no-op.
func (c *DB) DeleteCollection(name string) {
	c.collectionsLock.Lock()
	defer c.collectionsLock.Unlock()
	delete(c.collections, name)
}

// Reset removes all collections from the DB.
func (c *DB) Reset() {
	c.collectionsLock.Lock()
	defer c.collectionsLock.Unlock()
	c.collections = make(map[string]*Collection)
}
