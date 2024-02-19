package chromem

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"slices"
	"sort"
	"sync"
)

// Collection represents a collection of documents.
// It also has a configured embedding function, which is used when adding documents
// that don't have embeddings yet.
type Collection struct {
	Name string

	persistDirectory string
	metadata         map[string]string
	documents        map[string]*document
	documentsLock    sync.RWMutex
	embed            EmbeddingFunc
}

// We don't export this yet to keep the API surface to the bare minimum.
// Users create collections via [Client.CreateCollection].
func newCollection(name string, metadata map[string]string, embed EmbeddingFunc, dir string) (*Collection, error) {
	// We copy the metadata to avoid data races in case the caller modifies the
	// map after creating the collection while we range over it.
	m := make(map[string]string, len(metadata))
	for k, v := range metadata {
		m[k] = v
	}

	c := &Collection{
		Name: name,

		metadata:  m,
		documents: make(map[string]*document),
		embed:     embed,
	}

	// Persistence
	if dir != "" {
		safeName := hash2hex(name)
		c.persistDirectory = path.Join(dir, safeName)

		// Persist the metadata
		err := os.MkdirAll(c.persistDirectory, 0o700)
		if err != nil {
			return nil, fmt.Errorf("couldn't create collection directory: %w", err)
		}
		err = persist(c.persistDirectory, m)
		if err != nil {
			return nil, fmt.Errorf("couldn't persist collection metadata: %w", err)
		}
	}

	return c, nil
}

// Add embeddings to the datastore.
//
//   - ids: The ids of the embeddings you wish to add
//   - embeddings: The embeddings to add. If nil, embeddings will be computed based
//     on the documents using the embeddingFunc set for the Collection. Optional.
//   - metadatas: The metadata to associate with the embeddings. When querying,
//     you can filter on this metadata. Optional.
//   - documents: The documents to associate with the embeddings.
//
// A row-based API will be added when Chroma adds it (they already plan to).
func (c *Collection) Add(ctx context.Context, ids []string, embeddings [][]float32, metadatas []map[string]string, documents []string) error {
	return c.add(ctx, ids, documents, embeddings, metadatas, 1)
}

// AddConcurrently is like Add, but adds embeddings concurrently.
// This is mostly useful when you don't pass any embeddings so they have to be created.
// Upon error, concurrently running operations are canceled and the error is returned.
func (c *Collection) AddConcurrently(ctx context.Context, ids []string, embeddings [][]float32, metadatas []map[string]string, documents []string, concurrency int) error {
	if concurrency < 1 {
		return errors.New("concurrency must be at least 1")
	}
	return c.add(ctx, ids, documents, embeddings, metadatas, concurrency)
}

// Performs a nearest neighbors query on a collection specified by UUID.
//
//   - queryText: The text to search for.
//   - nResults: The number of results to return. Must be > 0.
//   - where: Conditional filtering on metadata. Optional.
//   - whereDocument: Conditional filtering on documents. Optional.
func (c *Collection) Query(ctx context.Context, queryText string, nResults int, where, whereDocument map[string]string) ([]Result, error) {
	c.documentsLock.RLock()
	defer c.documentsLock.RUnlock()
	if len(c.documents) == 0 {
		return nil, nil
	}

	if nResults <= 0 {
		return nil, errors.New("nResults must be > 0")
	}

	// Validate whereDocument operators
	for k := range whereDocument {
		if !slices.Contains(supportedFilters, k) {
			return nil, errors.New("unsupported operator")
		}
	}

	// Filter docs by metadata and content
	filteredDocs := filterDocs(c.documents, where, whereDocument)

	// No need to continue if the filters got rid of all documents
	if len(filteredDocs) == 0 {
		return nil, nil
	}

	queryVectors, err := c.embed(ctx, queryText)
	if err != nil {
		return nil, fmt.Errorf("couldn't create embedding of query: %w", err)
	}

	// For the remaining documents, calculate cosine similarity.
	res, err := calcDocSimilarity(ctx, queryVectors, filteredDocs)
	if err != nil {
		return nil, fmt.Errorf("couldn't calculate cosine similarity: %w", err)
	}

	// Sort by similarity
	sort.Slice(res, func(i, j int) bool {
		// The `less` function would usually use `<`, but we want to sort descending.
		return res[i].Similarity > res[j].Similarity
	})

	// Return the top nResults
	return res[:nResults], nil
}

func (c *Collection) add(ctx context.Context, ids []string, documents []string, embeddings [][]float32, metadatas []map[string]string, concurrency int) error {
	if len(ids) == 0 || len(documents) == 0 {
		return errors.New("ids and documents must not be empty")
	}
	if len(ids) != len(documents) {
		return errors.New("ids and documents must have the same length")
	}
	if len(embeddings) != 0 && len(ids) != len(embeddings) {
		return errors.New("ids, embeddings and documents must have the same length")
	}
	if len(metadatas) != 0 && len(ids) != len(metadatas) {
		return errors.New("ids, metadatas and documents must have the same length")
	}

	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	var wg sync.WaitGroup
	var globalErr error
	var globalErrLock sync.Mutex
	semaphore := make(chan struct{}, concurrency)
	for i, document := range documents {
		var embedding []float32
		var metadata map[string]string
		if len(embeddings) != 0 {
			embedding = embeddings[i]
		}
		if len(metadatas) != 0 {
			metadata = metadatas[i]
		}

		wg.Add(1)
		go func(id string, embedding []float32, metadata map[string]string, document string) {
			defer wg.Done()

			// Don't even start if we already have an error
			if ctx.Err() != nil {
				return
			}

			// Wait here while $concurrency other goroutines are creating documents.
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			err := c.addRow(ctx, id, document, embedding, metadata)
			if err != nil {
				globalErrLock.Lock()
				defer globalErrLock.Unlock()
				// Another goroutine might have already set the error.
				if globalErr == nil {
					globalErr = err
					// Cancel the operation for all other goroutines.
					cancel(globalErr)
				}
				return
			}
		}(ids[i], embedding, metadata, document)
	}

	wg.Wait()

	return globalErr
}

func (c *Collection) addRow(ctx context.Context, id string, document string, embedding []float32, metadata map[string]string) error {
	doc, err := newDocument(ctx, id, embedding, metadata, document, c.embed)
	if err != nil {
		return fmt.Errorf("couldn't create document '%s': %w", id, err)
	}

	c.documentsLock.Lock()
	// We don't defer the unlock because we want to do it earlier.
	c.documents[id] = doc
	c.documentsLock.Unlock()

	// Persist the document
	if c.persistDirectory != "" {
		safeID := hash2hex(id)
		filePath := path.Join(c.persistDirectory, safeID)
		err := persist(filePath, doc)
		if err != nil {
			return fmt.Errorf("couldn't persist document: %w", err)
		}
	}

	return nil
}
