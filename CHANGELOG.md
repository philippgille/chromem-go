Changelog
=========

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

vNext
-----

Highlights in this release are query performance improvements (5x faster, 98% fewer memory allocations), an extended interface, and a new code example for semantic search across 5,000 arXiv papers.

### Added

- Added arXiv semantic search example (PR [#45](https://github.com/philippgille/chromem-go/pull/45))
- Added basic query benchmark (PR [#46](https://github.com/philippgille/chromem-go/pull/46))
- Added unit test for collection query errors (PR [#51](https://github.com/philippgille/chromem-go/pull/51))
- Added `Collection.QueryEmbedding()` method for when you already have the embedding of your query (PR [#52](https://github.com/philippgille/chromem-go/pull/52))

### Improved

- Changed the example link target to directory instead of `main.go` file (PR [#43](https://github.com/philippgille/chromem-go/pull/43))
- Improved query performance (5x faster, 98% fewer memory allocations) (PR [#47](https://github.com/philippgille/chromem-go/pull/47), [#53](https://github.com/philippgille/chromem-go/pull/53), [#54](https://github.com/philippgille/chromem-go/pull/54))
- Extended parameter validation (PR [#50](https://github.com/philippgille/chromem-go/pull/50), [#51](https://github.com/philippgille/chromem-go/pull/51))

### Fixed

- Fixed path joining (PR [#44](https://github.com/philippgille/chromem-go/pull/44))

### Breaking changes

- Due to vectors now being normalized at the time of adding the document to the collection instead of when querying, the persisted data from prior versions is incompatible with this version (PR [#47](https://github.com/philippgille/chromem-go/pull/47))

v0.4.0 (2024-03-06)
-------------------

Highlights in this release are optional persistence, an extended interface, support for creating embeddings with [Ollama](https://github.com/ollama/ollama/), the exporting of the `Document` struct, and more Go-idiomatic methods to add documents to collections.

### Added

- Extended the interface:
  - `DB.ListCollections()` (PR [#12](https://github.com/philippgille/chromem-go/pull/12))
  - `DB.GetCollection()` (PR [#13](https://github.com/philippgille/chromem-go/pull/13) + [#19](https://github.com/philippgille/chromem-go/pull/19))
  - `DB.DeleteCollection()` (PR [#14](https://github.com/philippgille/chromem-go/pull/14))
  - `DB.Reset()` (PR [#15](https://github.com/philippgille/chromem-go/pull/15))
  - `DB.GetOrCreateCollection()` (PR [#22](https://github.com/philippgille/chromem-go/pull/22))
  - `Collection.Count()` (PR [#27](https://github.com/philippgille/chromem-go/pull/27))
  - `Document` struct, `NewDocument()` function, `Collection.AddDocument()` and `Collection.AddDocuments()` methods (PR [#34](https://github.com/philippgille/chromem-go/pull/34))
    - More Go-idiomatic alternatives to `Collection.Add()`
- Added various unit tests (PR [#20](https://github.com/philippgille/chromem-go/pull/20), [#39](https://github.com/philippgille/chromem-go/pull/39))
- Added optional persistence! Via multiple PRs:
  - Write in PR [#23](https://github.com/philippgille/chromem-go/pull/23), [#24](https://github.com/philippgille/chromem-go/pull/24), [#31](https://github.com/philippgille/chromem-go/pull/31)
  - Read in PR [#25](https://github.com/philippgille/chromem-go/pull/25)
  - Delete in PR [#26](https://github.com/philippgille/chromem-go/pull/26)
- Added support for creating embeddings with [Ollama](https://github.com/ollama/ollama/) (PR [#32](https://github.com/philippgille/chromem-go/pull/32))
- Added example documentation (PR [#42](https://github.com/philippgille/chromem-go/pull/42))

### Improved

- Improved example (PR [#11](https://github.com/philippgille/chromem-go/pull/11), [#28](https://github.com/philippgille/chromem-go/pull/28), [#33](https://github.com/philippgille/chromem-go/pull/33))
- Stop exporting `Collection.Metadata` (PR [#16](https://github.com/philippgille/chromem-go/pull/16))
  - Goal: Prevent direct modifications which could cause data races in case of the user doing a modification while `chromem-go` for example ranges over it during a `Collection.Query()` call.
- Copy metadata in constructors (PR [#17](https://github.com/philippgille/chromem-go/pull/17))
  - Goal: Prevent direct modifications which could cause data races in case of the user doing a modification while `chromem-go` for example ranges over it.
- Improved CI (PR [#18](https://github.com/philippgille/chromem-go/pull/18))
  - Add Go 1.22 to test matrix, update used GitHub Action from v4 to v5, use race detector during tests
- Reorganize code internally (PR [#21](https://github.com/philippgille/chromem-go/pull/21))
- Switched to newer recommended check for file related `ErrNotExist` errors (PR [#29](https://github.com/philippgille/chromem-go/pull/29))
- Added more validations in several existing methods (PR [#30](https://github.com/philippgille/chromem-go/pull/30))
- Internal variable renamed (PR [#37](https://github.com/philippgille/chromem-go/pull/37))
- Fail unit tests immediately (PR [#40](https://github.com/philippgille/chromem-go/pull/40))

### Fixed

- Fixed metadatas validation in `Collection.AddConcurrently()` (PR [#35](https://github.com/philippgille/chromem-go/pull/35))
- Fixed Godoc of `Collection.Query()` method (PR [#36](https://github.com/philippgille/chromem-go/pull/36))
- Fixed length of result slice (PR [#38](https://github.com/philippgille/chromem-go/pull/38))
- Fixed filter test (PR [#41](https://github.com/philippgille/chromem-go/pull/41))

### Breaking changes

- Because functions can't be (de-)serialized, `GetCollection` requires a new parameter of type `EmbeddingFunc`, in order to set the correct func when using a DB with persistence and it just loaded the collections and documents from storage. (PR [#25](https://github.com/philippgille/chromem-go/pull/25))
- Some methods now return an error (due to file operations when persistence is used)
- Additional validations will return an early error, but most (if not all) prior calls with the invalid parameters probably lead to some errors down the line anyway
- `Collection.Metadata` is not exported anymore
- `Result.Document` field was renamed to `Result.Content`, to avoid confusion with the now exported `Document` struct

v0.3.0 (2024-02-10)
-------------------

### Added

- Added support for more OpenAI embedding models (PR [#6](https://github.com/philippgille/chromem-go/pull/6))
- Added support for more embedding creators/providers: (PR [#10](https://github.com/philippgille/chromem-go/pull/10))
  - [Mistral](https://docs.mistral.ai/platform/endpoints/#embedding-models), [Jina](https://jina.ai/embeddings), [mixedbread.ai](https://www.mixedbread.ai/), [LocalAI](https://github.com/mudler/LocalAI)

### Improved

- Improve concurrency when adding documents to collection (PR [#2](https://github.com/philippgille/chromem-go/pull/2))
- Rename `Client` to `DB` to better indicate that the database is embedded and there's no client-server separation (PR [#3](https://github.com/philippgille/chromem-go/pull/3))
- Change OpenAPI embedding model from "text-embedding-ada-002" to "text-embedding-3-small" (PR [#4](https://github.com/philippgille/chromem-go/pull/4))
- Allow custom base URL for OpenAI, enabling the use of [Azure OpenAI](https://azure.microsoft.com/en-us/products/ai-services/openai-service), [LiteLLM](https://github.com/BerriAI/litellm), [ollama](https://github.com/ollama/ollama/blob/main/docs/openai.md) etc. (PR [#7](https://github.com/philippgille/chromem-go/pull/7))
- Renamed `EmbeddingFunc` constructors to follow best practice (PR [#9](https://github.com/philippgille/chromem-go/pull/9))

### Fixed

- Don't allow `nResults` arg < 0 (PR [#5](https://github.com/philippgille/chromem-go/pull/5))

### Breaking changes

- Several function names and signatures were changed in this release. This can happen as long as the version is at `v0.x.y`.

v0.2.0 (2024-01-01)
-------------------

### Added

- Added GitHub Actions config ([commit](https://github.com/philippgille/chromem-go/commits/fae84f2069ec28bbf9f4e30dca569f447d6aee6a))
- Added `CHANGELOG.md` ([commit](https://github.com/philippgille/chromem-go/commits/bb0aa24b95ed19a743b2b5aa60098077bebdea41))
- Exported embedding creation functions ([commit](https://github.com/philippgille/chromem-go/commits/9d8ce4ae88c08bc975a0ed6b180bc01dcb2a390f))
- Added `Collection.AddConcurrently` to add embeddings concurrently ([commit](https://github.com/philippgille/chromem-go/commits/50fe3b743696d0209d2e4c617633ba335870ab7d))

### Improved

- Improved example code ([commit](https://github.com/philippgille/chromem-go/commits/c6437611d2fd48c5458b1932d5df62f90501981f))
- Removed unused field in `Client` ([commit](https://github.com/philippgille/chromem-go/commits/9c8b01ad386008d09675a26b7eca9c9605af5b1c))
- Improved validation in `Query` method ([commit](https://github.com/philippgille/chromem-go/commits/0bd196ee7c36164fad123c7b21766c7444de246d))
- Added and improved Godoc ([commit](https://github.com/philippgille/chromem-go/commits/c3a4db9563efb270af5aee585a7fca54b2ab08dc))
- Improved locking around a collection's documents ([commit](https://github.com/philippgille/chromem-go/commits/cefec66912d2fc96928154a9159ca05bd52c5149))
- Removed dependency on third party library for OpenAI ([commit](https://github.com/philippgille/chromem-go/commits/1a28e1b89808cb223d67808a8e90bb9c36d2d801))
- Parallelized document querying (PR [#1](https://github.com/philippgille/chromem-go/pull/1))

v0.1.0 (2023-12-29)
-------------------

Initial release with a minimal Chroma-like interface and a working retrieval augmented generation (RAG) example.
