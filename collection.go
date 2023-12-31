package chromem

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Collection represents a collection of documents.
// It also has a configured embedding function, which is used when adding documents
// that don't have embeddings yet.
type Collection struct {
	Name     string
	Metadata map[string]string

	documents     map[string]*document
	documentsLock sync.RWMutex

	embed EmbeddingFunc
}

// We don't export this yet to keep the API surface to the bare minimum.
// Users create collections via [Client.CreateCollection].
func newCollection(name string, metadata map[string]string, embed EmbeddingFunc) *Collection {
	return &Collection{
		Name:     name,
		Metadata: metadata,

		documents: make(map[string]*document),

		embed: embed,
	}
}

// Add embeddings to the datastore.
//
//   - ids: The ids of the embeddings you wish to add
//   - embeddings: The embeddings to add. If nil, embeddings will be computed based on the documents using the embeddingFunc set for the Collection. Optional.
//   - metadatas: The metadata to associate with the embeddings. When querying, you can filter on this metadata. Optional.
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

	var wg sync.WaitGroup
	var globalErr error
	var globalErrLock sync.RWMutex
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
			globalErrLock.RLock()
			// We don't defer the unlock because we want to unlock much earlier.
			if globalErr != nil {
				globalErrLock.RUnlock()
				return
			}
			globalErrLock.RUnlock()

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
	defer c.documentsLock.Unlock()
	c.documents[id] = doc

	return nil
}
