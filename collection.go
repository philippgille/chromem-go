package chromem

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"sync"
)

// Collection represents a collection of documents.
// It also has a configured embedding function, which is used when adding documents
// that don't have embeddings yet.
type Collection struct {
	Name string

	metadata      map[string]string
	documents     map[string]*Document
	documentsLock sync.RWMutex
	embed         EmbeddingFunc

	persistDirectory string
	compress         bool

	// ⚠️ When adding fields here, consider adding them to the persistence struct
	// versions in [DB.Export] and [DB.Import] as well!
}

// NegativeMode represents the mode to use for the negative text.
// See QueryOptions for more information.
type NegativeMode string

const (
	// NEGATIVE_MODE_FILTER filters out results based on the similarity between the
	// negative embedding and the document embeddings.
	// NegativeFilterThreshold controls the threshold for filtering. Documents with
	// similarity above the threshold will be removed from the results.
	NEGATIVE_MODE_FILTER NegativeMode = "filter"

	// NEGATIVE_MODE_SUBTRACT subtracts the negative embedding from the query embedding.
	// This is the default behavior.
	NEGATIVE_MODE_SUBTRACT NegativeMode = "subtract"

	// The default threshold for the negative filter.
	DEFAULT_NEGATIVE_FILTER_THRESHOLD = 0.5
)

// QueryOptions represents the options for a query.
type QueryOptions struct {
	// The text to search for.
	QueryText string

	// The embedding of the query to search for. It must be created
	// with the same embedding model as the document embeddings in the collection.
	// The embedding will be normalized if it's not the case yet.
	// If both QueryText and QueryEmbedding are set, QueryEmbedding will be used.
	QueryEmbedding []float32

	// The number of results to return.
	NResults int

	// Conditional filtering on metadata.
	Where map[string]string

	// Conditional filtering on documents.
	WhereDocument map[string]string

	// Negative is the negative query options.
	// They can be used to exclude certain results from the query.
	Negative NegativeQueryOptions
}

type NegativeQueryOptions struct {
	// Mode is the mode to use for the negative text.
	Mode NegativeMode

	// Text is the text to exclude from the results.
	Text string

	// Embedding is the embedding of the negative text. It must be created
	// with the same embedding model as the document embeddings in the collection.
	// The embedding will be normalized if it's not the case yet.
	// If both Text and Embedding are set, Embedding will be used.
	Embedding []float32

	// FilterThreshold is the threshold for the negative filter. Used when Mode is NEGATIVE_MODE_FILTER.
	FilterThreshold float32
}

// We don't export this yet to keep the API surface to the bare minimum.
// Users create collections via [Client.CreateCollection].
func newCollection(name string, metadata map[string]string, embed EmbeddingFunc, dbDir string, compress bool) (*Collection, error) {
	// We copy the metadata to avoid data races in case the caller modifies the
	// map after creating the collection while we range over it.
	m := make(map[string]string, len(metadata))
	for k, v := range metadata {
		m[k] = v
	}

	c := &Collection{
		Name: name,

		metadata:  m,
		documents: make(map[string]*Document),
		embed:     embed,
	}

	// Persistence
	if dbDir != "" {
		safeName := hash2hex(name)
		c.persistDirectory = filepath.Join(dbDir, safeName)
		c.compress = compress
		return c, c.persistMetadata()
	}

	return c, nil
}

// Add embeddings to the datastore.
//
//   - ids: The ids of the embeddings you wish to add
//   - embeddings: The embeddings to add. If nil, embeddings will be computed based
//     on the contents using the embeddingFunc set for the Collection. Optional.
//   - metadatas: The metadata to associate with the embeddings. When querying,
//     you can filter on this metadata. Optional.
//   - contents: The contents to associate with the embeddings.
//
// This is a Chroma-like method. For a more Go-idiomatic one, see [Collection.AddDocuments].
func (c *Collection) Add(ctx context.Context, ids []string, embeddings [][]float32, metadatas []map[string]string, contents []string) error {
	return c.AddConcurrently(ctx, ids, embeddings, metadatas, contents, 1)
}

// AddConcurrently is like Add, but adds embeddings concurrently.
// This is mostly useful when you don't pass any embeddings, so they have to be created.
// Upon error, concurrently running operations are canceled and the error is returned.
//
// This is a Chroma-like method. For a more Go-idiomatic one, see [Collection.AddDocuments].
func (c *Collection) AddConcurrently(ctx context.Context, ids []string, embeddings [][]float32, metadatas []map[string]string, contents []string, concurrency int) error {
	if len(ids) == 0 {
		return errors.New("ids are empty")
	}
	if len(embeddings) == 0 && len(contents) == 0 {
		return errors.New("either embeddings or contents must be filled")
	}
	if len(embeddings) != 0 {
		if len(embeddings) != len(ids) {
			return errors.New("ids and embeddings must have the same length")
		}
	} else {
		// Assign empty slice, so we can simply access via index later
		embeddings = make([][]float32, len(ids))
	}
	if len(metadatas) != 0 {
		if len(ids) != len(metadatas) {
			return errors.New("when metadatas is not empty it must have the same length as ids")
		}
	} else {
		// Assign empty slice, so we can simply access via index later
		metadatas = make([]map[string]string, len(ids))
	}
	if len(contents) != 0 {
		if len(contents) != len(ids) {
			return errors.New("ids and contents must have the same length")
		}
	} else {
		// Assign empty slice, so we can simply access via index later
		contents = make([]string, len(ids))
	}
	if concurrency < 1 {
		return errors.New("concurrency must be at least 1")
	}

	// Convert Chroma-style parameters into a slice of documents.
	docs := make([]Document, 0, len(ids))
	for i, id := range ids {
		docs = append(docs, Document{
			ID:        id,
			Metadata:  metadatas[i],
			Embedding: embeddings[i],
			Content:   contents[i],
		})
	}

	return c.AddDocuments(ctx, docs, concurrency)
}

// AddDocuments adds documents to the collection with the specified concurrency.
// If the documents don't have embeddings, they will be created using the collection's
// embedding function.
// Upon error, concurrently running operations are canceled and the error is returned.
func (c *Collection) AddDocuments(ctx context.Context, documents []Document, concurrency int) error {
	if len(documents) == 0 {
		// TODO: Should this be a no-op instead?
		return errors.New("documents slice is nil or empty")
	}
	if concurrency < 1 {
		return errors.New("concurrency must be at least 1")
	}
	// For other validations we rely on AddDocument.

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

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, concurrency)
	for _, doc := range documents {
		wg.Add(1)
		go func(doc Document) {
			defer wg.Done()

			// Don't even start if another goroutine already failed.
			if ctx.Err() != nil {
				return
			}

			// Wait here while $concurrency other goroutines are creating documents.
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			err := c.AddDocument(ctx, doc)
			if err != nil {
				setSharedErr(fmt.Errorf("couldn't add document '%s': %w", doc.ID, err))
				return
			}
		}(doc)
	}

	wg.Wait()

	return sharedErr
}

// AddDocument adds a document to the collection.
// If the document doesn't have an embedding, it will be created using the collection's
// embedding function.
func (c *Collection) AddDocument(ctx context.Context, doc Document) error {
	if doc.ID == "" {
		return errors.New("document ID is empty")
	}
	if len(doc.Embedding) == 0 && doc.Content == "" {
		return errors.New("either document embedding or content must be filled")
	}

	// We copy the metadata to avoid data races in case the caller modifies the
	// map after creating the document while we range over it.
	m := make(map[string]string, len(doc.Metadata))
	for k, v := range doc.Metadata {
		m[k] = v
	}

	// Create embedding if they don't exist, otherwise normalize if necessary
	if len(doc.Embedding) == 0 {
		embedding, err := c.embed(ctx, doc.Content)
		if err != nil {
			return fmt.Errorf("couldn't create embedding of document: %w", err)
		}
		doc.Embedding = embedding
	} else {
		if !isNormalized(doc.Embedding) {
			doc.Embedding = normalizeVector(doc.Embedding)
		}
	}

	c.documentsLock.Lock()
	// We don't defer the unlock because we want to do it earlier.
	c.documents[doc.ID] = &doc
	c.documentsLock.Unlock()

	// Persist the document
	if c.persistDirectory != "" {
		docPath := c.getDocPath(doc.ID)
		err := persistToFile(docPath, doc, c.compress, "")
		if err != nil {
			return fmt.Errorf("couldn't persist document to %q: %w", docPath, err)
		}
	}

	return nil
}

// ListIDs returns the IDs of all documents in the collection.
func (c *Collection) ListIDs(_ context.Context) []string {
	c.documentsLock.RLock()
	defer c.documentsLock.RUnlock()

	ids := make([]string, 0, len(c.documents))
	for id := range c.documents {
		ids = append(ids, id)
	}

	return ids
}

// GetByID returns a document by its ID.
// The returned document is a copy of the original document, so it can be safely
// modified without affecting the collection.
func (c *Collection) GetByID(ctx context.Context, id string) (Document, error) {
	if id == "" {
		return Document{}, errors.New("document ID is empty")
	}

	c.documentsLock.RLock()
	defer c.documentsLock.RUnlock()

	doc, ok := c.documents[id]
	if ok {
		// Clone the document
		res := *doc
		// Above copies the simple fields, but we need to copy the slices and maps
		res.Metadata = maps.Clone(doc.Metadata)
		res.Embedding = slices.Clone(doc.Embedding)

		return res, nil
	}

	return Document{}, fmt.Errorf("document with ID '%v' not found", id)
}

// Delete removes document(s) from the collection.
//
//   - where: Conditional filtering on metadata. Optional.
//   - whereDocument: Conditional filtering on documents. Optional.
//   - ids: The ids of the documents to delete. If empty, all documents are deleted.
func (c *Collection) Delete(_ context.Context, where, whereDocument map[string]string, ids ...string) error {
	// must have at least one of where, whereDocument or ids
	if len(where) == 0 && len(whereDocument) == 0 && len(ids) == 0 {
		return fmt.Errorf("must have at least one of where, whereDocument or ids")
	}

	if len(c.documents) == 0 {
		return nil
	}

	for k := range whereDocument {
		if !slices.Contains(supportedFilters, k) {
			return errors.New("unsupported whereDocument operator")
		}
	}

	var docIDs []string

	c.documentsLock.Lock()
	defer c.documentsLock.Unlock()

	if where != nil || whereDocument != nil {
		// metadata + content filters
		filteredDocs := filterDocs(c.documents, where, whereDocument)
		for _, doc := range filteredDocs {
			docIDs = append(docIDs, doc.ID)
		}
	} else {
		docIDs = ids
	}

	// No-op if no docs are left
	if len(docIDs) == 0 {
		return nil
	}

	for _, docID := range docIDs {
		delete(c.documents, docID)

		// Remove the document from disk
		if c.persistDirectory != "" {
			docPath := c.getDocPath(docID)
			err := removeFile(docPath)
			if err != nil {
				return fmt.Errorf("couldn't remove document at %q: %w", docPath, err)
			}
		}
	}

	return nil
}

// Count returns the number of documents in the collection.
func (c *Collection) Count() int {
	c.documentsLock.RLock()
	defer c.documentsLock.RUnlock()
	return len(c.documents)
}

// Result represents a single result from a query.
type Result struct {
	ID        string
	Metadata  map[string]string
	Embedding []float32
	Content   string

	// The cosine similarity between the query and the document.
	// The higher the value, the more similar the document is to the query.
	// The value is in the range [-1, 1].
	Similarity float32
}

// Query performs an exhaustive nearest neighbor search on the collection.
//
//   - queryText: The text to search for. Its embedding will be created using the
//     collection's embedding function.
//   - nResults: The maximum number of results to return. Must be > 0.
//     There can be fewer results if a filter is applied.
//   - where: Conditional filtering on metadata. Optional.
//   - whereDocument: Conditional filtering on documents. Optional.
func (c *Collection) Query(ctx context.Context, queryText string, nResults int, where, whereDocument map[string]string) ([]Result, error) {
	if queryText == "" {
		return nil, errors.New("queryText is empty")
	}

	queryVector, err := c.embed(ctx, queryText)
	if err != nil {
		return nil, fmt.Errorf("couldn't create embedding of query: %w", err)
	}

	return c.QueryEmbedding(ctx, queryVector, nResults, where, whereDocument)
}

// QueryWithOptions performs an exhaustive nearest neighbor search on the collection.
//
//   - options: The options for the query. See [QueryOptions] for more information.
func (c *Collection) QueryWithOptions(ctx context.Context, options QueryOptions) ([]Result, error) {
	if options.QueryText == "" && len(options.QueryEmbedding) == 0 {
		return nil, errors.New("QueryText and QueryEmbedding options are empty")
	}

	var err error
	queryVector := options.QueryEmbedding
	if len(queryVector) == 0 {
		queryVector, err = c.embed(ctx, options.QueryText)
		if err != nil {
			return nil, fmt.Errorf("couldn't create embedding of query: %w", err)
		}
	}

	negativeFilterThreshold := options.Negative.FilterThreshold
	negativeVector := options.Negative.Embedding
	if len(negativeVector) == 0 && options.Negative.Text != "" {
		negativeVector, err = c.embed(ctx, options.Negative.Text)
		if err != nil {
			return nil, fmt.Errorf("couldn't create embedding of negative: %w", err)
		}
	}

	if len(negativeVector) != 0 {
		if !isNormalized(negativeVector) {
			negativeVector = normalizeVector(negativeVector)
		}

		if options.Negative.Mode == NEGATIVE_MODE_SUBTRACT {
			queryVector = subtractVector(queryVector, negativeVector)
			queryVector = normalizeVector(queryVector)
		} else if options.Negative.Mode == NEGATIVE_MODE_FILTER {
			if negativeFilterThreshold == 0 {
				negativeFilterThreshold = DEFAULT_NEGATIVE_FILTER_THRESHOLD
			}
		} else {
			return nil, fmt.Errorf("unsupported negative mode: %q", options.Negative.Mode)
		}
	}

	result, err := c.queryEmbedding(ctx, queryVector, negativeVector, negativeFilterThreshold, options.NResults, options.Where, options.WhereDocument)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// QueryEmbedding performs an exhaustive nearest neighbor search on the collection.
//
//   - queryEmbedding: The embedding of the query to search for. It must be created
//     with the same embedding model as the document embeddings in the collection.
//     The embedding will be normalized if it's not the case yet.
//   - nResults: The maximum number of results to return. Must be > 0.
//     There can be fewer results if a filter is applied.
//   - where: Conditional filtering on metadata. Optional.
//   - whereDocument: Conditional filtering on documents. Optional.
func (c *Collection) QueryEmbedding(ctx context.Context, queryEmbedding []float32, nResults int, where, whereDocument map[string]string) ([]Result, error) {
	return c.queryEmbedding(ctx, queryEmbedding, nil, 0, nResults, where, whereDocument)
}

// queryEmbedding performs an exhaustive nearest neighbor search on the collection.
func (c *Collection) queryEmbedding(ctx context.Context, queryEmbedding, negativeEmbeddings []float32, negativeFilterThreshold float32, nResults int, where, whereDocument map[string]string) ([]Result, error) {
	if len(queryEmbedding) == 0 {
		return nil, errors.New("queryEmbedding is empty")
	}
	if nResults <= 0 {
		return nil, errors.New("nResults must be > 0")
	}
	c.documentsLock.RLock()
	defer c.documentsLock.RUnlock()
	if nResults > len(c.documents) {
		return nil, errors.New("nResults must be <= the number of documents in the collection")
	}

	if len(c.documents) == 0 {
		return nil, nil
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

	// Normalize embedding if not the case yet. We only support cosine similarity
	// for now and all documents were already normalized when added to the collection.
	if !isNormalized(queryEmbedding) {
		queryEmbedding = normalizeVector(queryEmbedding)
	}

	// If the filtering already reduced the number of documents to fewer than nResults,
	// we only need to find the most similar docs among the filtered ones.
	resLen := nResults
	if len(filteredDocs) < nResults {
		resLen = len(filteredDocs)
	}

	// For the remaining documents, get the most similar docs.
	nMaxDocs, err := getMostSimilarDocs(ctx, queryEmbedding, negativeEmbeddings, negativeFilterThreshold, filteredDocs, resLen)
	if err != nil {
		return nil, fmt.Errorf("couldn't get most similar docs: %w", err)
	}

	res := make([]Result, 0, len(nMaxDocs))
	for i := 0; i < len(nMaxDocs); i++ {
		res = append(res, Result{
			ID:         nMaxDocs[i].docID,
			Metadata:   c.documents[nMaxDocs[i].docID].Metadata,
			Embedding:  c.documents[nMaxDocs[i].docID].Embedding,
			Content:    c.documents[nMaxDocs[i].docID].Content,
			Similarity: nMaxDocs[i].similarity,
		})
	}

	return res, nil
}

// getDocPath generates the path to the document file.
func (c *Collection) getDocPath(docID string) string {
	safeID := hash2hex(docID)
	docPath := filepath.Join(c.persistDirectory, safeID)
	docPath += ".gob"
	if c.compress {
		docPath += ".gz"
	}
	return docPath
}

// persistMetadata persists the collection metadata to disk
func (c *Collection) persistMetadata() error {
	// Persist name and metadata
	metadataPath := filepath.Join(c.persistDirectory, metadataFileName)
	metadataPath += ".gob"
	if c.compress {
		metadataPath += ".gz"
	}
	pc := struct {
		Name     string
		Metadata map[string]string
	}{
		Name:     c.Name,
		Metadata: c.metadata,
	}
	err := persistToFile(metadataPath, pc, c.compress, "")
	if err != nil {
		return err
	}

	return nil
}
