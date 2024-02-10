package chromem

const baseURLMixedbread = "https://api.mixedbread.ai"

type EmbeddingModelMixedbread string

const (
	EmbeddingModelMixedbreadUAELargeV1          EmbeddingModelMixedbread = "UAE-Large-V1"
	EmbeddingModelMixedbreadBGELargeENV15       EmbeddingModelMixedbread = "bge-large-en-v1.5"
	EmbeddingModelMixedbreadGTELarge            EmbeddingModelMixedbread = "gte-large"
	EmbeddingModelMixedbreadE5LargeV2           EmbeddingModelMixedbread = "e5-large-v2"
	EmbeddingModelMixedbreadMultilingualE5Large EmbeddingModelMixedbread = "multilingual-e5-large"
	EmbeddingModelMixedbreadMultilingualE5Base  EmbeddingModelMixedbread = "multilingual-e5-base"
	EmbeddingModelMixedbreadAllMiniLML6V2       EmbeddingModelMixedbread = "all-MiniLM-L6-v2"
	EmbeddingModelMixedbreadGTELargeZh          EmbeddingModelMixedbread = "gte-large-zh"
)

// NewEmbeddingFuncMixedbread returns a function that creates embeddings for a document
// using the mixedbread.ai API.
func NewEmbeddingFuncMixedbread(apiKey string, model EmbeddingModelMixedbread) EmbeddingFunc {
	return NewEmbeddingFuncOpenAICompat(baseURLMixedbread, apiKey, string(model))
}
