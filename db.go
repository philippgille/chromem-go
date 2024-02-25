package chromem

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
	collections      map[string]*Collection
	collectionsLock  sync.RWMutex
	persistDirectory string
}

// NewDB creates a new in-memory chromem-go DB.
func NewDB() *DB {
	return &DB{
		collections: make(map[string]*Collection),
	}
}

// NewPersistentDB creates a new persistent chromem-go DB.
// If the path is empty, it defaults to "./chromem-go".
// The persistence covers the collections (including their documents) and the metadata.
// However it doesn't cover the EmbeddingFunc, as functions can't be serialized.
// When some data is persisted and you create a new persistent DB with the same
// path, you'll have to provide the same EmbeddingFunc as before when getting an
// existing collection and adding more documents to it.
func NewPersistentDB(path string) (*DB, error) {
	if path == "" {
		path = "./chromem-go"
	} else {
		// Clean in case the user provides something like "./db/../db"
		path = filepath.Clean(path)
	}

	db := &DB{
		persistDirectory: path,
		collections:      make(map[string]*Collection),
	}

	// If the directory doesn't exist, create it and return an empty DB.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.MkdirAll(path, 0o700)
		if err != nil {
			return nil, fmt.Errorf("couldn't create persistence directory: %w", err)
		}

		return db, nil
	}

	// Otherwise, read all collections and their documents from the directory.
	err := filepath.WalkDir(path, func(p string, info os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("couldn't walk DB directory: %w", err)
		}
		// WalkDir reads root, which we can skip.
		if path == p {
			return nil
		}
		// First level is the subdirectories for the collections, so skip any files.
		if !info.IsDir() {
			return nil
		}
		// For each subdirectory, create a collection and read its name, metadata
		// and documents.
		// TODO: Parallelize this (e.g. chan with $numCPU buffer and $numCPU goroutines
		// reading from it).
		c := &Collection{
			// We can fill Name, persistDirectory and metadata only after reading
			// the metadata.
			documents: make(map[string]*document),
			// We can fill embed only when the user calls DB.GetCollection() or
			// DB.GetOrCreateCollection().
		}
		collectionPath := filepath.Join(path, info.Name())
		err = filepath.WalkDir(collectionPath, func(p string, info os.DirEntry, err error) error {
			if err != nil {
				return fmt.Errorf("couldn't walk collection directory: %w", err)
			}
			// Files should be metadata and documents; skip subdirectories.
			if info.IsDir() {
				return nil
			}

			if info.Name() == metadataFileName+".gob" {
				pc := struct {
					Name     string
					Metadata map[string]string
				}{}
				err := read(p, &pc)
				if err != nil {
					return fmt.Errorf("couldn't read collection metadata: %w", err)
				}
				c.Name = pc.Name
				c.persistDirectory = filepath.Dir(p)
				c.metadata = pc.Metadata
			} else {
				// Read document
				d := &document{}
				err := read(p, d)
				if err != nil {
					return fmt.Errorf("couldn't read document: %w", err)
				}
				c.documents[d.ID] = d
			}

			return nil
		})
		if err != nil {
			return fmt.Errorf("couldn't read collection directory: %w", err)
		}
		db.collections[c.Name] = c

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("couldn't read persisted database: %w", err)
	}

	return db, nil
}

// CreateCollection creates a new collection with the given name and metadata.
//
//   - name: The name of the collection to create.
//   - metadata: Optional metadata to associate with the collection.
//   - embeddingFunc: Optional function to use to embed documents.
//     Uses the default embedding function if not provided.
func (db *DB) CreateCollection(name string, metadata map[string]string, embeddingFunc EmbeddingFunc) (*Collection, error) {
	if embeddingFunc == nil {
		embeddingFunc = NewEmbeddingFuncDefault()
	}
	collection, err := newCollection(name, metadata, embeddingFunc, db.persistDirectory)
	if err != nil {
		return nil, fmt.Errorf("couldn't create collection: %w", err)
	}

	db.collectionsLock.Lock()
	defer db.collectionsLock.Unlock()
	db.collections[name] = collection
	return collection, nil
}

// ListCollections returns all collections in the DB, mapping name->Collection.
// The returned map is a copy of the internal map, so it's safe to directly modify
// the map itself. Direct modifications of the map won't reflect on the DB's map.
// To do that use the DB's methods like CreateCollection() and DeleteCollection().
// The map is not an entirely deep clone, so the collections themselves are still
// the original ones. Any methods on the collections like Add() for adding documents
// will be reflected on the DB's collections and are concurrency-safe.
func (db *DB) ListCollections() map[string]*Collection {
	db.collectionsLock.RLock()
	defer db.collectionsLock.RUnlock()

	res := make(map[string]*Collection, len(db.collections))
	for k, v := range db.collections {
		res[k] = v
	}

	return res
}

// GetCollection returns the collection with the given name.
// The returned value is a reference to the original collection, so any methods
// on the collection like Add() will be reflected on the DB's collection. Those
// operations are concurrency-safe.
// If the collection doesn't exist, this returns nil.
func (db *DB) GetCollection(name string) *Collection {
	db.collectionsLock.RLock()
	defer db.collectionsLock.RUnlock()
	return db.collections[name]
}

// GetOrCreateCollection returns the collection with the given name if it exists
// in the DB, or otherwise creates it. When creating:
//
//   - name: The name of the collection to create.
//   - metadata: Optional metadata to associate with the collection.
//   - embeddingFunc: Optional function to use to embed documents.
//     Uses the default embedding function if not provided.
func (db *DB) GetOrCreateCollection(name string, metadata map[string]string, embeddingFunc EmbeddingFunc) (*Collection, error) {
	// No need to lock here, because the methods we call do that.
	collection := db.GetCollection(name)
	if collection == nil {
		var err error
		collection, err = db.CreateCollection(name, metadata, embeddingFunc)
		if err != nil {
			return nil, fmt.Errorf("couldn't create collection: %w", err)
		}
	}
	return collection, nil
}

// DeleteCollection deletes the collection with the given name.
// If the collection doesn't exist, this is a no-op.
func (db *DB) DeleteCollection(name string) {
	db.collectionsLock.Lock()
	defer db.collectionsLock.Unlock()
	delete(db.collections, name)
}

// Reset removes all collections from the DB.
func (db *DB) Reset() {
	db.collectionsLock.Lock()
	defer db.collectionsLock.Unlock()
	db.collections = make(map[string]*Collection)
}
