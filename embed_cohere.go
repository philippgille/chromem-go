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
	EmbeddingModelCohereMultilingualV2      EmbeddingModelCohere = "embed-multilingual-v2.0"
	EmbeddingModelCohereEnglishLightV2      EmbeddingModelCohere = "embed-english-light-v2.0"
	EmbeddingModelCohereEnglishV2           EmbeddingModelCohere = "embed-english-v2.0"
	EmbeddingModelCohereMultilingualLightV3 EmbeddingModelCohere = "embed-multilingual-light-v3.0"
	EmbeddingModelCohereEnglishLightV3      EmbeddingModelCohere = "embed-english-light-v3.0"
	EmbeddingModelCohereMultilingualV3      EmbeddingModelCohere = "embed-multilingual-v3.0"
	EmbeddingModelCohereEnglishV3           EmbeddingModelCohere = "embed-english-v3.0"
)

type InputTypeCohere string

const (
	InputTypeSearchDocumentCohere InputTypeCohere = "search_document"
	InputTypeSearchQueryCohere    InputTypeCohere = "search_query"
	InputTypeClassificationCohere InputTypeCohere = "classification"
	InputTypeClusteringCohere     InputTypeCohere = "clustering"
)

const baseURLCohere = "https://api.cohere.ai/v1"

var validInputTypesCohere = []string{
	string(InputTypeSearchDocumentCohere),
	string(InputTypeSearchQueryCohere),
	string(InputTypeClassificationCohere),
	string(InputTypeClusteringCohere),
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
func NewEmbeddingFuncCohere(apiKey string, model EmbeddingModelCohere) EmbeddingFunc {
	// We don't set a default timeout here, although it's usually a good idea.
	// In our case though, the library user can set the timeout on the context,
	// and it might have to be a long timeout, depending on the text length.
	client := &http.Client{}

	var checkedNormalized bool
	checkNormalized := sync.Once{}

	return func(ctx context.Context, text string) ([]float32, error) {
		var inputType string
		for _, validInputType := range validInputTypesCohere {
			if strings.HasPrefix(text, validInputType+": ") {
				inputType = validInputType
				text = strings.TrimPrefix(text, validInputType+": ")
				break
			}
		}
		if inputType == "" {
			return nil, errors.New("text must start with a valid input type")
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
