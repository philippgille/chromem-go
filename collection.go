package chromem

import (
	"context"
	"errors"
	"sync"
)

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

	var embedding []float32
	var metadata map[string]string
	var err error
	c.documentsLock.Lock()
	defer c.documentsLock.Unlock()
	for i, document := range documents {
		if len(embeddings) != 0 {
			embedding = embeddings[i]
		}
		if len(metadatas) != 0 {
			metadata = metadatas[i]
		}
		c.documents[ids[i]], err = newDocument(ctx, ids[i], embedding, metadata, document, c.embed)
		if err != nil {
			return err
		}
	}
	return nil
}
