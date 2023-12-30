package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/philippgille/chromem-go"
	"github.com/sashabaranov/go-openai"
)

func main() {
	ctx := context.Background()

	// First we ask an LLM a specific question that it won't know the answer to.
	openAIClient := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
	res, err := openAIClient.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT3Dot5Turbo1106,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: "For which storage backends does gokv provide an implementation?",
			},
		},
	})
	if err != nil {
		panic(err)
	}
	reply := res.Choices[0].Message.Content
	reply = "\t" + strings.ReplaceAll(reply, "\n", "\n\t") // Indent for readability
	fmt.Println("Reply before providing the LLM with context:")
	fmt.Printf("============================================\n\n")
	fmt.Println(reply)

	// Now we use our vector database for retrieval augmented generation (RAG) to provide the LLM with context.

	// Set up chromem-go in-memory, for easy prototyping.
	client := chromem.NewClient()
	// Create collection.
	collection := client.CreateCollection("philippgille-projects", nil, nil)
	// Add docs to the collection.
	// Here we're adding the READMEs of some of my projects.
	projects := []string{"gokv", "ln-paywall", "chromem-go"}
	readmes := make([]string, 0, len(projects))
	for _, project := range projects {
		res, err := http.Get(fmt.Sprintf("https://raw.githubusercontent.com/philippgille/%s/master/README.md", project))
		if err != nil {
			panic(err)
		}
		defer res.Body.Close()
		text, err := io.ReadAll(res.Body)
		if err != nil {
			panic(err)
		}
		readmes = append(readmes, string(text))
	}
	err = collection.Add(ctx, projects, nil, nil, readmes)
	if err != nil {
		panic(err)
	}

	// Search for documents similar to the one we added
	docRes, err := collection.Query(ctx, "For which storage backends does gokv provide an implementation?", 1, nil, nil)
	if err != nil {
		panic(err)
	}
	// We asked for a single result (the most similar one), so we can just take the first element.
	doc := docRes[0]

	// Now we can ask the LLM again, providing the document we found with the query as context.
	res, err = openAIClient.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT3Dot5Turbo1106,
		Messages: []openai.ChatCompletionMessage{
			{
				Role: openai.ChatMessageRoleUser,
				Content: "For which storage backends does gokv provide an implementation?\n\n" +
					"Here is some context that might help you answer my question:\n\n" +
					doc.Document,
			},
		},
	})
	if err != nil {
		panic(err)
	}
	reply = res.Choices[0].Message.Content
	reply = "\t" + strings.ReplaceAll(reply, "\n", "\n\t") // Indent for readability
	fmt.Println("\nReply after providing the LLM with context:")
	fmt.Printf("===========================================\n\n")
	fmt.Println(reply)
}
