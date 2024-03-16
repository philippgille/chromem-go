package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/philippgille/chromem-go"
)

const (
	question = "When did the Monarch Company exist?"
	// We use a local LLM running in Ollama for the embedding: https://huggingface.co/nomic-ai/nomic-embed-text-v1.5
	embeddingModel = "nomic-embed-text"
)

func main() {
	ctx := context.Background()

	// Warm up Ollama, in case the model isn't loaded yet
	log.Println("Warming up Ollama...")
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
	// You can pass nil as embedding function to use the default (OpenAI text-embedding-3-small),
	// which is very good and cheap. It would require the OPENAI_API_KEY environment
	// variable to be set.
	// For this example we choose to use a locally running embedding model though.
	// It requires Ollama to serve its API at "http://localhost:11434/api".
	collection, err := db.GetOrCreateCollection("Wikipedia", nil, chromem.NewEmbeddingFuncOllama(embeddingModel))
	if err != nil {
		panic(err)
	}
	// Add docs to the collection, if the collection was just created (and not
	// loaded from persistent storage).
	docs := []chromem.Document{}
	if collection.Count() == 0 {
		// Here we use a DBpedia sample, where each line contains the lead section/introduction
		// to some Wikipedia article and its category.
		f, err := os.Open("dbpedia_sample.jsonl")
		if err != nil {
			panic(err)
		}
		defer f.Close()
		d := json.NewDecoder(f)
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

			docs = append(docs, chromem.Document{
				ID:       strconv.Itoa(i),
				Metadata: map[string]string{"category": article.Category},
				Content:  article.Text,
			})
		}
		log.Println("Adding documents to chromem-go, including creating their embeddings via Ollama API...")
		err = collection.AddDocuments(ctx, docs, runtime.NumCPU())
		if err != nil {
			panic(err)
		}
	} else {
		log.Println("Not reading JSON lines because collection was loaded from persistent storage.")
	}

	// Search for documents that are semantically similar to the original question.
	// We ask for the two most similar documents, but you can use more or less depending
	// on your needs and the supported context size of the LLM you use.
	// You can limit the search by filtering on content or metadata (like the article's
	// category), but we don't do that in this example.
	start := time.Now()
	log.Println("Querying chromem-go...")
	docRes, err := collection.Query(ctx, question, 2, nil, nil)
	if err != nil {
		panic(err)
	}
	log.Println("Search took", time.Since(start))
	// Here you could filter out any documents whose similarity is below a certain threshold.
	// if docRes[...].Similarity < 0.5 { ...

	// Print the retrieved documents and their similarity to the question.
	for i, res := range docRes {
		log.Printf("Document %d (similarity: %f): \"%s\"\n", i+1, res.Similarity, res.Content)
	}

	// Now we can ask the LLM again, augmenting the question with the knowledge we retrieved.
	// In this example we just use both retrieved documents as context.
	contexts := []string{docRes[0].Content, docRes[1].Content}
	log.Println("Asking LLM with augmented question...")
	reply = askLLM(ctx, contexts, question)
	log.Printf("Reply after augmenting the question with knowledge: \"" + reply + "\"\n")

	/* Output (can differ slightly on each run):
	2024/03/02 20:02:30 Warming up Ollama...
	2024/03/02 20:02:33 Question: When did the Monarch Company exist?
	2024/03/02 20:02:33 Asking LLM...
	2024/03/02 20:02:34 Initial reply from the LLM: "I cannot provide information on the Monarch Company, as I am unable to access real-time or comprehensive knowledge sources."
	2024/03/02 20:02:34 Setting up chromem-go...
	2024/03/02 20:02:34 Reading JSON lines...
	2024/03/02 20:02:34 Adding documents to chromem-go, including creating their embeddings via Ollama API...
	2024/03/02 20:03:11 Querying chromem-go...
	2024/03/02 20:03:11 Search took 231.672667ms
	2024/03/02 20:03:11 Document 1 (similarity: 0.723627): "Malleable Iron Range Company was a company that existed from 1896 to 1985 and primarily produced kitchen ranges made of malleable iron but also produced a variety of other related products. The company's primary trademark was 'Monarch' and was colloquially often referred to as the Monarch Company or just Monarch."
	2024/03/02 20:03:11 Document 2 (similarity: 0.550584): "The American Motor Car Company was a short-lived company in the automotive industry founded in 1906 lasting until 1913. It was based in Indianapolis Indiana United States. The American Motor Car Company pioneered the underslung design."
	2024/03/02 20:03:11 Asking LLM with augmented question...
	2024/03/02 20:03:32 Reply after augmenting the question with knowledge: "The Monarch Company existed from 1896 to 1985."
	*/
}
