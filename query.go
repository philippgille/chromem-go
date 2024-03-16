package chromem

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
)

var supportedFilters = []string{"$contains", "$not_contains"}

type docSim struct {
	docID      string
	similarity float32
}

// filterDocs filters a map of documents by metadata and content.
// It does this concurrently.
func filterDocs(docs map[string]*Document, where, whereDocument map[string]string) []*Document {
	filteredDocs := make([]*Document, 0, len(docs))
	filteredDocsLock := sync.Mutex{}

	// Determine concurrency. Use number of docs or CPUs, whichever is smaller.
	numCPUs := runtime.NumCPU()
	numDocs := len(docs)
	concurrency := numCPUs
	if numDocs < numCPUs {
		concurrency = numDocs
	}

	docChan := make(chan *Document, concurrency*2)

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

	// With filteredDocs being initialized as potentially large slice, let's return
	// nil instead of the empty slice.
	if len(filteredDocs) == 0 {
		filteredDocs = nil
	}
	return filteredDocs
}

// documentMatchesFilters checks if a document matches the given filters.
// When calling this function, the whereDocument keys must already be validated!
func documentMatchesFilters(document *Document, where, whereDocument map[string]string) bool {
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
			if !strings.Contains(document.Content, v) {
				return false
			}
		case "$not_contains":
			if strings.Contains(document.Content, v) {
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

func calcDocSimilarity(ctx context.Context, queryVectors []float32, docs []*Document) ([]docSim, error) {
	similarities := make([]docSim, 0, len(docs))
	similaritiesLock := sync.Mutex{}

	// Determine concurrency. Use number of docs or CPUs, whichever is smaller.
	numCPUs := runtime.NumCPU()
	numDocs := len(docs)
	concurrency := numCPUs
	if numDocs < numCPUs {
		concurrency = numDocs
	}

	var sharedErr error
	sharedErrLock := sync.Mutex{}
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	setSharedErr := func(err error) {
		sharedErrLock.Lock()
		defer sharedErrLock.Unlock()
		// Another goroutine might have already set the error.
		if sharedErr == nil {
			sharedErr = err
			// Cancel the operation for all other goroutines.
			cancel(sharedErr)
		}
	}

	wg := sync.WaitGroup{}
	// Instead of using a channel to pass documents into the goroutines, we just
	// split the slice into sub-slices and pass those to the goroutines.
	// This turned out to be faster in the query benchmarks.
	subSliceSize := len(docs) / concurrency // Can leave remainder, e.g. 10/3 = 3; leaves 1
	rem := len(docs) % concurrency
	for i := 0; i < concurrency; i++ {
		start := i * subSliceSize
		end := start + subSliceSize
		// Add remainder to last goroutine
		if i == concurrency-1 {
			end += rem
		}

		wg.Add(1)
		go func(subSlice []*Document) {
			defer wg.Done()
			for _, doc := range subSlice {
				// Stop work if another goroutine encountered an error.
				if ctx.Err() != nil {
					return
				}

				// As the vectors are normalized, the dot product is the cosine similarity.
				sim, err := dotProduct(queryVectors, doc.Embedding)
				if err != nil {
					setSharedErr(fmt.Errorf("couldn't calculate similarity for document '%s': %w", doc.ID, err))
					return
				}

				similaritiesLock.Lock()
				// We don't defer the unlock because we want to unlock much earlier.
				similarities = append(similarities, docSim{docID: doc.ID, similarity: sim})
				similaritiesLock.Unlock()
			}
		}(docs[start:end])
	}

	wg.Wait()

	if sharedErr != nil {
		return nil, sharedErr
	}

	return similarities, nil
}
