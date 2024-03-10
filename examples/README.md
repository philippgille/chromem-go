# Examples

1. [RAG Wikipedia Ollama](rag-wikipedia-ollama)
   - This example shows a retrieval augmented generation (RAG) application, using `chromem-go` as knowledge base for finding relevant info for a question. More specifically the app is doing *question answering*.
   - The underlying data is 200 Wikipedia articles (or rather their lead section / introduction).
   - We run the embeddings model and LLM in [Ollama](https://github.com/ollama/ollama), to showcase how a RAG application can run entirely offline, without relying on OpenAI or other third party APIs.
2. [Semantic search arXiv OpenAI](semantic-search-arxiv-openai)
   - This example shows a semantic search application, using `chromem-go` as vector database for finding semantically relevant search results.
   - We load and search across ~5,000 arXiv papers in the "Computer Science - Computation and Language" category, which is the relevant one for Natural Language Processing (NLP) related papers.
