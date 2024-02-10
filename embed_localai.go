package chromem

const baseURLLocalAI = "http://localhost:8080/v1"

// NewEmbeddingFuncLocalAI returns a function that creates embeddings for a document
// using the LocalAI API.
// You can start a LocalAI instance like this:
//
//	docker run -it -p 127.0.0.1:8080:8080 localai/localai:v2.7.0-ffmpeg-core bert-cpp
//
// And then call this constructor with model "bert-cpp-minilm-v6".
// But other embedding models are supported as well. See the LocalAI documentation
// for details.
func NewEmbeddingFuncLocalAI(model string) EmbeddingFunc {
	return NewEmbeddingFuncOpenAICompat(baseURLLocalAI, "", model)
}
