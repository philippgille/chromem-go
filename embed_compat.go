package chromem

const (
	baseURLMistral = "https://api.mistral.ai/v1"
	// Currently there's only one. Let's turn this into a pseudo-enum as soon as there are more.
	embeddingModelMistral = "mistral-embed"
)

// NewEmbeddingFuncMistral returns a function that creates embeddings for a text
// using the Mistral API.
func NewEmbeddingFuncMistral(apiKey string) EmbeddingFunc {
	// Mistral embeddings are normalized, see section "Distance Measures" on
	// https://docs.mistral.ai/guides/embeddings/.
	normalized := true

	// The Mistral API docs don't mention the `encoding_format` as optional,
	// but it seems to be, just like OpenAI. So we reuse the OpenAI function.
	return NewEmbeddingFuncOpenAICompat(baseURLMistral, apiKey, embeddingModelMistral, &normalized)
}

const baseURLJina = "https://api.jina.ai/v1"

type EmbeddingModelJina string

const (
	EmbeddingModelJina2BaseEN   EmbeddingModelJina = "jina-embeddings-v2-base-en"
	EmbeddingModelJina2BaseDE   EmbeddingModelJina = "jina-embeddings-v2-base-de"
	EmbeddingModelJina2BaseCode EmbeddingModelJina = "jina-embeddings-v2-base-code"
	EmbeddingModelJina2BaseZH   EmbeddingModelJina = "jina-embeddings-v2-base-zh"
)

// NewEmbeddingFuncJina returns a function that creates embeddings for a text
// using the Jina API.
func NewEmbeddingFuncJina(apiKey string, model EmbeddingModelJina) EmbeddingFunc {
	return NewEmbeddingFuncOpenAICompat(baseURLJina, apiKey, string(model), nil)
}

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

// NewEmbeddingFuncMixedbread returns a function that creates embeddings for a text
// using the mixedbread.ai API.
func NewEmbeddingFuncMixedbread(apiKey string, model EmbeddingModelMixedbread) EmbeddingFunc {
	return NewEmbeddingFuncOpenAICompat(baseURLMixedbread, apiKey, string(model), nil)
}

const baseURLLocalAI = "http://localhost:8080/v1"

// NewEmbeddingFuncLocalAI returns a function that creates embeddings for a text
// using the LocalAI API.
// You can start a LocalAI instance like this:
//
//	docker run -it -p 127.0.0.1:8080:8080 localai/localai:v2.7.0-ffmpeg-core bert-cpp
//
// And then call this constructor with model "bert-cpp-minilm-v6".
// But other embedding models are supported as well. See the LocalAI documentation
// for details.
func NewEmbeddingFuncLocalAI(model string) EmbeddingFunc {
	return NewEmbeddingFuncOpenAICompat(baseURLLocalAI, "", model, nil)
}

const (
	azureDefaultAPIVersion = "2024-02-01"
)

// NewEmbeddingFuncAzureOpenAI returns a function that creates embeddings for a text
// using the Azure OpenAI API.
// The `deploymentURL` is the URL of the deployed model, e.g. "https://YOUR_RESOURCE_NAME.openai.azure.com/openai/deployments/YOUR_DEPLOYMENT_NAME"
// See https://learn.microsoft.com/en-us/azure/ai-services/openai/how-to/embeddings?tabs=console#how-to-get-embeddings
func NewEmbeddingFuncAzureOpenAI(apiKey string, deploymentURL string, apiVersion string, model string) EmbeddingFunc {
	if apiVersion == "" {
		apiVersion = azureDefaultAPIVersion
	}
	return newEmbeddingFuncOpenAICompat(deploymentURL, apiKey, model, nil, map[string]string{"api-key": apiKey}, map[string]string{"api-version": apiVersion})
}
