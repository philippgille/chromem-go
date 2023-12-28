package chromem

import (
	"context"
	"os"

	"github.com/sashabaranov/go-openai"
)

// createEmbeddings creates the embeddings for a document.
// It uses the OpenAI ada v2 model via their API.
func createEmbeddings(ctx context.Context, document string) ([]float32, error) {
	client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
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
