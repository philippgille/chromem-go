package chromem

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

type EmbeddingModelCohere string

const (
	EmbeddingModelCohereMultilingualV2 EmbeddingModelCohere = "embed-multilingual-v2.0"
	EmbeddingModelCohereEnglishLightV2 EmbeddingModelCohere = "embed-english-light-v2.0"
	EmbeddingModelCohereEnglishV2      EmbeddingModelCohere = "embed-english-v2.0"

	EmbeddingModelCohereMultilingualLightV3 EmbeddingModelCohere = "embed-multilingual-light-v3.0"
	EmbeddingModelCohereEnglishLightV3      EmbeddingModelCohere = "embed-english-light-v3.0"
	EmbeddingModelCohereMultilingualV3      EmbeddingModelCohere = "embed-multilingual-v3.0"
	EmbeddingModelCohereEnglishV3           EmbeddingModelCohere = "embed-english-v3.0"
)

// Prefixes for external use.
const (
	InputTypeCohereSearchDocumentPrefix string = "search_document: "
	InputTypeCohereSearchQueryPrefix    string = "search_query: "
	InputTypeCohereClassificationPrefix string = "classification: "
	InputTypeCohereClusteringPrefix     string = "clustering: "
)

// Input types for internal use.
const (
	inputTypeCohereSearchDocument string = "search_document"
	inputTypeCohereSearchQuery    string = "search_query"
	inputTypeCohereClassification string = "classification"
	inputTypeCohereClustering     string = "clustering"
)

const baseURLCohere = "https://api.cohere.ai/v1"

var validInputTypesCohere = map[string]string{
	inputTypeCohereSearchDocument: InputTypeCohereSearchDocumentPrefix,
	inputTypeCohereSearchQuery:    InputTypeCohereSearchQueryPrefix,
	inputTypeCohereClassification: InputTypeCohereClassificationPrefix,
	inputTypeCohereClustering:     InputTypeCohereClusteringPrefix,
}

type cohereResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

// NewEmbeddingFuncCohere returns a function that creates embeddings for a text
// using Cohere's API. One important difference to OpenAI's and other's APIs is
// that Cohere differentiates between document embeddings and search/query embeddings.
// In order for this embedding func to do the differentiation, you have to prepend
// the text with either "search_document" or "search_query". We'll cut off that
// prefix before sending the document/query body to the API, we'll just use the
// prefix to choose the right "input type" as they call it.
//
// When you set up a chromem-go collection with this embedding function, you might
// want to create the document separately with [NewDocument] and then cut off the
// prefix before adding the document to the collection. Otherwise, when you query
// the collection, the returned documents will still have the prefix in their content.
//
//	cohereFunc := chromem.NewEmbeddingFuncCohere(cohereApiKey, chromem.EmbeddingModelCohereEnglishV3)
//	content := "The sky is blue because of Rayleigh scattering."
//	// Create the document with the prefix.
//	contentWithPrefix := chromem.InputTypeCohereSearchDocumentPrefix + content
//	doc, _ := NewDocument(ctx, id, metadata, nil, contentWithPrefix, cohereFunc)
//	// Remove the prefix so that later query results don't have it.
//	doc.Content = content
//	_ = collection.AddDocument(ctx, doc)
//
// This is not necessary if you don't keep the content in the documents, as chromem-go
// also works when documents only have embeddings.
// You can also keep the prefix in the document, and only remove it after querying.
//
// We plan to improve this in the future.
func NewEmbeddingFuncCohere(apiKey string, model EmbeddingModelCohere) EmbeddingFunc {
	// We don't set a default timeout here, although it's usually a good idea.
	// In our case though, the library user can set the timeout on the context,
	// and it might have to be a long timeout, depending on the text length.
	client := &http.Client{}

	var checkedNormalized bool
	checkNormalized := sync.Once{}

	return func(ctx context.Context, text string) ([]float32, error) {
		var inputType string
		for validInputType, validInputTypePrefix := range validInputTypesCohere {
			if strings.HasPrefix(text, validInputTypePrefix) {
				inputType = validInputType
				text = strings.TrimPrefix(text, validInputTypePrefix)
				break
			}
		}
		if inputType == "" {
			return nil, errors.New("text must start with a valid input type plus colon and space")
		}

		// Prepare the request body.
		reqBody, err := json.Marshal(map[string]any{
			"model":      model,
			"texts":      []string{text},
			"input_type": inputType,
		})
		if err != nil {
			return nil, fmt.Errorf("couldn't marshal request body: %w", err)
		}

		// Create the request. Creating it with context is important for a timeout
		// to be possible, because the client is configured without a timeout.
		req, err := http.NewRequestWithContext(ctx, "POST", baseURLCohere+"/embed", bytes.NewBuffer(reqBody))
		if err != nil {
			return nil, fmt.Errorf("couldn't create request: %w", err)
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		// Send the request.
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("couldn't send request: %w", err)
		}
		defer resp.Body.Close()

		// Check the response status.
		if resp.StatusCode != http.StatusOK {
			return nil, errors.New("error response from the embedding API: " + resp.Status)
		}

		// Read and decode the response body.
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("couldn't read response body: %w", err)
		}
		var embeddingResponse cohereResponse
		err = json.Unmarshal(body, &embeddingResponse)
		if err != nil {
			return nil, fmt.Errorf("couldn't unmarshal response body: %w", err)
		}

		// Check if the response contains embeddings.
		if len(embeddingResponse.Embeddings) == 0 || len(embeddingResponse.Embeddings[0]) == 0 {
			return nil, errors.New("no embeddings found in the response")
		}

		v := embeddingResponse.Embeddings[0]
		checkNormalized.Do(func() {
			if isNormalized(v) {
				checkedNormalized = true
			} else {
				checkedNormalized = false
			}
		})
		if !checkedNormalized {
			v = normalizeVector(v)
		}

		return v, nil
	}
}
