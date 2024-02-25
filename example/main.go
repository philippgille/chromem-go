package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"

	"github.com/philippgille/chromem-go"
	"github.com/sashabaranov/go-openai"
)

const (
	question = "Wich smooth jazz album received a Grammy nomination in 2009? I want to know the album name and artist."
	// We use a local LLM running in ollama: https://ollama.com/
	ollamaBaseURL = "http://localhost:11434/v1"
	// We use a very small model that doesn't need much resources and is fast, but
	// doesn't have much knowledge: https://ollama.com/library/tinyllama
	ollamaModel = "tinyllama:1.1b"
)

func main() {
	ctx := context.Background()

	// First we ask an LLM a fairly specific question that it won't know the answer
	// to.
	log.Println("Asking LLM...")
	reply := askLLM(ctx, "", question)
	fmt.Printf("\nInitial reply from the LLM:\n" +
		"===========================\n\n" +
		reply + "\n\n")

	// Now we use our vector database for retrieval augmented generation (RAG),
	// which means we provide the LLM with relevant knowledge.

	// Set up chromem-go with persistence, so that when the program restarts, the
	// DB's data is still available.
	log.Println("Setting up chromem-go...")
	db, err := chromem.NewPersistentDB("./db")
	if err != nil {
		panic(err)
	}
	// Create collection.
	// We don't pass any embedding function, leading to the default being used (OpenAI
	// text-embedding-3-small), which requires the OPENAI_API_KEY environment variable
	// to be set.
	collection, err := db.CreateCollection("Wikipedia", nil, nil)
	if err != nil {
		panic(err)
	}
	// Add docs to the collection.
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
	// In this example we just read the first 20 lines, but in a real-world scenario
	// you'd read the entire file.
	for i := 0; i < 20; i++ {
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
	log.Println("Adding documents to chromem-go...")
	err = collection.AddConcurrently(ctx, ids, nil, metadatas, texts, runtime.NumCPU())
	if err != nil {
		panic(err)
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
	context := docRes[0].Document + "\n\n" + docRes[1].Document
	log.Println("Asking LLM with augmented question...")
	reply = askLLM(ctx, context, question)
	fmt.Printf("\nReply after augmenting the question with knowledge:\n" +
		"===================================================\n\n" +
		reply + "\n\n")

	/* Output (can differ slightly on each run):

	2024/02/17 15:25:04 Asking LLM...

	Initial reply from the LLM:
	===========================

	"The Album That Received A Grammy Nominated In 2009" or "A Smooth Jazz Album That Was Nominated For The Grammy Award In 2009".

	2024/02/17 15:25:06 Setting up chromem-go...
	2024/02/17 15:25:06 Reading JSON lines...
	2024/02/17 15:25:06 Adding documents to chromem-go...
	2024/02/17 15:25:08 Querying chromem-go...
	2024/02/17 15:25:08 Asking LLM with augmented question...

	Reply after augmenting the question with knowledge:
	===================================================

	"The Spice of Life" by Earl Klugh. The nomination was for Best Pop Instrumental Album at the 51st Grammy Awards in 2009.
	*/
}

func askLLM(ctx context.Context, context, question string) string {
	// We use a local LLM running in ollama, which has an OpenAI-compatible API.
	// openAIClient := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
	openAIClient := openai.NewClientWithConfig(openai.ClientConfig{
		BaseURL:    ollamaBaseURL,
		HTTPClient: http.DefaultClient,
	})
	res, err := openAIClient.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		// Model: openai.GPT3Dot5Turbo,
		Model: ollamaModel,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "You are a helpful assistant. You answer the user's questions. Combine your knowledge with the context that the user might provide, as it's likely relevant to the user's question. If you are not sure, say that you don't know the answer.",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: "Context: " + context,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: "Question: " + question,
			},
			{
				Role:    openai.ChatMessageRoleAssistant,
				Content: "Based on your provided context and question, I think the answer is:",
			},
		},
	})
	if err != nil {
		panic(err)
	}
	reply := res.Choices[0].Message.Content

	return reply
}
