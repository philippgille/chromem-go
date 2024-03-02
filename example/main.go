package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/philippgille/chromem-go"
	"github.com/sashabaranov/go-openai"
)

const (
	question = "How many Albatros L 74 planes were produced?"
	// We use a local LLM running in ollama: https://ollama.com/
	ollamaBaseURL = "http://localhost:11434/v1"
	// We use a very small model that doesn't need much resources and is fast, but
	// doesn't have much knowledge: https://ollama.com/library/gemma
	// We found Gemma 2B to be superior to TinyLlama (1.1B), Stable LM 2 (1.6B)
	// and Phi-2 (2.7B) for the retrieval augmented generation (RAG) use case.
	ollamaModel = "gemma:2b"
)

func main() {
	ctx := context.Background()

	// Warm up ollama, in case the model isn't loaded yet
	log.Println("Warming up ollama...")
	_ = askLLM(ctx, nil, "Hello!")

	// First we ask an LLM a fairly specific question that it likely won't know
	// the answer to.
	log.Println("Question: " + question)
	log.Println("Asking LLM...")
	reply := askLLM(ctx, nil, question)
	log.Printf("Initial reply from the LLM: \"" + reply + "\"\n")

	// Now we use our vector database for retrieval augmented generation (RAG),
	// which means we provide the LLM with relevant knowledge.

	// Set up chromem-go with persistence, so that when the program restarts, the
	// DB's data is still available.
	log.Println("Setting up chromem-go...")
	db, err := chromem.NewPersistentDB("./db")
	if err != nil {
		panic(err)
	}
	// Create collection if it wasn't loaded from persistent storage yet.
	// We don't pass any embedding function, leading to the default being used (OpenAI
	// text-embedding-3-small), which requires the OPENAI_API_KEY environment variable
	// to be set.
	collection, err := db.GetOrCreateCollection("Wikipedia", nil, nil)
	if err != nil {
		panic(err)
	}
	// Add docs to the collection, if the collection was just created (and not
	// loaded from persistent storage).
	if collection.Count() == 0 {
		// Here we use a DBpedia sample, where each line contains the lead section/introduction
		// to some Wikipedia article and its category.
		f, err := os.Open("dbpedia_sample.jsonl")
		if err != nil {
			panic(err)
		}
		d := json.NewDecoder(f)
		var ids []string
		var metadatas []map[string]string
		var texts []string
		log.Println("Reading JSON lines...")
		for i := 1; ; i++ {
			var article struct {
				Text     string `json:"text"`
				Category string `json:"category"`
			}
			err := d.Decode(&article)
			if err == io.EOF {
				break // reached end of file
			} else if err != nil {
				panic(err)
			}

			ids = append(ids, strconv.Itoa(i))
			metadatas = append(metadatas, map[string]string{"category": article.Category})
			texts = append(texts, article.Text)
		}
		log.Println("Adding documents to chromem-go, including creating their embeddings via OpenAI API...")
		err = collection.AddConcurrently(ctx, ids, nil, metadatas, texts, runtime.NumCPU())
		if err != nil {
			panic(err)
		}
	} else {
		log.Println("Not reading JSON lines because collection was loaded from persistent storage.")
	}

	// Search for documents similar to the one we added just by passing the original
	// question.
	// We ask for the two most similar documents, but you can use more or less depending
	// on your needs and the supported context size of the LLM you use.
	log.Println("Querying chromem-go...")
	docRes, err := collection.Query(ctx, question, 2, nil, nil)
	if err != nil {
		panic(err)
	}
	// Here you could filter out any documents whose similarity is below a certain threshold.
	// if docRes[...].Similarity < 0.5 { ...

	// Now we can ask the LLM again, augmenting the question with the knowledge we retrieved.
	// In this example we just use both retrieved documents as context.
	contexts := []string{docRes[0].Document, docRes[1].Document}
	log.Println("Asking LLM with augmented question...")
	reply = askLLM(ctx, contexts, question)
	log.Printf("Reply after augmenting the question with knowledge: \"" + reply + "\"\n")

	/* Output (can differ slightly on each run):
	2024/03/02 14:52:40 Warming up ollama...
	2024/03/02 14:52:42 Question: How many Albatros L 74 planes were produced?
	2024/03/02 14:52:42 Asking LLM...
	2024/03/02 14:52:45 Initial reply from the LLM: "I am unable to provide a specific number for the number of Albatros L 74 planes produced, as I do not have access to real-time information or comprehensive records."
	2024/03/02 14:52:45 Setting up chromem-go...
	2024/03/02 14:52:45 Reading JSON lines...
	2024/03/02 14:52:45 Adding documents to chromem-go, including creating their embeddings via OpenAI API...
	2024/03/02 14:52:55 Querying chromem-go...
	2024/03/02 14:52:55 Asking LLM with augmented question...
	2024/03/02 14:53:01 Reply after augmenting the question with knowledge: "Answer: Only two Albatros L 74 planes were produced."
	*/
}

func askLLM(ctx context.Context, contexts []string, question string) string {
	// We use a local LLM running in ollama, which has an OpenAI-compatible API.
	openAIClient := openai.NewClientWithConfig(openai.ClientConfig{
		BaseURL:    ollamaBaseURL,
		HTTPClient: http.DefaultClient,
	})
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: "You are a helpful assistant. You answer the user's questions in a concise manner. If you are not sure, say that you don't know the answer. If the user provides contexts, use them to answer their question.",
		},
	}
	// Add contexts in reverse order, as many LLMs prioritize the latest message
	// or rather forget about older ones (despite fitting into the LLM context).
	for i := len(contexts) - 1; i >= 0; i-- {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: "Context:" + contexts[i],
		})
	}
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: "Question: " + question,
	})
	res, err := openAIClient.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:    ollamaModel,
		Messages: messages,
	})
	if err != nil {
		panic(err)
	}
	reply := res.Choices[0].Message.Content
	reply = strings.TrimSpace(reply)

	return reply
}
