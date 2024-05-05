# Examples

1. [Minimal example](minimal)
   - A minimal example with the least amount of code and no comments
   - Uses OpenAI for creating the embeddings
2. [RAG Wikipedia Ollama](rag-wikipedia-ollama)
   - This example shows a retrieval augmented generation (RAG) application, using `chromem-go` as knowledge base for finding relevant info for a question. More specifically the app is doing *question answering*.
   - The underlying data is 200 Wikipedia articles (or rather their lead section / introduction).
   - Runs the embeddings model and LLM in [Ollama](https://github.com/ollama/ollama), to showcase how a RAG application can run entirely offline, without relying on OpenAI or other third party APIs.
3. [Semantic search arXiv OpenAI](semantic-search-arxiv-openai)
   - This example shows a semantic search application, using `chromem-go` as vector database for finding semantically relevant search results.
   - Loads and searches across ~5,000 arXiv papers in the "Computer Science - Computation and Language" category, which is the relevant one for Natural Language Processing (NLP) related papers.
   - Uses OpenAI for creating the embeddings
4. [WebAssembly](webassembly)
   - This example shows how `chromem-go` can be compiled to WebAssembly and then used from JavaScript in a browser
5. [S3 Export/Import](s3-export-import)
   - This example shows how to export the DB to and import it from any S3-compatible blob storage service
