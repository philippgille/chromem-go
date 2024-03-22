package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/philippgille/chromem-go"
)

const searchTerm = "semantic search with vector databases"

func main() {
	ctx := context.Background()

	// Set up chromem-go with persistence, so that when the program restarts, the
	// DB's data is still available.
	log.Println("Setting up chromem-go...")
	db, err := chromem.NewPersistentDB("./db", false)
	if err != nil {
		panic(err)
	}
	// Create collection if it wasn't loaded from persistent storage yet.
	// We pass nil as embedding function to use the default (OpenAI text-embedding-3-small),
	// which is very good and cheap. It requires the OPENAI_API_KEY environment
	// variable to be set.
	collection, err := db.GetOrCreateCollection("arXiv cs.CL 2023", nil, nil)
	if err != nil {
		panic(err)
	}
	// Add docs to the collection, if the collection was just created (and not
	// loaded from persistent storage).
	docs := []chromem.Document{}
	if collection.Count() == 0 {
		// Here we use an arXiv metadata sample, where each line contains the metadata
		// of a paper, including its submitter, title and abstract.
		f, err := os.Open("/tmp/arxiv_cs-cl_2023.jsonl")
		if err != nil {
			panic(err)
		}
		defer f.Close()
		d := json.NewDecoder(f)
		log.Println("Reading JSON lines...")
		i := 0
		for {
			var paper struct {
				ID        string `json:"id"`
				Submitter string `json:"submitter"`
				Title     string `json:"title"`
				Abstract  string `json:"abstract"`
			}
			err := d.Decode(&paper)
			if err == io.EOF {
				break // reached end of file
			} else if err != nil {
				panic(err)
			}

			title := strings.ReplaceAll(paper.Title, "\n", " ")
			title = strings.ReplaceAll(title, "  ", " ")
			content := strings.TrimSpace(paper.Abstract)
			docs = append(docs, chromem.Document{
				ID:       paper.ID,
				Metadata: map[string]string{"submitter": paper.Submitter, "title": title},
				Content:  content,
			})
			i++
		}
		log.Println("Read and parsed", i, "documents.")
		log.Println("Adding documents to chromem-go, including creating their embeddings via OpenAI API...")
		err = collection.AddDocuments(ctx, docs, runtime.NumCPU())
		if err != nil {
			panic(err)
		}
	} else {
		log.Println("Not reading JSON lines because collection was loaded from persistent storage.")
	}

	// Search for documents that are semantically similar to the search term.
	// We ask for the 10 most similar documents, but you can use more or less depending
	// on your needs.
	// You can limit the search by filtering on content or metadata (like the paper's
	// submitter), but we don't do that in this example.
	log.Println("Querying chromem-go...")
	start := time.Now()
	docRes, err := collection.Query(ctx, searchTerm, 10, nil, nil)
	if err != nil {
		panic(err)
	}
	log.Println("Search (incl query embedding) took", time.Since(start))
	// Here you could filter out any documents whose similarity is below a certain threshold.
	// if docRes[...].Similarity < 0.5 { ...

	// Print the retrieved documents and their similarity to the question.
	buf := &strings.Builder{}
	for i, res := range docRes {
		content := strings.ReplaceAll(res.Content, "\n", " ")
		content = content[:min(100, len(content))] + "..."
		fmt.Fprintf(buf, "\t%d) Similarity %f:\n"+
			"\t\tURL: https://arxiv.org/abs/%s\n"+
			"\t\tSubmitter: %s\n"+
			"\t\tTitle: %s\n"+
			"\t\tAbstract: %s\n",
			i+1, res.Similarity, res.ID, res.Metadata["submitter"], res.Metadata["title"], content)
	}
	log.Printf("Search results:\n%s\n", buf.String())

	/* Output:
	2024/03/10 18:23:55 Setting up chromem-go...
	2024/03/10 18:23:55 Reading JSON lines...
	2024/03/10 18:23:55 Read and parsed 5006 documents.
	2024/03/10 18:23:55 Adding documents to chromem-go, including creating their embeddings via OpenAI API...
	2024/03/10 18:28:12 Querying chromem-go...
	2024/03/10 18:28:12 Search (incl query embedding) took 529.451163ms
	2024/03/10 18:28:12 Search results:
		1) Similarity 0.488895:
			URL: https://arxiv.org/abs/2209.15469
			Submitter: Christian Buck
			Title: Zero-Shot Retrieval with Search Agents and Hybrid Environments
			Abstract: Learning to search is the task of building artificial agents that learn to autonomously use a search...
		2) Similarity 0.480713:
			URL: https://arxiv.org/abs/2305.11516
			Submitter: Ryo Nagata Dr.
			Title: Contextualized Word Vector-based Methods for Discovering Semantic  Differences with No Training nor Word Alignment
			Abstract: In this paper, we propose methods for discovering semantic differences in words appearing in two cor...
		3) Similarity 0.476079:
			URL: https://arxiv.org/abs/2310.14025
			Submitter: Maria Lymperaiou
			Title: Large Language Models and Multimodal Retrieval for Visual Word Sense  Disambiguation
			Abstract: Visual Word Sense Disambiguation (VWSD) is a novel challenging task with the goal of retrieving an i...
		4) Similarity 0.474883:
			URL: https://arxiv.org/abs/2302.14785
			Submitter: Teven Le Scao
			Title: Joint Representations of Text and Knowledge Graphs for Retrieval and  Evaluation
			Abstract: A key feature of neural models is that they can produce semantic vector representations of objects (...
		5) Similarity 0.470326:
			URL: https://arxiv.org/abs/2309.02403
			Submitter: Dallas Card
			Title: Substitution-based Semantic Change Detection using Contextual Embeddings
			Abstract: Measuring semantic change has thus far remained a task where methods using contextual embeddings hav...
		6) Similarity 0.466851:
			URL: https://arxiv.org/abs/2309.08187
			Submitter: Vu Tran
			Title: Encoded Summarization: Summarizing Documents into Continuous Vector  Space for Legal Case Retrieval
			Abstract: We present our method for tackling a legal case retrieval task by introducing our method of encoding...
		7) Similarity 0.461783:
			URL: https://arxiv.org/abs/2307.16638
			Submitter: Maiia Bocharova Bocharova
			Title: VacancySBERT: the approach for representation of titles and skills for  semantic similarity search in the recruitment domain
			Abstract: The paper focuses on deep learning semantic search algorithms applied in the HR domain. The aim of t...
		8) Similarity 0.460481:
			URL: https://arxiv.org/abs/2106.07400
			Submitter: Clara Meister
			Title: Determinantal Beam Search
			Abstract: Beam search is a go-to strategy for decoding neural sequence models. The algorithm can naturally be ...
		9) Similarity 0.460001:
			URL: https://arxiv.org/abs/2305.04049
			Submitter: Yuxia Wu
			Title: Actively Discovering New Slots for Task-oriented Conversation
			Abstract: Existing task-oriented conversational search systems heavily rely on domain ontologies with pre-defi...
		10) Similarity 0.458321:
			URL: https://arxiv.org/abs/2305.08654
			Submitter: Taichi Aida
			Title: Unsupervised Semantic Variation Prediction using the Distribution of  Sibling Embeddings
			Abstract: Languages are dynamic entities, where the meanings associated with words constantly change with time...
	*/
}
