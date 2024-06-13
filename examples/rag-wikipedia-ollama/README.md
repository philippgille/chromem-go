# RAG Wikipedia Ollama

This example shows a retrieval augmented generation (RAG) application, using `chromem-go` as knowledge base for finding relevant info for a question. More specifically the app is doing *question answering*. The underlying data is 200 Wikipedia articles (or rather their lead section / introduction).

We run the embeddings model and LLM in [Ollama](https://github.com/ollama/ollama), to showcase how a RAG application can run entirely offline, without relying on OpenAI or other third party APIs. It doesn't require a GPU, and a CPU like an 11th Gen Intel i5-1135G7 (like in the first generation Framework Laptop 13) is fast enough.

As LLM we use Google's [Gemma (2B)](https://huggingface.co/google/gemma-2b), a very small model that doesn't need many resources and is fast, but doesn't have much knowledge, so it's a prime example for the combination of LLMs and vector databases. We found Gemma 2B to be superior to [TinyLlama (1.1B)](https://huggingface.co/TinyLlama/TinyLlama-1.1B-Chat-v1.0), [Stable LM 2 (1.6B)](https://huggingface.co/stabilityai/stablelm-2-zephyr-1_6b) and [Phi-2 (2.7B)](https://huggingface.co/microsoft/phi-2) for the RAG use case.

As embeddings model we use Nomic's [nomic-embed-text v1.5](https://huggingface.co/nomic-ai/nomic-embed-text-v1.5).

## How to run

1. Install Ollama: <https://ollama.com/download>
2. Download the two models:
   - `ollama pull gemma:2b`
   - `ollama pull nomic-embed-text`
3. Run the example: `go run .`

## Output

The output can differ slightly on each run, but it's along the lines of:

```log
2024/03/02 20:02:30 Warming up Ollama...
2024/03/02 20:02:33 Question: When did the Monarch Company exist?
2024/03/02 20:02:33 Asking LLM...
2024/03/02 20:02:34 Initial reply from the LLM: "I cannot provide information on the Monarch Company, as I am unable to access real-time or comprehensive knowledge sources."
2024/03/02 20:02:34 Setting up chromem-go...
2024/03/02 20:02:34 Reading JSON lines...
2024/03/02 20:02:34 Adding documents to chromem-go, including creating their embeddings via Ollama API...
2024/03/02 20:03:11 Querying chromem-go...
2024/03/02 20:03:11 Search (incl query embedding) took 231.672667ms
2024/03/02 20:03:11 Document 1 (similarity: 0.655357): "Malleable Iron Range Company was a company that existed from 1896 to 1985 and primarily produced kitchen ranges made of malleable iron but also produced a variety of other related products. The company's primary trademark was 'Monarch' and was colloquially often referred to as the Monarch Company or just Monarch."
2024/03/02 20:03:11 Document 2 (similarity: 0.504042): "The American Motor Car Company was a short-lived company in the automotive industry founded in 1906 lasting until 1913. It was based in Indianapolis Indiana United States. The American Motor Car Company pioneered the underslung design."
2024/03/02 20:03:11 Asking LLM with augmented question...
2024/03/02 20:03:32 Reply after augmenting the question with knowledge: "The Monarch Company existed from 1896 to 1985."
```

The majority of the time here is spent during the embeddings creation, where we are limited by the performance of the Ollama API, which depends on your CPU/GPU and the embeddings model.

## OpenAI

You can easily adapt the code to work with OpenAI instead of locally in Ollama.

Add the OpenAI API key in your environment as `OPENAI_API_KEY`.

Then, if you want to create the embeddings via OpenAI, but still use Gemma 2B as LLM:

<details><summary>Apply this patch</summary>

```diff
diff --git a/examples/rag-wikipedia-ollama/main.go b/examples/rag-wikipedia-ollama/main.go
index 55b3076..cee9561 100644
--- a/examples/rag-wikipedia-ollama/main.go
+++ b/examples/rag-wikipedia-ollama/main.go
@@ -14,8 +14,6 @@ import (
 
 const (
        question = "When did the Monarch Company exist?"
-       // We use a local LLM running in Ollama for the embedding: https://huggingface.co/nomic-ai/nomic-embed-text-v1.5
-       embeddingModel = "nomic-embed-text"
 )
 
 func main() {
@@ -48,7 +46,7 @@ func main() {
        // variable to be set.
        // For this example we choose to use a locally running embedding model though.
        // It requires Ollama to serve its API at "http://localhost:11434/api".
-       collection, err := db.GetOrCreateCollection("Wikipedia", nil, chromem.NewEmbeddingFuncOllama(embeddingModel))
+       collection, err := db.GetOrCreateCollection("Wikipedia", nil, nil)
        if err != nil {
                panic(err)
        }
@@ -82,7 +80,7 @@ func main() {
                                Content:  article.Text,
                        })
                }
-               log.Println("Adding documents to chromem-go, including creating their embeddings via Ollama API...")
+               log.Println("Adding documents to chromem-go, including creating their embeddings via OpenAI API...")
                err = collection.AddDocuments(ctx, docs, runtime.NumCPU())
                if err != nil {
                        panic(err)
```

</details>

Or alternatively, if you want to use OpenAI for everything (embeddings creation and LLM):

<details><summary>Apply this patch</summary>

```diff
diff --git a/examples/rag-wikipedia-ollama/llm.go b/examples/rag-wikipedia-ollama/llm.go
index 1fde4ec..7cb81cc 100644
--- a/examples/rag-wikipedia-ollama/llm.go
+++ b/examples/rag-wikipedia-ollama/llm.go
@@ -2,23 +2,13 @@ package main
 
 import (
  "context"
- "net/http"
+ "os"
  "strings"
  "text/template"
 
  "github.com/sashabaranov/go-openai"
 )
 
-const (
- // We use a local LLM running in Ollama for asking the question: https://github.com/ollama/ollama
- ollamaBaseURL = "http://localhost:11434/v1"
- // We use Google's Gemma (2B), a very small model that doesn't need much resources
- // and is fast, but doesn't have much knowledge: https://huggingface.co/google/gemma-2b
- // We found Gemma 2B to be superior to TinyLlama (1.1B), Stable LM 2 (1.6B)
- // and Phi-2 (2.7B) for the retrieval augmented generation (RAG) use case.
- llmModel = "gemma:2b"
-)
-
 // There are many different ways to provide the context to the LLM.
 // You can pass each context as user message, or the list as one user message,
 // or pass it in the system prompt. The system prompt itself also has a big impact
@@ -47,10 +37,7 @@ Don't mention the knowledge base, context or search results in your answer.
 
 func askLLM(ctx context.Context, contexts []string, question string) string {
  // We can use the OpenAI client because Ollama is compatible with OpenAI's API.
- openAIClient := openai.NewClientWithConfig(openai.ClientConfig{
-  BaseURL:    ollamaBaseURL,
-  HTTPClient: http.DefaultClient,
- })
+ openAIClient := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
  sb := &strings.Builder{}
  err := systemPromptTpl.Execute(sb, contexts)
  if err != nil {
@@ -66,7 +53,7 @@ func askLLM(ctx context.Context, contexts []string, question string) string {
   },
  }
  res, err := openAIClient.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
-  Model:    llmModel,
+  Model:    openai.GPT3Dot5Turbo,
   Messages: messages,
  })
  if err != nil {
diff --git a/examples/rag-wikipedia-ollama/main.go b/examples/rag-wikipedia-ollama/main.go
index 55b3076..044a246 100644
--- a/examples/rag-wikipedia-ollama/main.go
+++ b/examples/rag-wikipedia-ollama/main.go
@@ -12,19 +12,11 @@ import (
  "github.com/philippgille/chromem-go"
 )
 
-const (
- question = "When did the Monarch Company exist?"
- // We use a local LLM running in Ollama for the embedding: https://huggingface.co/nomic-ai/nomic-embed-text-v1.5
- embeddingModel = "nomic-embed-text"
-)
+const question = "When did the Monarch Company exist?"
 
 func main() {
  ctx := context.Background()
 
- // Warm up Ollama, in case the model isn't loaded yet
- log.Println("Warming up Ollama...")
- _ = askLLM(ctx, nil, "Hello!")
-
  // First we ask an LLM a fairly specific question that it likely won't know
  // the answer to.
  log.Println("Question: " + question)
@@ -48,7 +40,7 @@ func main() {
  // variable to be set.
  // For this example we choose to use a locally running embedding model though.
  // It requires Ollama to serve its API at "http://localhost:11434/api".
- collection, err := db.GetOrCreateCollection("Wikipedia", nil, chromem.NewEmbeddingFuncOllama(embeddingModel))
+ collection, err := db.GetOrCreateCollection("Wikipedia", nil, nil)
  if err != nil {
   panic(err)
  }
@@ -82,7 +74,7 @@ func main() {
     Content:  article.Text,
    })
   }
-  log.Println("Adding documents to chromem-go, including creating their embeddings via Ollama API...")
+  log.Println("Adding documents to chromem-go, including creating their embeddings via OpenAI API...")
   err = collection.AddDocuments(ctx, docs, runtime.NumCPU())
   if err != nil {
    panic(err)
```

</details>
