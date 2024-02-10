package chromem

const baseURLJina = "https://api.jina.ai/v1"

type EmbeddingModelJina string

const (
	EmbeddingModelJina2BaseEN   EmbeddingModelJina = "jina-embeddings-v2-base-en"
	EmbeddingModelJina2BaseDE   EmbeddingModelJina = "jina-embeddings-v2-base-de"
	EmbeddingModelJina2BaseCode EmbeddingModelJina = "jina-embeddings-v2-base-code"
	EmbeddingModelJina2BaseZH   EmbeddingModelJina = "jina-embeddings-v2-base-zh"
)

// NewEmbeddingFuncJina returns a function that creates embeddings for a document
// using the Jina API.
func NewEmbeddingFuncJina(apiKey string, model EmbeddingModelJina) EmbeddingFunc {
	return NewEmbeddingFuncOpenAICompat(baseURLJina, apiKey, string(model))
}
