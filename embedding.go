package chromem

import (
	"context"
	"os"

	"github.com/sashabaranov/go-openai"
)

// CreateEmbeddingsDefault returns a function that creates embeddings for a document using using
// OpenAI`s ada v2 model via their API.
// The API key is read from the environment variable "OPENAI_API_KEY".
func CreateEmbeddingsDefault() EmbeddingFunc {
	apiKey := os.Getenv("OPENAI_API_KEY")
	return CreateEmbeddingsOpenAI(apiKey)
}

// CreateEmbeddingsOpenAI returns a function that creates the embeddings for a document
// using OpenAI`s ada v2 model via their API.
func CreateEmbeddingsOpenAI(apiKey string) EmbeddingFunc {
	client := openai.NewClient(apiKey)
	return func(ctx context.Context, document string) ([]float32, error) {
		req := openai.EmbeddingRequest{
			Input: document,
			Model: openai.AdaEmbeddingV2,
		}

		res, err := client.CreateEmbeddings(ctx, req)
		if err != nil {
			return nil, err
		}

		return res.Data[0].Embedding, nil
	}
}
