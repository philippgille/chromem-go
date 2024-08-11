package chromem

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
)

const BaseURLOpenAI = "https://api.openai.com/v1"

type EmbeddingModelOpenAI string

const (
	EmbeddingModelOpenAI2Ada EmbeddingModelOpenAI = "text-embedding-ada-002"

	EmbeddingModelOpenAI3Small EmbeddingModelOpenAI = "text-embedding-3-small"
	EmbeddingModelOpenAI3Large EmbeddingModelOpenAI = "text-embedding-3-large"
)

type openAIResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

// NewEmbeddingFuncDefault returns a function that creates embeddings for a text
// using OpenAI`s "text-embedding-3-small" model via their API.
// The model supports a maximum text length of 8191 tokens.
// The API key is read from the environment variable "OPENAI_API_KEY".
func NewEmbeddingFuncDefault() EmbeddingFunc {
	apiKey := os.Getenv("OPENAI_API_KEY")
	return NewEmbeddingFuncOpenAI(apiKey, EmbeddingModelOpenAI3Small)
}

// NewEmbeddingFuncOpenAI returns a function that creates embeddings for a text
// using the OpenAI API.
func NewEmbeddingFuncOpenAI(apiKey string, model EmbeddingModelOpenAI) EmbeddingFunc {
	// OpenAI embeddings are normalized
	normalized := true
	return NewEmbeddingFuncOpenAICompat(BaseURLOpenAI, apiKey, string(model), &normalized)
}

// NewEmbeddingFuncOpenAICompat returns a function that creates embeddings for a text
// using an OpenAI compatible API. For example:
//   - Azure OpenAI: https://azure.microsoft.com/en-us/products/ai-services/openai-service
//   - LitLLM: https://github.com/BerriAI/litellm
//   - Ollama: https://github.com/ollama/ollama/blob/main/docs/openai.md
//   - etc.
//
// The `normalized` parameter indicates whether the vectors returned by the embedding
// model are already normalized, as is the case for OpenAI's and Mistral's models.
// The flag is optional. If it's nil, it will be autodetected on the first request
// (which bears a small risk that the vector just happens to have a length of 1).
func NewEmbeddingFuncOpenAICompat(baseURL, apiKey, model string, normalized *bool) EmbeddingFunc {
	return newEmbeddingFuncOpenAICompat(baseURL, apiKey, model, normalized, nil, nil)
}

// newEmbeddingFuncOpenAICompat returns a function that creates embeddings for a text
// using an OpenAI compatible API.
// It offers options to set request headers and query parameters
// e.g. to pass the `api-key` header and the `api-version` query parameter for Azure OpenAI.
//
// The `normalized` parameter indicates whether the vectors returned by the embedding
// model are already normalized, as is the case for OpenAI's and Mistral's models.
// The flag is optional. If it's nil, it will be autodetected on the first request
// (which bears a small risk that the vector just happens to have a length of 1).
func newEmbeddingFuncOpenAICompat(baseURL, apiKey, model string, normalized *bool, headers map[string]string, queryParams map[string]string) EmbeddingFunc {
	// We don't set a default timeout here, although it's usually a good idea.
	// In our case though, the library user can set the timeout on the context,
	// and it might have to be a long timeout, depending on the text length.
	client := &http.Client{}

	var checkedNormalized bool
	checkNormalized := sync.Once{}

	return func(ctx context.Context, text string) ([]float32, error) {
		// Prepare the request body.
		reqBody, err := json.Marshal(map[string]string{
			"input": text,
			"model": model,
		})
		if err != nil {
			return nil, fmt.Errorf("couldn't marshal request body: %w", err)
		}

		// Create the request. Creating it with context is important for a timeout
		// to be possible, because the client is configured without a timeout.
		req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/embeddings", bytes.NewBuffer(reqBody))
		if err != nil {
			return nil, fmt.Errorf("couldn't create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		// Add headers
		for k, v := range headers {
			req.Header.Add(k, v)
		}

		// Add query parameters
		q := req.URL.Query()
		for k, v := range queryParams {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()

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
		var embeddingResponse openAIResponse
		err = json.Unmarshal(body, &embeddingResponse)
		if err != nil {
			return nil, fmt.Errorf("couldn't unmarshal response body: %w", err)
		}

		// Check if the response contains embeddings.
		if len(embeddingResponse.Data) == 0 || len(embeddingResponse.Data[0].Embedding) == 0 {
			return nil, errors.New("no embeddings found in the response")
		}

		v := embeddingResponse.Data[0].Embedding
		if normalized != nil {
			if *normalized {
				return v, nil
			}
			return normalizeVector(v), nil
		}
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
