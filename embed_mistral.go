package chromem

const (
	baseURLMistral = "https://api.mistral.ai/v1"
	// Currently there's only one. Let's turn this into a pseudo-enum as soon as there are more.
	embeddingModelMistral = "mistral-embed"
)

// NewEmbeddingFuncMistral returns a function that creates embeddings for a document
// using the Mistral API.
func NewEmbeddingFuncMistral(apiKey string) EmbeddingFunc {
	// The Mistral API docs don't mention the `encoding_format` as optional,
	// but it seems to be, just like OpenAI. So we reuse the OpenAI function.
	return NewEmbeddingFuncOpenAICompat(baseURLMistral, apiKey, embeddingModelMistral)
}
