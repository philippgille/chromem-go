package chromem

import (
	"context"
	"runtime"
	"strings"
	"sync"
)

var supportedFilters = []string{"$contains", "$not_contains"}

// Result represents a single result from a query.
type Result struct {
	ID        string
	Embedding []float32
	Metadata  map[string]string
	Document  string

	// The cosine similarity between the query and the document.
	// The higher the value, the more similar the document is to the query.
	// The value is in the range [-1, 1].
	Similarity float32
}

// filterDocs filters a map of documents by metadata and content.
// It does this concurrently.
func filterDocs(docs map[string]*document, where, whereDocument map[string]string) []*document {
	filteredDocs := make([]*document, 0, len(docs))
	filteredDocsLock := sync.Mutex{}

	// Determine concurrency. Use number of docs or CPUs, whichever is smaller.
	numCPUs := runtime.NumCPU()
	numDocs := len(docs)
	concurrency := numCPUs
	if numDocs < numCPUs {
		concurrency = numDocs
	}

	docChan := make(chan *document, concurrency*2)

	wg := sync.WaitGroup{}
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for doc := range docChan {
				if documentMatchesFilters(doc, where, whereDocument) {
					filteredDocsLock.Lock()
					filteredDocs = append(filteredDocs, doc)
					filteredDocsLock.Unlock()
				}
			}
		}()
	}

	for _, doc := range docs {
		docChan <- doc
	}
	close(docChan)

	wg.Wait()

	return filteredDocs
}

// documentMatchesFilters checks if a document matches the given filters.
// When calling this function, the whereDocument keys must already be validated!
func documentMatchesFilters(document *document, where, whereDocument map[string]string) bool {
	// A document's metadata must have *all* the fields in the where clause.
	for k, v := range where {
		// TODO: Do we want to check for existence of the key? I.e. should
		// a where clause with empty string as value match a document's
		// metadata that doesn't have the key at all?
		if document.Metadata[k] != v {
			return false
		}
	}

	// A document must satisfy *all* filters, until we support the `$or` operator.
	for k, v := range whereDocument {
		switch k {
		case "$contains":
			if !strings.Contains(document.Document, v) {
				return false
			}
		case "$not_contains":
			if strings.Contains(document.Document, v) {
				return false
			}
		default:
			// No handling (error) required because we already validated the
			// operators. This simplifies the concurrency logic (no err var
			// and lock, no context to cancel).
		}
	}

	return true
}

func calcDocSimilarity(ctx context.Context, queryVectors []float32, docs []*document) ([]Result, error) {
	res := make([]Result, len(docs))
	resLock := sync.Mutex{}

	// Determine concurrency. Use number of docs or CPUs, whichever is smaller.
	numCPUs := runtime.NumCPU()
	numDocs := len(docs)
	concurrency := numCPUs
	if numDocs < numCPUs {
		concurrency = numDocs
	}

	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	docChan := make(chan *document, concurrency*2)
	var globalErr error
	globalErrLock := sync.Mutex{}

	wg := sync.WaitGroup{}
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for doc := range docChan {
				// Stop work if another goroutine encountered an error.
				if ctx.Err() != nil {
					return
				}

				sim, err := cosineSimilarity(queryVectors, doc.Vectors)
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

				resLock.Lock()
				// We don't defer the unlock because we want to unlock much earlier.
				res = append(res, Result{
					ID:        doc.ID,
					Embedding: doc.Vectors,
					Metadata:  doc.Metadata,
					Document:  doc.Document,

					Similarity: sim,
				})
				resLock.Unlock()
			}
		}()
	}

OuterLoop:
	for _, doc := range docs {
		// The doc channel has limited capacity, so writing to the channel blocks
		// when a goroutine runs into an error and then all goroutines stop processing
		// the channel and it gets full.
		// To avoid a deadlock we check for ctx.Done() here, which is closed by
		// the goroutine that encountered the error.
		select {
		case docChan <- doc:
		case <-ctx.Done():
			break OuterLoop
		}
	}
	close(docChan)

	wg.Wait()

	if globalErr != nil {
		return nil, globalErr
	}

	return res, nil
}
