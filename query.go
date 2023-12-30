package chromem

import (
	"context"
	"errors"
	"slices"
	"sort"
	"strings"
)

var supportedFilters = []string{"$contains", "$not_contains"}

type Result struct {
	ID        string
	Embedding []float32
	Metadata  map[string]string
	Document  string

	Similarity float32
}

func (c *Collection) Query(ctx context.Context, queryText string, nResults int, where, whereDocument map[string]string) ([]Result, error) {
	if nResults == 0 {
		return nil, errors.New("nResults must be > 0")
	}
	// Validate whereDocument operators
	for k := range whereDocument {
		if !slices.Contains(supportedFilters, k) {
			return nil, errors.New("unsupported operator")
		}
	}

	// Filter docs by metadata and content
	var docs []*document
	c.documentsLock.RLock()
	// We don't defer the unlock because we want to unlock much earlier
OuterLoop:
	for _, document := range c.documents {
		// A document's metadata must have *all* the fields in the where clause.
		for k, v := range where {
			// TODO: Do we want to check for existence of the key? I.e. should
			// a where clause with empty string as value match a document's
			// metadata that doesn't have the key at all?
			if document.Metadata[k] != v {
				continue OuterLoop
			}
		}
		// A document must satisfy *all* filters, until we support the `$or` operator.
		for k, v := range whereDocument {
			switch k {
			case "$contains":
				if !strings.Contains(document.Document, v) {
					continue OuterLoop
				}
			case "$not_contains":
				if strings.Contains(document.Document, v) {
					continue OuterLoop
				}
			default:
				return nil, errors.New("unsupported filter: " + k)
			}
		}
		docs = append(docs, document)
	}
	c.documentsLock.RUnlock()

	// No need to continue if the filters got rid of all documents
	if len(docs) == 0 {
		return nil, nil
	}

	queryVectors, err := c.embed(ctx, queryText)
	if err != nil {
		return nil, err
	}

	// For the remaining documents, calculate cosine similarity.
	res := make([]Result, len(docs))
	for i, document := range docs {
		sim, err := cosineSimilarity(queryVectors, document.Vectors)
		if err != nil {
			return nil, err
		}
		res[i] = Result{
			ID:        document.ID,
			Embedding: document.Vectors,
			Metadata:  document.Metadata,
			Document:  document.Document,

			Similarity: sim,
		}
	}

	// Sort by similarity
	sort.Slice(res, func(i, j int) bool {
		// The `less` function would usually use `<`, but we want to sort descending.
		return res[i].Similarity > res[j].Similarity
	})

	// Return the top nResults
	return res[:nResults], nil
}
