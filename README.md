# chromem-go

[![Go Reference](https://pkg.go.dev/badge/github.com/philippgille/chromem-go.svg)](https://pkg.go.dev/github.com/philippgille/chromem-go)
[![Build status](https://github.com/philippgille/chromem-go/actions/workflows/go.yml/badge.svg)](https://github.com/philippgille/chromem-go/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/philippgille/chromem-go)](https://goreportcard.com/report/github.com/philippgille/chromem-go)
[![GitHub Releases](https://img.shields.io/github/release/philippgille/chromem-go.svg)](https://github.com/philippgille/chromem-go/releases)

Embeddable vector database for Go with Chroma-like interface and zero third-party dependencies. In-memory with optional persistence.

Because `chromem-go` is embeddable it enables you to add retrieval augmented generation (RAG) and similar embeddings-based features into your Go app *without having to run a separate database*. Like when using SQLite instead of PostgreSQL/MySQL/etc.

It's *not* a library to connect to Chroma and also not a reimplementation of it in Go. It's a database on its own.

The focus is not scale (millions of documents) or number of features, but simplicity and performance for the most common use cases. On a mid-range 2020 Intel laptop CPU you can query 1,000 documents in 0.3 ms and 100,000 documents in 40 ms, with very few and small memory allocations. See [Benchmarks](#benchmarks) for details.

> ⚠️ The project is in beta, under heavy construction, and may introduce breaking changes in releases before `v1.0.0`. All changes are documented in the [`CHANGELOG`](./CHANGELOG.md).

## Contents

1. [Use cases](#use-cases)
2. [Interface](#interface)
3. [Features + Roadmap](#features)
4. [Installation](#installation)
5. [Usage](#usage)
6. [Benchmarks](#benchmarks)
7. [Development](#development)
8. [Motivation](#motivation)
9. [Related projects](#related-projects)

## Use cases

With a vector database you can do various things:

- Retrieval augmented generation (RAG), question answering (Q&A)
- Text and code search
- Recommendation systems
- Classification
- Clustering

Let's look at the RAG use case in more detail:

### RAG

The knowledge of large language models (LLMs) - even the ones with 30 billion, 70 billion parameters and more - is limited. They don't know anything about what happened after their training ended, they don't know anything about data they were not trained with (like your company's intranet, Jira / bug tracker, wiki or other kinds of knowledge bases), and even the data they *do* know they often can't reproduce it *exactly*, but start to *hallucinate* instead.

Fine-tuning an LLM can help a bit, but it's more meant to improve the LLMs reasoning about specific topics, or reproduce the style of written text or code. Fine-tuning does *not* add knowledge *1:1* into the model. Details are lost or mixed up. And knowledge cutoff (about anything that happened after the fine-tuning) isn't solved either.

=> A vector database can act as the up-to-date, precise knowledge for LLMs:

1. You store relevant documents that you want the LLM to know in the database.
2. The database stores the *embeddings* alongside the documents, which you can either provide or can be created by specific "embedding models" like OpenAI's `text-embedding-3-small`.
   - `chromem-go` can do this for you and supports multiple embedding providers and models out-of-the-box.
3. Later, when you want to talk to the LLM, you first send the question to the vector DB to find *similar*/*related* content. This is called "nearest neighbor search".
4. In the question to the LLM, you provide this content alongside your question.
5. The LLM can take this up-to-date precise content into account when answering.

Check out the [example code](examples) to see it in action!

## Interface

Our original inspiration was the [Chroma](https://www.trychroma.com/) interface, whose core API is the following (taken from their [README](https://github.com/chroma-core/chroma/blob/0.4.21/README.md)):

<details><summary>Chroma core interface</summary>

```python
import chromadb
# setup Chroma in-memory, for easy prototyping. Can add persistence easily!
client = chromadb.Client()

# Create collection. get_collection, get_or_create_collection, delete_collection also available!
collection = client.create_collection("all-my-documents")

# Add docs to the collection. Can also update and delete. Row-based API coming soon!
collection.add(
    documents=["This is document1", "This is document2"], # we handle tokenization, embedding, and indexing automatically. You can skip that and add your own embeddings as well
    metadatas=[{"source": "notion"}, {"source": "google-docs"}], # filter on these!
    ids=["doc1", "doc2"], # unique for each doc
)

# Query/search 2 most similar results. You can also .get by id
results = collection.query(
    query_texts=["This is a query document"],
    n_results=2,
    # where={"metadata_field": "is_equal_to_this"}, # optional filter
    # where_document={"$contains":"search_string"}  # optional filter
)
```

</details>

Our Go library exposes the same interface:

<details><summary>chromem-go equivalent</summary>

```go
package main

import "github.com/philippgille/chromem-go"

func main() {
    // Set up chromem-go in-memory, for easy prototyping. Can add persistence easily!
    // We call it DB instead of client because there's no client-server separation. The DB is embedded.
    db := chromem.NewDB()

    // Create collection. GetCollection, GetOrCreateCollection, DeleteCollection also available!
    collection, _ := db.CreateCollection("all-my-documents", nil, nil)

    // Add docs to the collection. Update and delete will be added in the future.
    // Can be multi-threaded with AddConcurrently()!
    // We're showing the Chroma-like method here, but more Go-idiomatic methods are also available!
    _ = collection.Add(ctx,
        []string{"doc1", "doc2"}, // unique ID for each doc
        nil, // We handle embedding automatically. You can skip that and add your own embeddings as well.
        []map[string]string{{"source": "notion"}, {"source": "google-docs"}}, // Filter on these!
        []string{"This is document1", "This is document2"},
    )

    // Query/search 2 most similar results. Getting by ID will be added in the future.
    results, _ := collection.Query(ctx,
        "This is a query document",
        2,
        map[string]string{"metadata_field": "is_equal_to_this"}, // optional filter
        map[string]string{"$contains": "search_string"},         // optional filter
    )
}
```

</details>

Initially `chromem-go` started with just the four core methods, but we added more over time. We intentionally don't want to cover 100% of Chroma's API surface though.  
We're providing some alternative methods that are more Go-idiomatic instead.

For the full interface see the Godoc: <https://pkg.go.dev/github.com/philippgille/chromem-go>

## Features

- [X] Zero dependencies on third party libraries
- [X] Embeddable (like SQLite, i.e. no client-server model, no separate DB to maintain)
- [X] Multithreaded processing (when adding and querying documents), making use of Go's native concurrency features
- [X] Experimental WebAssembly binding
- Embedding creators:
  - Hosted:
    - [X] [OpenAI](https://platform.openai.com/docs/guides/embeddings/embedding-models) (default)
    - [X] [Azure OpenAI](https://learn.microsoft.com/en-us/azure/ai-services/openai/how-to/embeddings)
    - [X] [Cohere](https://cohere.com/models/embed)
    - [X] [Mistral](https://docs.mistral.ai/platform/endpoints/#embedding-models)
    - [X] [Jina](https://jina.ai/embeddings)
    - [X] [mixedbread.ai](https://www.mixedbread.ai/)
  - Local:
    - [X] [Ollama](https://github.com/ollama/ollama)
    - [X] [LocalAI](https://github.com/mudler/LocalAI)
  - Bring your own (implement [`chromem.EmbeddingFunc`](https://pkg.go.dev/github.com/philippgille/chromem-go#EmbeddingFunc))
  - You can also pass existing embeddings when adding documents to a collection, instead of letting `chromem-go` create them
- Similarity search:
  - [X] Exhaustive nearest neighbor search using cosine similarity (sometimes also called exact search or brute-force search or FLAT index)
- Filters:
  - [X] Document filters: `$contains`, `$not_contains`
  - [X] Metadata filters: Exact matches
- Storage:
  - [X] In-memory
  - [X] Optional immediate persistence (writes one file for each added collection and document, encoded as [gob](https://go.dev/blog/gob), optionally gzip-compressed)
  - [X] Backups: Export and import of the entire DB to/from a single file (encoded as [gob](https://go.dev/blog/gob), optionally gzip-compressed and AES-GCM encrypted)
    - Includes methods for generic `io.Writer`/`io.Reader` so you can plug S3 buckets and other blob storage, see [examples/s3-export-import](examples/s3-export-import) for example code
- Data types:
  - [X] Documents (text)

### Roadmap

- Performance:
  - Use SIMD for dot product calculation on supported CPUs (draft PR: [#48](https://github.com/philippgille/chromem-go/pull/48))
  - Add [roaring bitmaps](https://github.com/RoaringBitmap/roaring) to speed up full text filtering
- Embedding creators:
  - Add an `EmbeddingFunc` that downloads and shells out to [llamafile](https://github.com/Mozilla-Ocho/llamafile)
- Similarity search:
  - Approximate nearest neighbor search with index (ANN)
    - Hierarchical Navigable Small World (HNSW)
    - Inverted file flat (IVFFlat)
- Filters:
  - Operators (`$and`, `$or` etc.)
- Storage:
  - JSON as second encoding format
  - Write-ahead log (WAL) as second file format
  - Optional remote storage (S3, PostgreSQL, ...)
- Data types:
  - Images
  - Videos

## Installation

`go get github.com/philippgille/chromem-go@latest`

## Usage

See the Godoc for a reference: <https://pkg.go.dev/github.com/philippgille/chromem-go>

For full, working examples, using the vector database for retrieval augmented generation (RAG) and semantic search and using either OpenAI or locally running the embeddings model and LLM (in Ollama), see the [example code](examples).

### Quickstart

This is taken from the ["minimal" example](examples/minimal):

```go
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

  err = c.AddDocuments(ctx, []chromem.Document{
    {
      ID:      "1",
      Content: "The sky is blue because of Rayleigh scattering.",
    },
    {
      ID:      "2",
      Content: "Leaves are green because chlorophyll absorbs red and blue light.",
    },
  }, runtime.NumCPU())
  if err != nil {
    panic(err)
  }

  res, err := c.Query(ctx, "Why is the sky blue?", 1, nil, nil)
  if err != nil {
    panic(err)
  }

  fmt.Printf("ID: %v\nSimilarity: %v\nContent: %v\n", res[0].ID, res[0].Similarity, res[0].Content)
}
```

Output:

```text
ID: 1
Similarity: 0.6833369
Content: The sky is blue because of Rayleigh scattering.
```

## Benchmarks

Benchmarked on 2024-03-17 with:

- Computer: Framework Laptop 13 (first generation, 2021)
- CPU: 11th Gen Intel Core i5-1135G7 (2020)
- Memory: 32 GB
- OS: Fedora Linux 39
  - Kernel: 6.7

```console
$ go test -benchmem -run=^$ -bench .
goos: linux
goarch: amd64
pkg: github.com/philippgille/chromem-go
cpu: 11th Gen Intel(R) Core(TM) i5-1135G7 @ 2.40GHz
BenchmarkCollection_Query_NoContent_100-8          13164      90276 ns/op     5176 B/op       95 allocs/op
BenchmarkCollection_Query_NoContent_1000-8          2142     520261 ns/op    13558 B/op      141 allocs/op
BenchmarkCollection_Query_NoContent_5000-8           561    2150354 ns/op    47096 B/op      173 allocs/op
BenchmarkCollection_Query_NoContent_25000-8          120    9890177 ns/op   211783 B/op      208 allocs/op
BenchmarkCollection_Query_NoContent_100000-8          30   39574238 ns/op   810370 B/op      232 allocs/op
BenchmarkCollection_Query_100-8                    13225      91058 ns/op     5177 B/op       95 allocs/op
BenchmarkCollection_Query_1000-8                    2226     519693 ns/op    13552 B/op      140 allocs/op
BenchmarkCollection_Query_5000-8                     550    2128121 ns/op    47108 B/op      173 allocs/op
BenchmarkCollection_Query_25000-8                    100   10063260 ns/op   211705 B/op      205 allocs/op
BenchmarkCollection_Query_100000-8                    30   39404005 ns/op   810295 B/op      229 allocs/op
PASS
ok   github.com/philippgille/chromem-go 28.402s
```

## Development

- Build: `go build ./...`
- Test: `go test -v -race -count 1 ./...`
- Benchmark:
  - `go test -benchmem -run=^$ -bench .` (add `> bench.out` or similar to write to a file)
  - With profiling: `go test -benchmem -run ^$ -cpuprofile cpu.out -bench .`
    - (profiles: `-cpuprofile`, `-memprofile`, `-blockprofile`, `-mutexprofile`)
- Compare benchmarks:
  1. Install `benchstat`: `go install golang.org/x/perf/cmd/benchstat@latest`
  2. Compare two benchmark results: `benchstat before.out after.out`

## Motivation

In December 2023, when I wanted to play around with retrieval augmented generation (RAG) in a Go program, I looked for a vector database that could be embedded in the Go program, just like you would embed SQLite in order to not require any separate DB setup and maintenance. I was surprised when I didn't find any, given the abundance of embedded key-value stores in the Go ecosystem.

At the time most of the popular vector databases like Pinecone, Qdrant, Milvus, Chroma, Weaviate and others were not embeddable at all or only in Python or JavaScript/TypeScript.

Then I found [@eliben](https://github.com/eliben)'s [blog post](https://eli.thegreenplace.net/2023/retrieval-augmented-generation-in-go/) and [example code](https://github.com/eliben/code-for-blog/tree/eda87b87dad9ed8bd45d1c8d6395efba3741ed39/2023/go-rag-openai) which showed that with very little Go code you could create a very basic PoC of a vector database.

That's when I decided to build my own vector database, embeddable in Go, inspired by the ChromaDB interface. ChromaDB stood out for being embeddable (in Python), and by showing its core API in 4 commands on their README and on the landing page of their website.

## Related projects

- Shoutout to [@eliben](https://github.com/eliben) whose [blog post](https://eli.thegreenplace.net/2023/retrieval-augmented-generation-in-go/) and [example code](https://github.com/eliben/code-for-blog/tree/eda87b87dad9ed8bd45d1c8d6395efba3741ed39/2023/go-rag-openai) inspired me to start this project!
- [Chroma](https://github.com/chroma-core/chroma): Looking at Pinecone, Qdrant, Milvus, Weaviate and others, Chroma stood out by showing its core API in 4 commands on their README and on the landing page of their website. It was also putting the most emphasis on its embeddability (in Python).
- The big, full-fledged client-server-based vector databases for maximum scale and performance:
  - [Pinecone](https://www.pinecone.io/): Closed source
  - [Qdrant](https://github.com/qdrant/qdrant): Written in Rust, not embeddable in Go
  - [Milvus](https://github.com/milvus-io/milvus): Written in Go and C++, but not embeddable as of December 2023
  - [Weaviate](https://github.com/weaviate/weaviate): Written in Go, but not embeddable in Go as of March 2024 (only in Python and JavaScript/TypeScript and that's experimental)
- Some non-specialized SQL, NoSQL and Key-Value databases added support for storing vectors and (some of them) querying based on similarity:
  - [pgvector](https://github.com/pgvector/pgvector) extension for [PostgreSQL](https://www.postgresql.org/): Client-server model
  - [Redis](https://github.com/redis/redis) ([1](https://redis.io/docs/interact/search-and-query/query/vector-search/), [2](https://redis.io/docs/interact/search-and-query/advanced-concepts/vectors/)): Client-server model
  - [sqlite-vss](https://github.com/asg017/sqlite-vss) extension for [SQLite](https://www.sqlite.org/): Embedded, but the [Go bindings](https://github.com/asg017/sqlite-vss/tree/8fc44301843029a13a474d1f292378485e1fdd62/bindings/go) require CGO. There's a [CGO-free Go library](https://gitlab.com/cznic/sqlite) for SQLite, but then it's without the vector search extension.
  - [DuckDB](https://github.com/duckdb/duckdb) has a function to calculate cosine similarity ([1](https://duckdb.org/docs/sql/functions/nested)): Embedded, but the Go bindings use CGO
  - [MongoDB](https://github.com/mongodb/mongo)'s cloud platform offers a vector search product ([1](https://www.mongodb.com/products/platform/atlas-vector-search)): Client-server model
- Some libraries for vector similarity search:
  - [Faiss](https://github.com/facebookresearch/faiss): Written in C++; 3rd party Go bindings use CGO
  - [Annoy](https://github.com/spotify/annoy): Written in C++; Go bindings use CGO ([1](https://github.com/spotify/annoy/blob/2be37c9e015544be2cf60c431f0cccc076151a2d/README_GO.rst))
  - [USearch](https://github.com/unum-cloud/usearch): Written in C++; Go bindings use CGO
- Some orchestration libraries, inspired by the Python library [LangChain](https://github.com/langchain-ai/langchain), but with no or only rudimentary embedded vector DB:
  - [LangChain Go](https://github.com/tmc/langchaingo)
  - [LinGoose](https://github.com/henomis/lingoose)
  - [GoLC](https://github.com/hupe1980/golc)
