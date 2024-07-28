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

type EmbeddingModelVertex string

const (
	EmbeddingModelVertexEnglishV1 EmbeddingModelVertex = "textembedding-gecko@001"
	EmbeddingModelVertexEnglishV2 EmbeddingModelVertex = "textembedding-gecko@002"
	EmbeddingModelVertexEnglishV3 EmbeddingModelVertex = "textembedding-gecko@003"
	EmbeddingModelVertexEnglishV4 EmbeddingModelVertex = "text-embedding-004"

	EmbeddingModelVertexMultilingualV1 EmbeddingModelVertex = "textembedding-gecko-multilingual@001"
	EmbeddingModelVertexMultilingualV2 EmbeddingModelVertex = "text-multilingual-embedding-002"
)

const baseURLVertex = "https://us-central1-aiplatform.googleapis.com/v1"

type vertexOptions struct {
	apiEndpoint  string
	autoTruncate bool
}

func defaultVertexOptions() *vertexOptions {
	return &vertexOptions{
		apiEndpoint:  baseURLVertex,
		autoTruncate: false,
	}
}

type VertexOption func(*vertexOptions)

func WithVertexAPIEndpoint(apiEndpoint string) VertexOption {
	return func(o *vertexOptions) {
		o.apiEndpoint = apiEndpoint
	}
}

func WithVertexAutoTruncate(autoTruncate bool) VertexOption {
	return func(o *vertexOptions) {
		o.autoTruncate = autoTruncate
	}
}

type vertexResponse struct {
	Predictions []vertexPrediction `json:"predictions"`
}

type vertexPrediction struct {
	Embeddings vertexEmbeddings `json:"embeddings"`
}

type vertexEmbeddings struct {
	Values []float32 `json:"values"`
	// there's more here, but we only care about the embeddings
}

func NewEmbeddingFuncVertex(apiKey, project string, model EmbeddingModelVertex, opts ...VertexOption) EmbeddingFunc {
	cfg := defaultVertexOptions()
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.apiEndpoint == "" {
		cfg.apiEndpoint = baseURLVertex
	}

	// We don't set a default timeout here, although it's usually a good idea.
	// In our case though, the library user can set the timeout on the context,
	// and it might have to be a long timeout, depending on the text length.
	client := &http.Client{}

	var checkedNormalized bool
	checkNormalized := sync.Once{}

	return func(ctx context.Context, text string) ([]float32, error) {
		b := map[string]any{
			"instances": []map[string]any{
				{
					"content": text,
				},
			},
			"parameters": map[string]any{
				"autoTruncate": cfg.autoTruncate,
			},
		}

		// Prepare the request body.
		reqBody, err := json.Marshal(b)
		if err != nil {
			return nil, fmt.Errorf("couldn't marshal request body: %w", err)
		}

		fullURL := fmt.Sprintf("%s/projects/%s/locations/us-central1/publishers/google/models/%s:predict", cfg.apiEndpoint, project, model)

		// Create the request. Creating it with context is important for a timeout
		// to be possible, because the client is configured without a timeout.
		req, err := http.NewRequestWithContext(ctx, "POST", fullURL, bytes.NewBuffer(reqBody))
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
		var embeddingResponse vertexResponse
		err = json.Unmarshal(body, &embeddingResponse)
		if err != nil {
			return nil, fmt.Errorf("couldn't unmarshal response body: %w", err)
		}

		// Check if the response contains embeddings.
		if len(embeddingResponse.Predictions) == 0 || len(embeddingResponse.Predictions[0].Embeddings.Values) == 0 {
			return nil, errors.New("no embeddings found in the response")
		}

		v := embeddingResponse.Predictions[0].Embeddings.Values
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
