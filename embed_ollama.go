package chromem

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
)

const defaultBaseURLOllama = "http://localhost:11434/api"

type ollamaResponse struct {
	Embedding []float32 `json:"embedding"`
}

// NewEmbeddingFuncOllama returns a function that creates embeddings for a text
// using Ollama's embedding API. You can pass any model that Ollama supports and
// that supports embeddings. A good one as of 2024-03-02 is "nomic-embed-text".
// See https://ollama.com/library/nomic-embed-text
// baseURLOllama is the base URL of the Ollama API. If it's empty,
// "http://localhost:11434/api" is used.
func NewEmbeddingFuncOllama(model string, baseURLOllama string) EmbeddingFunc {
	if baseURLOllama == "" {
		baseURLOllama = defaultBaseURLOllama
	}

	// We don't set a default timeout here, although it's usually a good idea.
	// In our case though, the library user can set the timeout on the context,
	// and it might have to be a long timeout, depending on the text length.
	client := &http.Client{}

	var checkedNormalized bool
	checkNormalized := sync.Once{}

	return func(ctx context.Context, text string) ([]float32, error) {
		// Prepare the request body.
		reqBody, err := json.Marshal(map[string]string{
			"model":  model,
			"prompt": text,
		})
		if err != nil {
			return nil, fmt.Errorf("couldn't marshal request body: %w", err)
		}

		// Create the request. Creating it with context is important for a timeout
		// to be possible, because the client is configured without a timeout.
		req, err := http.NewRequestWithContext(ctx, "POST", baseURLOllama+"/embeddings", bytes.NewBuffer(reqBody))
		if err != nil {
			return nil, fmt.Errorf("couldn't create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

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
		var embeddingResponse ollamaResponse
		err = json.Unmarshal(body, &embeddingResponse)
		if err != nil {
			return nil, fmt.Errorf("couldn't unmarshal response body: %w", err)
		}

		// Check if the response contains embeddings.
		if len(embeddingResponse.Embedding) == 0 {
			return nil, errors.New("no embeddings found in the response")
		}

		v := embeddingResponse.Embedding
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
