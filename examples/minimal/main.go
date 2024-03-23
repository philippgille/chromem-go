package main

import (
	"context"
	"fmt"
	"runtime"

	"github.com/philippgille/chromem-go"
)

func main() {
	ctx := context.Background()

	db := chromem.NewDB()

	c, err := db.CreateCollection("knowledge-base", nil, nil)
	if err != nil {
		panic(err)
	}

	c.AddDocuments(ctx, []chromem.Document{
		{
			ID:      "1",
			Content: "The sky is blue because of Rayleigh scattering.",
		},
		{
			ID:      "2",
			Content: "Leaves are green because chlorophyll absorbs red and blue light.",
		},
	}, runtime.NumCPU())

	res, err := c.Query(ctx, "Why is the sky blue?", 1, nil, nil)
	if err != nil {
		panic(err)
	}

	fmt.Printf("ID: %v\nSimilarity: %v\nContent: %v\n", res[0].ID, res[0].Similarity, res[0].Content)

	/* Output:
	ID: 1
	Similarity: 0.6833369
	Content: The sky is blue because of Rayleigh scattering.
	*/
}
