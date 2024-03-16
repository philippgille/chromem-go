# Semantic search arXiv OpenAI

This example shows a semantic search application, using `chromem-go` as vector database for finding semantically relevant search results. We load and search across ~5,000 arXiv papers in the "Computer Science - Computation and Language" category, which is the relevant one for Natural Language Processing (NLP) related papers.

This is not a retrieval augmented generation (RAG) app, because after *retrieving* the semantically relevant results, we don't *augment* any prompt to an LLM. No LLM is generates the final output.

## How to run

1. Prepare the dataset
   1. Download `arxiv-metadata-oai-snapshot.json` from <https://www.kaggle.com/datasets/Cornell-University/arxiv>
   2. Filter by "Computer Science - Computation and Language" category (see [taxonomy](https://arxiv.org/category_taxonomy)), filter by updates from 2023
      1. Ensure you have [ripgrep](https://github.com/BurntSushi/ripgrep) installed, or adapt the following commands to use grep
      2. Run `rg '"categories":"cs.CL"' ~/Downloads/arxiv-metadata-oai-snapshot.json | rg '"update_date":"2023' > /tmp/arxiv_cs-cl_2023.jsonl` (adapt input file path if necessary)
   3. Check the data
      1. `wc -l /tmp/arxiv_cs-cl_2023.jsonl` should show ~5,000 lines
      2. `du -h /tmp/arxiv_cs-cl_2023.jsonl` should show ~8.8 MB
2. Set the OpenAI API key in your env as `OPENAI_API_KEY`
3. Run the example: `go run .`

## Output

The output can differ slightly on each run, but it's along the lines of:

```log
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
```

The majority of the time here is spent during the embeddings creation, where we are limited by the performance of the OpenAI API.
