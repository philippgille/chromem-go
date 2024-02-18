Changelog
=========

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

vNext
-----

### Added

- Added `DB.ListCollections()` (PR [#12](https://github.com/philippgille/chromem-go/pull/12))
- Added `DB.GetCollection()` (PR [#13](https://github.com/philippgille/chromem-go/pull/13) + [#19](https://github.com/philippgille/chromem-go/pull/19))
- Added `DB.GetOrCreateCollection()` (PR [#22](https://github.com/philippgille/chromem-go/pull/22))
- Added `DB.DeleteCollection()` (PR [#14](https://github.com/philippgille/chromem-go/pull/14))
- Added `DB.Reset()` (PR [#15](https://github.com/philippgille/chromem-go/pull/15))
- Added various unit tests (PR [#20](https://github.com/philippgille/chromem-go/pull/20))

### Improved

- Improved example (PR [#11](https://github.com/philippgille/chromem-go/pull/11))
- Stop exporting `Collection.Metadata` (PR [#16](https://github.com/philippgille/chromem-go/pull/16))
  - Goal: Prevent direct modifications which could cause data races in case of the user doing a modification while `chromem-go` for example ranges over it during a `Collection.Query()` call.
- Copy metadata in constructors (PR [#17](https://github.com/philippgille/chromem-go/pull/17))
  - Goal: Prevent direct modifications which could cause data races in case of the user doing a modification while `chromem-go` for example ranges over it.
- Improved CI (PR [#18](https://github.com/philippgille/chromem-go/pull/18))
  - Add Go 1.22 to test matrix, update used GitHub Action from v4 to v5, use race detector during tests
- Reorganize code internally (PR [#21](https://github.com/philippgille/chromem-go/pull/21))

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
