package main

import (
	"context"
	"fmt"
	"log"
	"runtime"

	"github.com/philippgille/chromem-go"

	"github.com/joho/godotenv"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	ctx := context.Background()

	db, err := chromem.NewPersistentDBWithOptions("./collections", chromem.DBConfig{
		Compress:          true,
		Wal:               true,
		WalSegmentMaxSize: 8 * 1024 * 1024,
	})

	if err != nil {
		log.Fatal("Cannot Create the db object")
	}

	defer db.Close()

	// Passing nil as embedding function leads to OpenAI being used and requires
	// "OPENAI_API_KEY" env var to be set. Other providers are supported as well.
	// For example pass `chromem.NewEmbeddingFuncOllama(...)` to use Ollama.
	c, err := db.GetOrCreateCollection("knowledge-base", nil, nil)
	if err != nil {
		panic(err)
	}

	err = c.AddDocuments(ctx, []chromem.Document{
		{
			ID:      "1",
			Content: "The sky is blue because of Rayleigh scattering.",
			Metadata: map[string]string{
				"name": "vaibhav",
			},
		},
		{
			ID:      "2",
			Content: "Leaves are green because chlorophyll absorbs red and blue light.",
			Metadata: map[string]string{
				"name": "bhardwaj",
			},
		},
	}, runtime.NumCPU())
	if err != nil {
		panic(err)
	}

	res, err := c.Query(ctx, "Why is the sky blue?", 1, map[string]string{
		"name": "bhardwaj",
	}, nil)
	if err != nil {
		panic(err)
	}

	fmt.Printf("ID: %v\nSimilarity: %v\nContent: %v\n", res[0].ID, res[0].Similarity, res[0].Content)
}
