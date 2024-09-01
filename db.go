package chromem

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

// EmbeddingFunc is a function that creates embeddings for a given text.
// chromem-go will use OpenAI`s "text-embedding-3-small" model by default,
// but you can provide your own function, using any model you like.
// The function must return a *normalized* vector, i.e. the length of the vector
// must be 1. OpenAI's and Mistral's embedding models do this by default. Some
// others like Nomic's "nomic-embed-text-v1.5" don't.
type EmbeddingFunc func(ctx context.Context, text string) ([]float32, error)

// DB is the chromem-go database. It holds collections, which hold documents.
//
//	+----+    1-n    +------------+    n-n    +----------+
//	| DB |-----------| Collection |-----------| Document |
//	+----+           +------------+           +----------+
type DB struct {
	collections     map[string]*Collection
	collectionsLock sync.RWMutex

	persistDirectory string
	compress         bool

	// ⚠️ When adding fields here, consider adding them to the persistence struct
	// versions in [DB.Export] and [DB.Import] as well!
}

// NewDB creates a new in-memory chromem-go DB.
// While it doesn't write files when you add collections and documents, you can
// still use [DB.Export] and [DB.Import] to export and import the entire DB
// from a file.
func NewDB() *DB {
	return &DB{
		collections: make(map[string]*Collection),
	}
}

// NewPersistentDB creates a new persistent chromem-go DB.
// If the path is empty, it defaults to "./chromem-go".
// If compress is true, the files are compressed with gzip.
//
// The persistence covers the collections (including their documents) and the metadata.
// However, it doesn't cover the EmbeddingFunc, as functions can't be serialized.
// When some data is persisted, and you create a new persistent DB with the same
// path, you'll have to provide the same EmbeddingFunc as before when getting an
// existing collection and adding more documents to it.
//
// Currently, the persistence is done synchronously on each write operation, and
// each document addition leads to a new file, encoded as gob. In the future we
// will make this configurable (encoding, async writes, WAL-based writes, etc.).
//
// In addition to persistence for each added collection and document you can use
// [DB.ExportToFile] / [DB.ExportToWriter] and [DB.ImportFromFile] /
// [DB.ImportFromReader] to export and import the entire DB to/from a file or
// writer/reader, which also works for the pure in-memory DB.
func NewPersistentDB(path string, compress bool) (*DB, error) {
	if path == "" {
		path = "./chromem-go"
	} else {
		// Clean in case the user provides something like "./db/../db"
		path = filepath.Clean(path)
	}

	// We check for this file extension and skip others
	ext := ".gob"
	if compress {
		ext += ".gz"
	}

	db := &DB{
		collections:      make(map[string]*Collection),
		persistDirectory: path,
		compress:         compress,
	}

	// If the directory doesn't exist, create it and return an empty DB.
	fi, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			err := os.MkdirAll(path, 0o700)
			if err != nil {
				return nil, fmt.Errorf("couldn't create persistence directory: %w", err)
			}

			return db, nil
		}
		return nil, fmt.Errorf("couldn't get info about persistence directory: %w", err)
	} else if !fi.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", path)
	}

	// Otherwise, read all collections and their documents from the directory.
	dirEntries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("couldn't read persistence directory: %w", err)
	}
	for _, dirEntry := range dirEntries {
		// Collections are subdirectories, so skip any files (which the user might
		// have placed).
		if !dirEntry.IsDir() {
			continue
		}
		// For each subdirectory, create a collection and read its name, metadata
		// and documents.
		// TODO: Parallelize this (e.g. chan with $numCPU buffer and $numCPU goroutines
		// reading from it).
		collectionPath := filepath.Join(path, dirEntry.Name())
		collectionDirEntries, err := os.ReadDir(collectionPath)
		if err != nil {
			return nil, fmt.Errorf("couldn't read collection directory: %w", err)
		}
		c := &Collection{
			documents:        make(map[string]*Document),
			persistDirectory: collectionPath,
			compress:         compress,
			// We can fill Name and metadata only after reading
			// the metadata.
			// We can fill embed only when the user calls DB.GetCollection() or
			// DB.GetOrCreateCollection().
		}
		for _, collectionDirEntry := range collectionDirEntries {
			// Files should be metadata and documents; skip subdirectories which
			// the user might have placed.
			if collectionDirEntry.IsDir() {
				continue
			}

			fPath := filepath.Join(collectionPath, collectionDirEntry.Name())
			// Differentiate between collection metadata, documents and other files.
			if collectionDirEntry.Name() == metadataFileName+ext {
				// Read name and metadata
				pc := struct {
					Name     string
					Metadata map[string]string
				}{}
				err := readFromFile(fPath, &pc, "")
				if err != nil {
					return nil, fmt.Errorf("couldn't read collection metadata: %w", err)
				}
				c.Name = pc.Name
				c.metadata = pc.Metadata
			} else if strings.HasSuffix(collectionDirEntry.Name(), ext) {
				// Read document
				d := &Document{}
				err := readFromFile(fPath, d, "")
				if err != nil {
					return nil, fmt.Errorf("couldn't read document: %w", err)
				}
				c.documents[d.ID] = d
			} else {
				// Might be a file that the user has placed
				continue
			}
		}
		// If we have neither name nor documents, it was likely a user-added
		// directory, so skip it.
		if c.Name == "" && len(c.documents) == 0 {
			continue
		}
		// If we have no name, it means there was no metadata file
		if c.Name == "" {
			return nil, fmt.Errorf("collection metadata file not found: %s", collectionPath)
		}

		db.collections[c.Name] = c
	}

	return db, nil
}

// Import imports the DB from a file at the given path. The file must be encoded
// as gob and can optionally be compressed with flate (as gzip) and encrypted
// with AES-GCM.
// This works for both the in-memory and persistent DBs.
// Existing collections are overwritten.
//
// - filePath: Mandatory, must not be empty
// - encryptionKey: Optional, must be 32 bytes long if provided
//
// Deprecated: Use [DB.ImportFromFile] instead.
func (db *DB) Import(filePath string, encryptionKey string) error {
	return db.ImportFromFile(filePath, encryptionKey)
}

// ImportFromFile imports the DB from a file at the given path. The file must be
// encoded as gob and can optionally be compressed with flate (as gzip) and encrypted
// with AES-GCM.
// This works for both the in-memory and persistent DBs.
// Existing collections are overwritten.
//
//   - filePath: Mandatory, must not be empty
//   - encryptionKey: Optional, must be 32 bytes long if provided
//   - collections: Optional. If provided, only the collections with the given names
//     are imported. Non-existing collections are ignored.
//     If not provided, all collections are imported.
func (db *DB) ImportFromFile(filePath string, encryptionKey string, collections ...string) error {
	if filePath == "" {
		return fmt.Errorf("file path is empty")
	}
	if encryptionKey != "" {
		// AES 256 requires a 32 byte key
		if len(encryptionKey) != 32 {
			return errors.New("encryption key must be 32 bytes long")
		}
	}

	// If the file doesn't exist or is a directory, return an error.
	fi, err := os.Stat(filePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("file doesn't exist: %s", filePath)
		}
		return fmt.Errorf("couldn't get info about the file: %w", err)
	} else if fi.IsDir() {
		return fmt.Errorf("path is a directory: %s", filePath)
	}

	// Create persistence structs with exported fields so that they can be decoded
	// from gob.
	type persistenceCollection struct {
		Name      string
		Metadata  map[string]string
		Documents map[string]*Document
	}
	persistenceDB := struct {
		Collections map[string]*persistenceCollection
	}{
		Collections: make(map[string]*persistenceCollection, len(db.collections)),
	}

	db.collectionsLock.Lock()
	defer db.collectionsLock.Unlock()

	err = readFromFile(filePath, &persistenceDB, encryptionKey)
	if err != nil {
		return fmt.Errorf("couldn't read file: %w", err)
	}

	for _, pc := range persistenceDB.Collections {
		if len(collections) > 0 && !slices.Contains(collections, pc.Name) {
			continue
		}
		c := &Collection{
			Name: pc.Name,

			metadata:  pc.Metadata,
			documents: pc.Documents,
		}
		if db.persistDirectory != "" {
			c.persistDirectory = filepath.Join(db.persistDirectory, hash2hex(pc.Name))
			c.compress = db.compress
			err = c.persistMetadata()
			if err != nil {
				return fmt.Errorf("couldn't persist collection metadata: %w", err)
			}
			for _, doc := range c.documents {
				docPath := c.getDocPath(doc.ID)
				err = persistToFile(docPath, doc, c.compress, "")
				if err != nil {
					return fmt.Errorf("couldn't persist document to %q: %w", docPath, err)
				}
			}
		}
		db.collections[c.Name] = c
	}

	return nil
}

// ImportFromReader imports the DB from a reader. The stream must be encoded as
// gob and can optionally be compressed with flate (as gzip) and encrypted with
// AES-GCM.
// This works for both the in-memory and persistent DBs.
// Existing collections are overwritten.
// If the writer has to be closed, it's the caller's responsibility.
// This can be used to import DBs from object storage like S3. See
// https://github.com/philippgille/chromem-go/tree/main/examples/s3-export-import
// for an example.
//
//   - reader: An implementation of [io.ReadSeeker]
//   - encryptionKey: Optional, must be 32 bytes long if provided
//   - collections: Optional. If provided, only the collections with the given names
//     are imported. Non-existing collections are ignored.
//     If not provided, all collections are imported.
func (db *DB) ImportFromReader(reader io.ReadSeeker, encryptionKey string, collections ...string) error {
	if encryptionKey != "" {
		// AES 256 requires a 32 byte key
		if len(encryptionKey) != 32 {
			return errors.New("encryption key must be 32 bytes long")
		}
	}

	// Create persistence structs with exported fields so that they can be decoded
	// from gob.
	type persistenceCollection struct {
		Name      string
		Metadata  map[string]string
		Documents map[string]*Document
	}
	persistenceDB := struct {
		Collections map[string]*persistenceCollection
	}{
		Collections: make(map[string]*persistenceCollection, len(db.collections)),
	}

	db.collectionsLock.Lock()
	defer db.collectionsLock.Unlock()

	err := readFromReader(reader, &persistenceDB, encryptionKey)
	if err != nil {
		return fmt.Errorf("couldn't read stream: %w", err)
	}

	for _, pc := range persistenceDB.Collections {
		if len(collections) > 0 && !slices.Contains(collections, pc.Name) {
			continue
		}
		c := &Collection{
			Name: pc.Name,

			metadata:  pc.Metadata,
			documents: pc.Documents,
		}
		if db.persistDirectory != "" {
			c.persistDirectory = filepath.Join(db.persistDirectory, hash2hex(pc.Name))
			c.compress = db.compress
			err = c.persistMetadata()
			if err != nil {
				return fmt.Errorf("couldn't persist collection metadata: %w", err)
			}
			for _, doc := range c.documents {
				docPath := c.getDocPath(doc.ID)
				err := persistToFile(docPath, doc, c.compress, "")
				if err != nil {
					return fmt.Errorf("couldn't persist document to %q: %w", docPath, err)
				}
			}
		}
		db.collections[c.Name] = c
	}

	return nil
}

// Export exports the DB to a file at the given path. The file is encoded as gob,
// optionally compressed with flate (as gzip) and optionally encrypted with AES-GCM.
// This works for both the in-memory and persistent DBs.
// If the file exists, it's overwritten, otherwise created.
//
//   - filePath: If empty, it defaults to "./chromem-go.gob" (+ ".gz" + ".enc")
//   - compress: Optional. Compresses as gzip if true.
//   - encryptionKey: Optional. Encrypts with AES-GCM if provided. Must be 32 bytes
//     long if provided.
//
// Deprecated: Use [DB.ExportToFile] instead.
func (db *DB) Export(filePath string, compress bool, encryptionKey string) error {
	return db.ExportToFile(filePath, compress, encryptionKey)
}

// ExportToFile exports the DB to a file at the given path. The file is encoded as gob,
// optionally compressed with flate (as gzip) and optionally encrypted with AES-GCM.
// This works for both the in-memory and persistent DBs.
// If the file exists, it's overwritten, otherwise created.
//
//   - filePath: If empty, it defaults to "./chromem-go.gob" (+ ".gz" + ".enc")
//   - compress: Optional. Compresses as gzip if true.
//   - encryptionKey: Optional. Encrypts with AES-GCM if provided. Must be 32 bytes
//     long if provided.
//   - collections: Optional. If provided, only the collections with the given names
//     are exported. Non-existing collections are ignored.
//     If not provided, all collections are exported.
func (db *DB) ExportToFile(filePath string, compress bool, encryptionKey string, collections ...string) error {
	if filePath == "" {
		filePath = "./chromem-go.gob"
		if compress {
			filePath += ".gz"
		}
		if encryptionKey != "" {
			filePath += ".enc"
		}
	}
	if encryptionKey != "" {
		// AES 256 requires a 32 byte key
		if len(encryptionKey) != 32 {
			return errors.New("encryption key must be 32 bytes long")
		}
	}

	// Create persistence structs with exported fields so that they can be encoded
	// as gob.
	type persistenceCollection struct {
		Name      string
		Metadata  map[string]string
		Documents map[string]*Document
	}
	persistenceDB := struct {
		Collections map[string]*persistenceCollection
	}{
		Collections: make(map[string]*persistenceCollection, len(db.collections)),
	}

	db.collectionsLock.RLock()
	defer db.collectionsLock.RUnlock()

	for k, v := range db.collections {
		if len(collections) == 0 || slices.Contains(collections, k) {
			persistenceDB.Collections[k] = &persistenceCollection{
				Name:      v.Name,
				Metadata:  v.metadata,
				Documents: v.documents,
			}
		}
	}

	err := persistToFile(filePath, persistenceDB, compress, encryptionKey)
	if err != nil {
		return fmt.Errorf("couldn't export DB: %w", err)
	}

	return nil
}

// ExportToWriter exports the DB to a writer. The stream is encoded as gob,
// optionally compressed with flate (as gzip) and optionally encrypted with AES-GCM.
// This works for both the in-memory and persistent DBs.
// If the writer has to be closed, it's the caller's responsibility.
// This can be used to export DBs to object storage like S3. See
// https://github.com/philippgille/chromem-go/tree/main/examples/s3-export-import
// for an example.
//
//   - writer: An implementation of [io.Writer]
//   - compress: Optional. Compresses as gzip if true.
//   - encryptionKey: Optional. Encrypts with AES-GCM if provided. Must be 32 bytes
//     long if provided.
//   - collections: Optional. If provided, only the collections with the given names
//     are exported. Non-existing collections are ignored.
//     If not provided, all collections are exported.
func (db *DB) ExportToWriter(writer io.Writer, compress bool, encryptionKey string, collections ...string) error {
	if encryptionKey != "" {
		// AES 256 requires a 32 byte key
		if len(encryptionKey) != 32 {
			return errors.New("encryption key must be 32 bytes long")
		}
	}

	// Create persistence structs with exported fields so that they can be encoded
	// as gob.
	type persistenceCollection struct {
		Name      string
		Metadata  map[string]string
		Documents map[string]*Document
	}
	persistenceDB := struct {
		Collections map[string]*persistenceCollection
	}{
		Collections: make(map[string]*persistenceCollection, len(db.collections)),
	}

	db.collectionsLock.RLock()
	defer db.collectionsLock.RUnlock()

	for k, v := range db.collections {
		if len(collections) == 0 || slices.Contains(collections, k) {
			persistenceDB.Collections[k] = &persistenceCollection{
				Name:      v.Name,
				Metadata:  v.metadata,
				Documents: v.documents,
			}
		}
	}

	err := persistToWriter(writer, persistenceDB, compress, encryptionKey)
	if err != nil {
		return fmt.Errorf("couldn't export DB: %w", err)
	}

	return nil
}

// CreateCollection creates a new collection with the given name and metadata.
//
//   - name: The name of the collection to create.
//   - metadata: Optional metadata to associate with the collection.
//   - embeddingFunc: Optional function to use to embed documents.
//     Uses the default embedding function if not provided.
func (db *DB) CreateCollection(name string, metadata map[string]string, embeddingFunc EmbeddingFunc) (*Collection, error) {
	if name == "" {
		return nil, errors.New("collection name is empty")
	}
	if embeddingFunc == nil {
		embeddingFunc = NewEmbeddingFuncDefault()
	}
	collection, err := newCollection(name, metadata, embeddingFunc, db.persistDirectory, db.compress)
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
// To do that use the DB's methods like [DB.CreateCollection] and [DB.DeleteCollection].
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
// The embeddingFunc param is only used if the DB is persistent and was just loaded
// from storage, in which case no embedding func is set yet (funcs are not (de-)serializable).
// It can be nil, in which case the default one will be used.
// The returned collection is a reference to the original collection, so any methods
// on the collection like Add() will be reflected on the DB's collection. Those
// operations are concurrency-safe.
// If the collection doesn't exist, this returns nil.
func (db *DB) GetCollection(name string, embeddingFunc EmbeddingFunc) *Collection {
	db.collectionsLock.RLock()
	defer db.collectionsLock.RUnlock()

	c, ok := db.collections[name]
	if !ok {
		return nil
	}

	if c.embed == nil {
		if embeddingFunc == nil {
			c.embed = NewEmbeddingFuncDefault()
		} else {
			c.embed = embeddingFunc
		}
	}
	return c
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
	collection := db.GetCollection(name, embeddingFunc)
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
// If the DB is persistent, it also removes the collection's directory.
// You shouldn't hold any references to the collection after calling this method.
func (db *DB) DeleteCollection(name string) error {
	db.collectionsLock.Lock()
	defer db.collectionsLock.Unlock()

	col, ok := db.collections[name]
	if !ok {
		return nil
	}

	if db.persistDirectory != "" {
		collectionPath := col.persistDirectory
		err := os.RemoveAll(collectionPath)
		if err != nil {
			return fmt.Errorf("couldn't delete collection directory: %w", err)
		}
	}

	delete(db.collections, name)
	return nil
}

// Reset removes all collections from the DB.
// If the DB is persistent, it also removes all contents of the DB directory.
// You shouldn't hold any references to old collections after calling this method.
func (db *DB) Reset() error {
	db.collectionsLock.Lock()
	defer db.collectionsLock.Unlock()

	if db.persistDirectory != "" {
		err := os.RemoveAll(db.persistDirectory)
		if err != nil {
			return fmt.Errorf("couldn't delete persistence directory: %w", err)
		}
		// Recreate empty root level directory
		err = os.MkdirAll(db.persistDirectory, 0o700)
		if err != nil {
			return fmt.Errorf("couldn't recreate persistence directory: %w", err)
		}
	}

	// Just assign a new map, the GC will take care of the rest.
	db.collections = make(map[string]*Collection)
	return nil
}
