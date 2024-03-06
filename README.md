# chromem-go

[![Go Reference](https://pkg.go.dev/badge/github.com/philippgille/chromem-go.svg)](https://pkg.go.dev/github.com/philippgille/chromem-go)
[![Build status](https://github.com/philippgille/chromem-go/actions/workflows/go.yml/badge.svg)](https://github.com/philippgille/chromem-go/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/philippgille/chromem-go)](https://goreportcard.com/report/github.com/philippgille/chromem-go)
[![GitHub Releases](https://img.shields.io/github/release/philippgille/chromem-go.svg)](https://github.com/philippgille/chromem-go/releases)

Embeddable vector database for Go with Chroma-like interface and zero third-party dependencies. In-memory with optional persistence.

It's *not* a library to connect to Chroma and also not a reimplementation of it in Go. It's a database on its own.

Because `chromem-go` is embeddable it enables you to add retrieval augmented generation (RAG) and similar embeddings-based features into your Go app *without having to run a separate database*. Like when using SQLite instead of PostgreSQL/MySQL/etc.

The focus is not scale or number of features, but simplicity.

> ⚠️ The project is in beta, under heavy construction, and may introduce breaking changes in releases before `v1.0.0`. All changes are documented in the [`CHANGELOG`](./CHANGELOG.md).

## Contents

1. [Use cases](#use-cases)
2. [Interface](#interface)
3. [Features](#features)
4. [Usage](#usage)
5. [Motivation](#motivation)
6. [Related projects](#related-projects)

## Use cases

With a vector database you can do various things:

- Retrieval augmented generation (RAG), question answering (Q&A)
- Text and code search
- Recommendation systems
- Classification
- Clustering

Let's look at the RAG use case in more detail:

### RAG

The knowledge of large language models (LLMs) - even the ones with with 30 billion, 70 billion paramters and more - is limited. They don't know anything about what happened after their training ended, they don't know anything about data they were not trained with (like your company's intranet, Jira / bug tracker, wiki or other kinds of knowledge bases), and even the data they *do* know they often can't reproduce it *exactly*, but start to *hallucinate* instead.

Fine-tuning an LLM can help a bit, but it's more meant to improve the LLMs reasoning about specific topics, or reproduce the style of written text or code. Fine-tuning does *not* add knowledge *1:1* into the model. Details are lost or mixed up. And knowledge cutoff (about anything that happened after the fine-tuning) isn't solved either.

=> A vector database can act as the the up-to-date, precise knowledge for LLMs:

1. You store relevant documents that you want the LLM to know in the database.
2. The database stores the *embeddings* alongside the documents, which you can either provide or can be created by specific "embedding models" like OpenAI's `text-embedding-3-small`.
   - `chromem-go` can do this for you and supports multiple embedding providers and models out-of-the-box.
3. Later, when you want to talk to the LLM, you first send the question to the vector DB to find *similar*/*related* content. This is called "nearest neighbor search".
4. In the question to the LLM, you provide this content alongside your question.
5. The LLM can take this up-to-date precise content into account when answering.

Check out the [example code](example/main.go) to see it in action!

## Interface

For the full interface see <https://pkg.go.dev/github.com/philippgille/chromem-go>.

Our inspiration was the [Chroma](https://www.trychroma.com/) interface, whose core API is the following (taken from their [README](https://github.com/chroma-core/chroma/blob/0.4.21/README.md)):

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

Our Go library exposes the same interface:

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

Initially `chromem-go` started with just the four core methods, but we added more over time. We intentionally don't want to cover 100% of Chroma's API surface though.  
We're providing some alternative methods that are more Go-idiomatic instead.

See the Godoc for details: <https://pkg.go.dev/github.com/philippgille/chromem-go>

## Features

- [X] Zero dependencies on third party libraries
- [X] Embeddable (like SQLite, i.e. no client-server model, no separate DB to maintain)
- [X] Multi-threaded processing (when adding and querying documents), making use of Go's native concurrency features
- Embedding creators:
  - Hosted:
    - [X] [OpenAI](https://platform.openai.com/docs/guides/embeddings/embedding-models) (default)
    - [X] [Mistral](https://docs.mistral.ai/platform/endpoints/#embedding-models)
    - [X] [Jina](https://jina.ai/embeddings)
    - [X] [mixedbread.ai](https://www.mixedbread.ai/)
  - Local:
    - [X] [Ollama](https://github.com/ollama/ollama)
    - [X] [LocalAI](https://github.com/mudler/LocalAI)
  - Bring your own (implement [`chromem.EmbeddingFunc`](https://pkg.go.dev/github.com/philippgille/chromem-go#EmbeddingFunc))
  - You can also pass existing embeddings when adding documents to a collection, instead of letting `chromem-go` create them
- Similarity search:
  - [X] Exhaustive nearest neighbor search using cosine similarity
    - A.k.a. "exact" or brute-force search. Sometimes called FLAT index.
- Filters:
  - [X] Document filters: `$contains`, `$not_contains`
  - [X] Metadata filters: Exact matches
- Storage:
  - [X] In-memory
  - [X] Optional local persistence (file based, encoded as [gob](https://go.dev/blog/gob))
- Data types:
  - [X] Documents (text)

### Roadmap

- Performance:
  - [ ] Add Go benchmark code
  - [ ] Improve code based on CPU and memory profiles
- Embedding creators:
  - [ ] Add an `EmbeddingFunc` that downloads and shells out to [llamafile](https://github.com/Mozilla-Ocho/llamafile)
- Similarity search:
  - [ ] Approximate nearest neighbor search with index (ANN)
    - [ ] Hierarchical Navigable Small World (HNSW)
    - [ ] Inverted file flat (IVFFlat)
- Filters:
  - [ ] Operators (`$and`, `$or` etc.)
- Storage:
  - [ ] JSON as second encoding format
  - [ ] Write-ahead log (WAL) as second file format
  - [ ] Compression
  - [ ] Encryption (at rest)
  - [ ] Optional remote storage (S3, PostgreSQL, ...)
- Data types:
  - [ ] Images
  - [ ] Videos

## Usage

For a full, working example, using the vector database for retrieval augmented generation (RAG), see [example/main.go](example/main.go)

## Motivation

In December 2023, when I wanted to play around with retrieval augmented generation (RAG) in a Go program, I looked for a vector database that could be embedded in the Go program, just like you would embed SQLite in order to not require any separate DB setup and maintenance. I was surprised when I didn't find any, given the abundance of embedded key-value stores in the Go ecosystem.

At the time most of the popular vector databases like Pinecone, Qdrant, Milvus, Weaviate and others were not embeddable at all. ChromaDB was, but only in Python.

Then I found [@eliben](https://github.com/eliben)'s [blog post](https://eli.thegreenplace.net/2023/retrieval-augmented-generation-in-go/) and [example code](https://github.com/eliben/code-for-blog/tree/eda87b87dad9ed8bd45d1c8d6395efba3741ed39/2023/go-rag-openai) which showed that with very little Go code you could create a very basic PoC of a vector database.

That's when I decided to build my own vector database, embeddable in Go, inspired by the ChromaDB interface. ChromaDB stood out for being embeddable (in Python), and by showing its core API in 4 commands on their README and on the landing page of their website.

## Related projects

- Shoutout to [@eliben](https://github.com/eliben) whose [blog post](https://eli.thegreenplace.net/2023/retrieval-augmented-generation-in-go/) and [example code](https://github.com/eliben/code-for-blog/tree/eda87b87dad9ed8bd45d1c8d6395efba3741ed39/2023/go-rag-openai) inspired me to start this project!
- [Chroma](https://github.com/chroma-core/chroma): Looking at Pinecone, Qdrant, Milvus, Weaviate and others, Chroma stood out by showing its core API in 4 commands on their README and on the landing page of their website. It was also the only one which could be embedded (in Python).
- The big, full-fledged client-server-based vector databases for maximum scale and performance:
  - [Pinecone](https://www.pinecone.io/): Closed source
  - [Qdrant](https://github.com/qdrant/qdrant): Written in Rust
  - [Milvus](https://github.com/milvus-io/milvus): Written in Go and C++, but not embeddable as of December 2023
  - [Weaviate](https://github.com/weaviate/weaviate): Written in Go, but not embeddable as of December 2023
- Some non-specialized SQL, NoSQL and Key-Value databases added support for storing vectors and (some of them) querying based on similarity:
  - [`pgvector`](https://github.com/pgvector/pgvector) extension for PostgreSQL: Client-server model
  - [Redis](https://github.com/redis/redis) ([1](https://redis.io/docs/interact/search-and-query/query/vector-search/), [2](https://redis.io/docs/interact/search-and-query/advanced-concepts/vectors/)): Client-server model
  - [DuckDB](https://github.com/duckdb/duckdb) has a function to calculate cosine similarity ([1](https://duckdb.org/docs/sql/functions/nested)): Can be embedded (like SQLite), but the Go library is CGO-based and there's no way to query with a text input and get the most similar stored text as output (as of 2024-03-06)
  - [MongoDB](https://github.com/mongodb/mongo)'s cloud platform offers a vector search product ([1](https://www.mongodb.com/products/platform/atlas-vector-search)): Client-server model
- Some libraries for vector similarity search:
  - [Faiss](https://github.com/facebookresearch/faiss): Written in C++; 3rd party Go bindings use CGO
  - [Annoy](https://github.com/spotify/annoy): Written in C++; Go bindings use CGO ([1](https://github.com/spotify/annoy/blob/2be37c9e015544be2cf60c431f0cccc076151a2d/README_GO.rst))
  - [USearch](https://github.com/unum-cloud/usearch): Written in C++; Go bindings use CGO
